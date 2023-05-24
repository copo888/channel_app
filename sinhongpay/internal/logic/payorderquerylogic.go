package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/sinhongpay/internal/payutils"
	"github.com/copo888/channel_app/sinhongpay/internal/svc"
	"github.com/copo888/channel_app/sinhongpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
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
	randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	timestamp := time.Now().Format("20060102150405")
	// 組請求參數
	data := url.Values{}
	if req.OrderNo != "" {
		data.Set("out_trade_no", req.OrderNo)
	}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	headMap := make(map[string]string)
	headMap["sid"] = channel.MerId
	headMap["timestamp"] = timestamp
	headMap["nonce"] = randomID
	headMap["url"] = "/pay/orderquery"

	// 組請求參數 FOR JSON
	//data := struct {
	//	merchId  string
	//	orderId  string
	//	time     string
	//	signType string
	//	sign     string
	//}{
	//	merchId:  channel.MerId,
	//	orderId:  req.OrderNo,
	//	time:     timestamp,
	//	signType: "MD5",
	//}

	// 加簽
	headSource := payutils.JoinStringsInASCII(headMap, "", false, false, "")
	newSource := headSource + "out_trade_no" + req.OrderNo + channel.MerKey
	newSign := payutils.GetSign(newSource)
	logx.WithContext(l.ctx).Info("加签参数: ", newSource)
	logx.WithContext(l.ctx).Info("签名字串: ", newSign)
	headMap["sign"] = newSign

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Headers(headMap).Timeout(20).Trace(span).Form(data)
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
		Code       string `json:"code"`
		Msg        string `json:"msg,optional"`
		Status     string `json:"status,optional"` //状态 WAIT 等待支付 SUCCESS 支付成功 CLOSE订单关闭 UNCLAIMED 未认领 ERROR错误金额订单
		Amount     string `json:"amount,optional"`
		PayAmount  string `json:"pay_amount, optional"`
		OutTradeNo string `json:"out_trade_no,optional"`
		TradeTime  string `json:"trade_time,optional"`
		Currency   string `json:"currency, optional"`
		Sign       string `json:"sign, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != "1000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.PayAmount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Status == "SUCCESS" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
