package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/samplepay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"time"

	"github.com/copo888/channel_app/samplepay/internal/svc"
	"github.com/copo888/channel_app/samplepay/internal/types"

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

	logx.Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	timestamp := time.Now().Format("20060102150405")

	// 組請求參數
	data := url.Values{}
	data.Set("merchId", channel.MerId)
	data.Set("orderId", req.OrderNo)
	data.Set("time", timestamp)
	data.Set("signType", "MD5")

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	// 請求渠道
	logx.Infof("支付查詢请求地址:%s,支付請求參數:%#v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, err := gozzle.Post(channel.PayQueryUrl).Timeout(10).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(10).Trace(span).JSON(resp)

	logx.Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	if err != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	}

	// 渠道回覆處理
	channelResp := struct {
		Code  string `json:"code"`
		Msg   string `json:"msg"`
		State string `json:"state"` //状态 1-成功 2-等待付款 7-关闭
		Money string `json:"money"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Code != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Money, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.State == "1" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
