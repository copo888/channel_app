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
	"github.com/copo888/channel_app/taotaopay/internal/payutils"
	"github.com/copo888/channel_app/taotaopay/internal/svc"
	"github.com/copo888/channel_app/taotaopay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	//channelPayTypeModel := &typesX.ChannelPayType{}
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl := "http://f828-211-75-36-190.ngrok.io/taotaopay/api/pay-call-back"
	randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	amount := utils.FloatMul(req.TransactionAmount, "100")

	// 組請求參數
	data := url.Values{}
	data.Set("money", strconv.FormatFloat(amount, 'f', 0, 64))
	data.Set("trade_no", req.OrderNo)
	data.Set("notify_url", notifyUrl)
	data.Set("order_type", "0")
	data.Set("pay_code", req.ChannelPayType)
	data.Set("appid", channel.MerId)
	data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)
	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg, optional"`
		Data struct {
			PayInfo struct {
				OrderNo string `json:"order_no"`
				TradeNo string `json:"trade_no"`
				Money   string `json:"money"`
				//IsPay       `json:"is_pay"`
				//PayMoney  float64 `json:"pay_money"`
				Sign   string `json:"sign"`
				PayUrl string `json:"pay_url"`
			} `json:"pay_info"`
		} `json:"data"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
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

	if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	amountTrans := utils.FloatDiv(channelResp.Data.PayInfo.Money, "100")
	amountFormat := strconv.FormatFloat(amountTrans, 'f', 0, 64)
	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Data.PayInfo.PayUrl,
		OrderAmount:    amountFormat,
		ChannelOrderNo: channelResp.Data.PayInfo.TradeNo,
		Status:         "1",
	}

	return
}
