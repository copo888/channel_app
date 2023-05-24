package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/dpay/internal/payutils"
	"github.com/copo888/channel_app/dpay/internal/svc"
	"github.com/copo888/channel_app/dpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	data := url.Values{}
	if req.OrderNo != "" {
		data.Set("order_sn", req.OrderNo)
	}
	if req.ChannelOrderNo != "" {
		data.Set("order_sn", req.ChannelOrderNo)
	}
	data.Set("cus_code", channel.MerId)

	// 組請求參數 FOR JSON
	//data := struct {
	//	CusCode string `json:"cus_code"`
	//	OrderSn string `json:"order_sn"`
	//	Sign     string `json:"sign"`
	//}{
	//	CusCode: channel.MerId,
	//	OrderSn: req.OrderNo,
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Result  string `json:"result"`
		status  string `json:"status"`
		Message string `json:"message"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Result != "success" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	channelResp2 := struct {
		OrderInfo struct {
			OrderSn           string  `json:"order_sn"`
			CusOrderSn        string  `json:"cus_order_sn"`
			CurrencyType      string  `json:"currency_type"`
			OriginalAmount    float64 `json:"original_amount"`
			OrderAmount       float64 `json:"order_amount"`
			ReceiveAmount     float64 `json:"receive_amount"`
			ExchangeAmount    float64 `json:"exchange_amount"`
			OrderStatus       string  `json:"order_status"`
			ClientTriggered   string  `json:"client_triggered"`
			ClientTriggeredAt string  `json:"client_triggered_at"`
		} `json:"order_info, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp2); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}
	//orderAmount, errParse := strconv.ParseFloat(channelResp.Money, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp2.OrderInfo.OrderStatus == "success" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp2.OrderInfo.ReceiveAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
