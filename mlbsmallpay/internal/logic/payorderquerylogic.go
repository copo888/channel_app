package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/mlbsmallpay/internal/payutils"
	"github.com/copo888/channel_app/mlbsmallpay/internal/svc"
	"github.com/copo888/channel_app/mlbsmallpay/internal/types"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	unix := time.Now().Unix()
	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo string `json:"merchantNo"`
		TradeNo    string `json:"tradeNo"`
		OrderNo    string `json:"orderNo"`
		Time       int64 `json:"time"`
		AppSecret  string `json:"appSecret"`
		Sign       string `json:"sign"`
	}{
		MerchantNo:  channel.MerId,
		OrderNo:  req.OrderNo,
		Time:     unix,
		AppSecret:  l.svcCtx.Config.AppSecret,
	}

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
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
		Code  int64 `json:"code"`
		Text   string `json:"text"`
		Status string `json:"status"` //状态 1-成功 2-等待付款 7-关闭
		Paid float64 `json:"paid"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Text)
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.Paid, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.Status == "PAID" || channelResp.Status == "MANUAL PAID" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.Paid,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
