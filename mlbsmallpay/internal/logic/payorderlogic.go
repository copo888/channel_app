package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/mlbsmallpay/internal/payutils"
	"github.com/copo888/channel_app/mlbsmallpay/internal/service"
	"github.com/copo888/channel_app/mlbsmallpay/internal/svc"
	"github.com/copo888/channel_app/mlbsmallpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"
)

type PayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	if (strings.EqualFold(req.PayType, "YK") || strings.EqualFold(req.PayType, "YL")) && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	datetime := time.Now().Format("2006-01-02 15:04:05")
	unix := time.Now().Unix()
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo string `json:"merchantNo"`
		OrderNo    string `json:"orderNo"`
		UserNo     string `json:"userNo"`
		UserName   string `json:"userName"`
		ChannelNo  string `json:"channelNo"`
		Amount     string `json:"amount"`
		Discount   string `json:"discount"`
		PayeeName  string `json:"payeeName"`
		Extra      string `json:"extra"`
		Datetime   string `json:"datetime"`
		NotifyUrl  string `json:"notifyUrl"`
		ReturnUrl  string `json:"returnUrl"`
		Time       int64  `json:"time"`
		AppSecret  string `json:"appSecret"`
		Sign       string `json:"sign"`
	}{
		MerchantNo: channel.MerId,
		OrderNo:    req.OrderNo,
		ChannelNo:  req.ChannelPayType,
		UserName:   req.UserId,
		Amount:     req.TransactionAmount,
		PayeeName:  req.UserId,
		Datetime:   datetime,
		NotifyUrl:  notifyUrl,
		ReturnUrl:  req.PageUrl,
		Time:       unix,
		AppSecret:  l.svcCtx.Config.AppSecret,
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code                   int64   `json:"code"`
		Text                   string  `json:"text"`
		TradeNo                string  `json:"tradeNo"`
		OrderNo                string  `json:"orderNo"`
		TargetUrl              string  `json:"targetUrl"`
		Payable                float64 `json:"payable"`
		PayeeTitle             string  `json:"payeeTitle"`
		PayeeBankName          string  `json:"payeeBankName"`
		PayeeAccountNumber     string  `json:"payeeAccountNumber"`
		PayeeAccountBankBranch string  `json:"payeeAccountBankBranch"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Text)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		//amount, err2 := strconv.ParseFloat(channelResp.Amount, 64)
		//if err2 != nil {
		//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
		//}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.PayeeTitle,
			CardNumber: channelResp.PayeeAccountNumber,
			BankName:   channelResp.PayeeBankName,
			BankBranch: channelResp.PayeeAccountBankBranch,
			Amount:     channelResp.Payable,
			Link:       channelResp.TargetUrl,
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: "",
			IsCheckOutMer:  false, // 自組收銀台回傳 true
		}, nil
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.TargetUrl,
		ChannelOrderNo: "",
	}

	return
}
