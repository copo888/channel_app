package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/vp8pay/internal/svc"
	"github.com/copo888/channel_app/vp8pay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

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

	// 請求渠道
	url := channel.PayQueryUrl + "/" + req.OrderNo
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", url)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Get(url).
		Header("Accept", "application/json").
		Header("Content-Type", "application/json").
		Header("Authorization", "Bearer "+channel.MerKey).
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
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			TradeNo       string  `json:"trade_no"`
			OutTradeNo    string  `json:"out_trade_no"`
			Amount        string  `json:"amount"`
			RequestAmount float64 `json:"request_amount"`
			State         string  `json:"state"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if !channelResp.Success {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	// processing => 处理中 timeout => 逾時 completed => 成功 failed => 失败
	orderStatus := "0"
	if channelResp.Data.State == "completed" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount:    orderAmount,
		ChannelOrderNo: channelResp.Data.TradeNo,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
