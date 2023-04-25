package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/qipay/internal/svc"
	"github.com/copo888/channel_app/qipay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
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


	url := channel.PayQueryUrl+"?order_id="+req.OrderNo
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", url, req.OrderNo)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Get(url).
		Header("CLIENTSID", channel.MerId).
		Header("ACCESSTOKEN", channel.MerKey).
		Timeout(20).Trace(span).Do()

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
		Data []struct {
			OrderSid      int64  `json:"order_sid"`
			OrderId       string  `json:"order_id"`
			Payer         string  `json:"payer"`
			Amount        string `json:"amount"`
			Status        int64   `json:"status"`
			Type          int64   `json:"type"`  // 1 為收款,2 為出款
		} `json:"data"`
	}{}

	// 訂單狀態表
	// 0 新訂單
	// 1 已配對
	// 2 已收單
	// 3 已完成
	// 4 回調失敗
	// 90 付款超時
	// 91 收款超時
	// 92 金額不符
	// 95 訂單無效
	// 99 超時配對,無效單

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	} else if len(channelResp.Data) < 1 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, "查无资料")
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data[0].Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	channelStatus := channelResp.Data[0].Status
	orderStatus := "0"
	if channelStatus == 3 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
