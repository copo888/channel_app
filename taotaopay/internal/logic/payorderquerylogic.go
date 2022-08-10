package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/taotaopay/internal/payutils"
	"github.com/copo888/channel_app/taotaopay/internal/svc"
	"github.com/copo888/channel_app/taotaopay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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

	randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)

	// 組請求參數
	data := url.Values{}
	data.Set("appid", channel.MerId)
	if req.OrderNo != "" {
		data.Set("trade_no", req.OrderNo) //商户订单号(COPO)
	}
	if req.ChannelOrderNo != "" {
		data.Set("order_no", req.ChannelOrderNo) //系统订单号(渠道單號)
	}
	data.Set("nonce_str", randomID)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%#v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	}

	// 渠道回覆處理
	channelResp := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg, optional"`
		Data struct {
			Order struct {
				OrderNo string  `json:"order_no"` //支付中心生成的订单号
				Money   float64 `json:"money"`
				IsPay   int     `json:"is_pay"` //订单状态0未支付 1交易成功
				Sign    string  `json:"sign"`
				PayUrl  string  `json:"pay_url"`
			} `json:"order"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}
	parseAmount := strconv.FormatFloat(channelResp.Data.Order.Money, 'f', 0, 64)
	orderAmount := utils.FloatDiv(parseAmount, "100")

	orderNo := channelResp.Data.Order.OrderNo

	orderStatus := "0"
	if channelResp.Data.Order.IsPay == 1 { //订单状态0未支付 1交易成功
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount:    orderAmount,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
		ChannelOrderNo: orderNo,
	}

	return
}
