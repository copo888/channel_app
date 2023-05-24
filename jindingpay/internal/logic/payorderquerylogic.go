package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/jindingpay/internal/payutils"
	"github.com/copo888/channel_app/jindingpay/internal/svc"
	"github.com/copo888/channel_app/jindingpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	traceID string
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	//data := url.Values{}
	//if req.OrderNo != "" {
	//	data.Set("trade_no", req.OrderNo)
	//}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	//data.Set("appid", channel.MerId)
	//data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON
	data := struct {
		TradeNo string `json:"tradeNo"`
		Cid string `json:"cid"`
		Type string `json:"type"`
		Sign string `json:"sign"`
	}{
		TradeNo: req.OrderNo,
		Cid: channel.MerId,
		Type: "003",
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	//res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Retcode string `json:"retcode"`
		Retmsg string `json:"retmsg"`
		Status string `json:"status"` //1：代付成功；2：处理中；3：失败
		Amount float64 `json:"amount"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Retcode != "0" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Retmsg)
	}
	respAmount := utils.FloatDivF(channelResp.Amount, 100)

	orderStatus := "0"
	if channelResp.Status == "1" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: respAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
