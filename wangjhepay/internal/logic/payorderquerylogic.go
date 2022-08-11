package logic

import (
	"context"
	"crypto/tls"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/wangjhepay/internal/payutils"
	"github.com/copo888/channel_app/wangjhepay/internal/svc"
	"github.com/copo888/channel_app/wangjhepay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/url"
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

	// 組請求參數
	data := url.Values{}
	data.Set("order_sn", req.OrderNo)
	data.Set("bid", channel.MerId)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	// 忽略證書
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Transport(tr).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)
	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理
	channelResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg, optional"`
		Time int64  `json:"time"`
		Data struct {
			OrderSn      string `json:"order_sn"`
			SysOrderSn   string `json:"sys_order_sn"`
			Money        string `json:"money"`
			PayMoney     string `json:"pay_money"`
			AddTime      int64  `json:"add_time"`
			PayTime      int64  `json:"pay_time"`
			PayState     int64  `json:"pay_state"` //支付状态： 0未支付,1已支付,2已取消
			PayStateText string `json:"pay_state_text"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Code != 100 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderStatus := "0"
	if channelResp.Data.PayState == 1 {
		orderStatus = "1"
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data.PayMoney, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, errParse.Error())
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount:    orderAmount,
		ChannelOrderNo: channelResp.Data.SysOrderSn,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
