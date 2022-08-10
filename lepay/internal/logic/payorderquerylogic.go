package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/lepay/internal/payutils"
	"github.com/copo888/channel_app/lepay/internal/svc"
	"github.com/copo888/channel_app/lepay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	//timestamp := time.Now().Format("20060102150405")

	// 組請求參數
	//data := url.Values{}
	//data.Set("merchant_number", channel.MerId)
	//data.Set("merchant_order_number", req.OrderNo)

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNumber      string `json:"merchant_number"`
		MerchantOrderNumber string `json:"merchant_order_number"`
		Sign                string `json:"sign"`
	}{
		MerchantNumber:      channel.MerId,
		MerchantOrderNumber: req.OrderNo,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%#v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	//res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	}

	// 渠道回覆處理
	channelResp := struct {
		Status              int64  `json:"status"` // 0 ⇒ 订单已建立	1 ⇒ 订单未付款	2 ⇒ 订单支付成功（注意 2、6 的状态皆为成功） 3 ⇒ 订单审核中 4 ⇒ 订单审核失败    5 ⇒ 订单支付失败 6 ⇒ 订单手动确认成功
		Amount              string `json:"amount"`
		MerchantOrderNumber string `json:"merchant_order_number"`
		SystemOrderNumber   string `json:"system_order_number"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}
	orderAmount, errParse := strconv.ParseFloat(channelResp.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Status == 2 || channelResp.Status == 6 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount:    orderAmount,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
		ChannelOrderNo: channelResp.SystemOrderNumber,
	}

	return
}
