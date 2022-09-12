package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/uzpay/internal/payutils"
	"github.com/copo888/channel_app/uzpay/internal/svc"
	"github.com/copo888/channel_app/uzpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	//var payTypeCode string
	//if err = l.svcCtx.MyDB.Table("tx_orders").Select("pay_type_code").Where("order_no = ?", req.OrderNo).Take(&payTypeCode).Error; err != nil {
	//	return nil, err
	//}
	//
	var merKey, uid string
	uid = "55640"
	merKey = "405967717d0f5371031d2a73c1eb9951"
	//if payTypeCode == "AK"{
	//	uid = "55639"
	//	merKey = "2161963f230d7fcc7656aeeb3634b074"
	//}else if payTypeCode == "YK" {
	//	uid = "55640"
	//	merKey = "405967717d0f5371031d2a73c1eb9951"
	//}else if payTypeCode == "A5" {
	//	uid = "55641"
	//	merKey = "2ff3ef77a229c0111ef2033b8f25343a"
	//}
	// 組請求參數
	data := url.Values{}
	if req.OrderNo != "" {
		data.Set("orderid", req.OrderNo)
	}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	data.Set("uid", uid)

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
	sign := payutils.SortAndSignFromUrlValues(data, merKey)
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
		Success bool `json:"success"`
		Code  int `json:"code"`
		Msg   string `json:"msg"`
		Order struct{
			Oid string `json:"oid"`
			Orderid string `json:"orderid"`
			Userid string `json:"userid"`
			Amount string `json:"amount"`
			TransactionAmount string `json:"transactionAmount"`
			Type string `json:"type"`
			Status string `json:"status"`
		}
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Order.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Order.Status == "verified" {
		orderStatus = "1"
	}else if channelResp.Order.Status == "revoked" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
