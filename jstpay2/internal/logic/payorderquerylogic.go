package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/jstpay2/internal/payutils"
	"github.com/copo888/channel_app/jstpay2/internal/svc"
	"github.com/copo888/channel_app/jstpay2/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
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
	timestamp := time.Now().Format("20060102150405")

	// 組請求參數 FOR JSON
	data := struct {
		DesTime    string `json:"DesTime"`
		CID        string `json:"CID"`
		BID        string `json:"BID"`
		BillSerial string `json:"BillSerial"`
		SignCode   string `json:"SignCode"`
	}{
		DesTime:    timestamp,
		CID:        channel.MerId,
		BID:        "1",
		BillSerial: req.OrderNo,
	}

	// 加簽 JSON
	source := "BID=" + data.BID + "&CID=" + data.CID + "&BillSerial=" + data.BillSerial + "&DesTime=" + data.DesTime + "&Key=" + channel.MerKey
	sign := payutils.GetSign(source)
	logx.Info("加签参数: ", source)
	logx.Info("签名字串: ", sign)
	data.SignCode = sign

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
		Success bool   `json:"Success"`
		Message string `json:"Message, optional"`
		Result  []struct {
			Status     int64   `json:"Status"`
			PayValue   float64 `json:"PayValue"`
			BillSerial string  `json:"BillSerial"`
		} `json:"Result"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if !channelResp.Success {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	orderStatus := "0"
	if channelResp.Result[0].Status == 1 { // 0:新订单 1:订单完成 2:订单取消 3:订单逾期
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount:    channelResp.Result[0].PayValue,
		ChannelOrderNo: channelResp.Result[0].BillSerial,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
