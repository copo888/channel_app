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
	"github.com/copo888/channel_app/gtpay/internal/payutils"
	"github.com/copo888/channel_app/gtpay/internal/service"
	"github.com/copo888/channel_app/gtpay/internal/svc"
	"github.com/copo888/channel_app/gtpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
)

type PayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s,orderNo: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl := "http://8a3e-211-75-36-190.ngrok.io/api/pay-call-back"
	ip := utils.GetRandomIp()
	merchId, _ := strconv.ParseInt(channel.MerId, 10, 64)
	money, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	// 組請求參數 FOR JSON
	data := struct {
		MerchId    int64   `json:"merchant_id"`
		PayType    string  `json:"pay_type"`
		PayName    string  `json:"pay_name,optional"`
		PaySuffix  string  `json:"pay_suffix,optional"` // 付款卡后6位
		OrderId    string  `json:"out_trade_no"`
		NotifyUrl  string  `json:"notify_url"`
		ReturnUrl  string  `json:"return_url,optional"`
		Money      float64 `json:"money"`
		ClientIp   string  `json:"client_ip"`
		ReturnType string  `json:"return_type"`
		SignType   string  `json:"sign_type"`
		Sign       string  `json:"sign"`
	}{
		MerchId:   merchId,
		PayType:   req.ChannelPayType,
		PayName:   req.UserId,
		OrderId:   req.OrderNo,
		NotifyUrl: notifyUrl,
		ClientIp:  ip,
		ReturnUrl: req.PageUrl,
		Money:     money,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}
	if req.JumpType == "json" {
		data.ReturnType = "info"
	}
	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.SignType = "md5"
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data,
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	var channelResp1 struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,optional"`
	}

	channelResp := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MerchantId  int     `json:"merchant_id"`
			PayType     string  `json:"pay_type"`
			TradeNo     string  `json:"trade_no"`
			OutTradeNo  string  `json:"out_trade_no"`
			Money       float64 `json:"money"`
			Fee         float64 `json:"fee"`
			SignType    string  `json:"sign_type"`
			RealName    string  `json:"realname"`
			BankName    string  `json:"bank_name"`
			BankNumber  string  `json:"bank_number"`
			BankAddress string  `json:"bank_address"`
			PayUrl      string  `json:"pay_url"`
			Sign        string  `json:"sign"`
		} `json:"data,optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp1); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	if channelResp1.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp1.Message)
	} else {
		if err = res.DecodeJSON(&channelResp); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}
	}
	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.Data.RealName,
			CardNumber: channelResp.Data.BankNumber,
			BankName:   channelResp.Data.BankName,
			BankBranch: "",
			Amount:     channelResp.Data.Money,
			Link:       channelResp.Data.PayUrl,
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
		PayPageInfo:    channelResp.Data.PayUrl,
		ChannelOrderNo: "",
	}

	return
}
