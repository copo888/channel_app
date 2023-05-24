package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/hqpay/internal/payutils"
	"github.com/copo888/channel_app/hqpay/internal/svc"
	"github.com/copo888/channel_app/hqpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	data := url.Values{}
	data.Set("paymentReference", req.OrderNo)
	data.Set("merchant", channel.MerId)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("sign", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)

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
		Whether string `json:"whether"`
		Msg     string `json:"message,optional"`
		PayInfo struct {
			Price        float64 `json:"price,optional"`
			RevisedPrice float64 `json:"revisedPrice,optional"` //修正价格
			ScanTime     int64   `json:"scanTime,optional"`
			CreateTime   int64   `json:"createTime,optional"`
			Status       int     `json:"statusStr,optional"` //0，待处理。1，成功。2，失败。
			StatusStr    string  `json:"StatusStr,optional"`
		} `json:"data, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.PayInfo.Price, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.PayInfo.Status == 1 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.PayInfo.Price,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
