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
	"github.com/copo888/channel_app/kypay/internal/payutils"
	"github.com/copo888/channel_app/kypay/internal/service"
	"github.com/copo888/channel_app/kypay/internal/svc"
	"github.com/copo888/channel_app/kypay/internal/types"
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
	if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	//組請求參數 FOR JSON
	data := struct {
		MerchId      string `json:"merchant_serial"`
		OrderId      string `json:"merchant_order_no"`
		PayType      string `json:"channel_type"`
		PayerName    string `json:"payer_name"`
		Money        string `json:"amount"`
		NotifyUrl    string `json:"notify_url"`
		CallbackUrl  string `json:"callback_url"`
		Format       string `json:"format"`
		RandomString string `json:"random_string"`
		Note         string `json:"note"`
		Sign         string `json:"sign"`
	}{
		MerchId:      channel.MerId,
		Money:        req.TransactionAmount,
		OrderId:      req.OrderNo,
		PayerName:    req.UserId,
		NotifyUrl:    notifyUrl,
		RandomString: payutils.GetSign(req.OrderNo),
		PayType:      req.ChannelPayType,
	}

	if strings.EqualFold(req.JumpType, "json") {
		data.Format = "JSON"
	} else {
		data.Format = "PAGE"
	}

	data.Sign = payutils.SortAndSignSHA256FromObj(data, channel.MerKey, l.ctx)
	data.CallbackUrl = notifyUrl
	data.Note = "note"
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
	// 若有證書問題 請使用

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

		channelResp2 := struct {
			Msg string `json:"message,optional"`
		}{}

		// 返回body 轉 struct
		if err = res.DecodeJSON(&channelResp2); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp2.Msg)
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Status string `json:"status,optional"`
		Msg    string `json:"msg,optional"`
		Data   struct {
			MerchantOrderNo string `json:"merchant_order_no,optional"`
			PlatformOrderNo string `json:"platform_order_no,optional"`
			PayerName       string `json:"payer_name,optional"`
			Amount          string `json:"amount,optional"`
			RealAmount      string `json:"real_amount,optional"`
			CreatedAt       string `json:"created_at,optional"`
			ExpiredAt       string `json:"expired_at,optional"`
		} `json:"data,optional"`
		PayUrl     string `json:"pay_url,optional"`
		Name       string `json:"name,optional"`
		BankCardNo string `json:"bank_card_no,optional"`
		BankName   string `json:"bank_name,optional"`
		BankBranch string `json:"bank_branch,optional"`
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
		Content:   fmt.Sprintf("%+v", channelResp),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Status != "success" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {

		amount, _ := strconv.ParseFloat(channelResp.Data.RealAmount, 64)

		isCheckOutMer := false // 自組收銀台回傳 true
		if req.MerchantId == "ME00015" {
			isCheckOutMer = true
		}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.Name,
			CardNumber: channelResp.BankCardNo,
			BankName:   channelResp.BankName,
			BankBranch: channelResp.BankBranch,
			Amount:     amount,
			Link:       "",
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: "",
			IsCheckOutMer:  isCheckOutMer, // 自組收銀台回傳 true
		}, nil
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.PayUrl,
		ChannelOrderNo: "",
	}

	return
}
