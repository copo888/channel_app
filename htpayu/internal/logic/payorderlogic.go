package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/htpayu/internal/payutils"
	"github.com/copo888/channel_app/htpayu/internal/service"
	"github.com/copo888/channel_app/htpayu/internal/svc"
	"github.com/copo888/channel_app/htpayu/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
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

	if len(req.PlayerId) == 0 {
		logx.WithContext(l.ctx).Errorf("playerId不可为空 playerId:%s", req.PlayerId)
		return nil, errorx.New(responsex.INVALID_PLAYER_ID)
	}
	if len(req.SourceIp) == 0 {
		logx.WithContext(l.ctx).Errorf("sourceIp:%s", req.SourceIp)
		return nil, errorx.New(responsex.INVALID_USER_IP)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	amountStr := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	data := url.Values{}
	data.Set("amount", amountStr)
	data.Set("clientip", req.SourceIp)
	data.Set("currency", "USDT")
	data.Set("mhtorderno", req.OrderNo)
	data.Set("mhtuserid", req.PlayerId)
	data.Set("notifyurl", notifyUrl)
	data.Set("opmhtid", channel.MerId)
	data.Set("paytype", "crypto_payasyougo")
	data.Set("random", "5bJSg3QKK1g5")
	data.Set("returnurl", req.PageUrl)
	data.Set("extra", "{\"channel\":\"TRC20\"}")

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("sign", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		OrderNo:    req.OrderNo,
		LogType:    constants.DATA_REQUEST_CHANNEL,
		LogSource:  constants.API_ZF,
		Content:    data,
		TraceId:    l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		RtCode int    `json:"rtCode"`
		Msg    string `json:"msg"`
		Result struct {
			Pforderno string `json:"pforderno,optional"`
			Payurl    string `json:"payurl,optional"`
		} `json:"result,optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		OrderNo:    req.OrderNo,
		LogType:    constants.RESPONSE_FROM_CHANNEL,
		LogSource:  constants.API_ZF,
		Content:    fmt.Sprintf("%+v", channelResp),
		TraceId:    l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.RtCode != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Result.Payurl,
		ChannelOrderNo: channelResp.Result.Pforderno,
	}

	return
}
