package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/baisiangpay/internal/payutils"
	"github.com/copo888/channel_app/baisiangpay/internal/svc"
	"github.com/copo888/channel_app/baisiangpay/internal/types"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 組請求參數 FOR JSON
	data := struct {
		PayOrderId    string `json:"pay_order_id"`
		PayMerchantId string `json:"pay_merchant_id"`
		PayDatetime   string `json:"pay_datetime"`
		PaySign       string `json:"pay_sign"`
	}{
		PayOrderId:    req.OrderNo,
		PayMerchantId: channel.MerId,
		PayDatetime:   timestamp,
	}

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.PaySign = sign

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
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PayTransactionId string `json:"pay_transaction_id"`
			PayAmount        string `json:"pay_amount"`
			PayRealAmount    string `json:"pay_real_amount"`
			PayStatus        string `json:"pay_status"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Message != "success" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data.PayAmount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Data.PayStatus == "30001" { // 30000 :未付款, 30001 :收款成功 30005 .表示收款失败
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		ChannelOrderNo: channelResp.Data.PayTransactionId,
		OrderAmount:    orderAmount,
		OrderStatus:    orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
