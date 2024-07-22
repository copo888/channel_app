package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/direpay/internal/payutils"
	"github.com/copo888/channel_app/direpay/internal/svc"
	"github.com/copo888/channel_app/direpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
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
	// 組請求參數

	// 組請求參數 FOR JSON
	data := struct {
		Command  string `json:"command"`
		HashCode string `json:"hashCode"`
		TxId     string `json:"txid, optional"`
	}{
		Command:  "fiat_payment_status",
		HashCode: payutils.GetSign("fiat_payment_status" + channel.MerKey),
		TxId:     req.OrderNo,
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).
		Header("Authorization", "Bearer "+l.svcCtx.Config.AccessToken).
		Header("Content-type", "application/json").
		JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if (res.Status() < 400 && res.Status() >= 300) || res.Status() < 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, string(res.Body()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Type          string `json:"type, optional"`
		Txid          string `json:"txid, optional"`
		CreatedAt     string `json:"created_at, optional"`
		RequestAmount string `json:"request_amount, optional"`
		Currency      string `json:"currency, optional"`
		ActualAmount  string `json:"actual_amount, optional"`
		Status        string `json:"status, optional"`
		Method        string `json:"method, optional"`
		Message       string `json:"message, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}
	var orderAmount float64
	var errParse error
	if len(channelResp.ActualAmount) > 0 {
		orderAmount, errParse = strconv.ParseFloat(channelResp.ActualAmount, 64)
		if errParse != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
		}
	} else if len(channelResp.RequestAmount) > 0 {
		orderAmount, errParse = strconv.ParseFloat(channelResp.RequestAmount, 64)
		if errParse != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
		}
	}

	orderStatus := "0"
	if channelResp.Status == "completed" { //pending, completed, too_little, too_much, expired
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
