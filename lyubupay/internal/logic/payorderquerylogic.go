package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/lyubupay/internal/payutils"
	"github.com/copo888/channel_app/lyubupay/internal/svc"
	"github.com/copo888/channel_app/lyubupay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
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
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 組請求參數 FOR JSON
	data := struct {
		MchId      string `json:"mch_id"`
		OutTradeNo string `json:"out_trade_no"`
		Timestamp  string `json:"timestamp"`
		Sign       string `json:"sign"`
	}{
		MchId:      channel.MerId,
		OutTradeNo: req.OrderNo,
		Timestamp:  timestamp,
	}

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg, optional"`
		Data struct {
			TradeNo string `json:"trade_no"`
			PayUrl  string `json:"pay_url"`
			State   int64  `json:"state"`
			Money   string `json:"money"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data.Money, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Data.State == 2 { // 订单状态：0=失败，1=待支付，2=支付成功，3=冲正
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount:    orderAmount,
		ChannelOrderNo: channelResp.Data.TradeNo,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
