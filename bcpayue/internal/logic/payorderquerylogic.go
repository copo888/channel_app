package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/bcpayue/internal/payutils"
	"github.com/copo888/channel_app/bcpayue/internal/svc"
	"github.com/copo888/channel_app/bcpayue/internal/types"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
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
	// 組請求參數

	// 組請求參數 FOR JSON
	data := struct {
		Command  string `json:"command"`
		HashCode string `json:"hashCode"`
		TxId     string `json:"txid, optional"`
	}{
		Command:  "transaction_status",
		HashCode: payutils.GetSign("transaction_status" + channel.MerKey),
		TxId:     req.OrderNo,
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjRiNjY2YjJiMjU4OTk2NjYyYjdjMzMzOWNlOTQ2OGI0ZTFmMGJmOWFlM2U0MTk2YjM4YThjNGE5ZGIzODZmNTMyZjkxMTk5YmExNTMwZDJlIn0.eyJhdWQiOiIxIiwianRpIjoiNGI2NjZiMmIyNTg5OTY2NjJiN2MzMzM5Y2U5NDY4YjRlMWYwYmY5YWUzZTQxOTZiMzhhOGM0YTlkYjM4NmY1MzJmOTExOTliYTE1MzBkMmUiLCJpYXQiOjE3MTUwNTMyNDYsIm5iZiI6MTcxNTA1MzI0NiwiZXhwIjoxNzE1NjU4MDQ1LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.F5kXd0iUAxMG5EU9D33gdIGIe58r5OHDfun-xfXZ0L7hoIdWZXsudL9kR637r4b_MRQz8oeUeOuAFFwF0eEHxW-0YtE6tySzJggwwHE2TRnjrleG3WlQUpIudiu_J9QCU03mJMWGqJyyAeRLL0julZYX5U3zpk0Bl5gzOH7BgQgcBRCUq8mKyR-QtO6IJLP6HLlSaRVNoM1_Ze8C7VgX9Fyko95ALTENrlr8DWggGkqoimK8vMmkxcMs06B8f3tIBY0XyMi9WnVaCVhMxjrMFik9DsVAr9QOXcKoxo-tO3k8-5oG75jmRLitVzt4vtLfbSnPShP2cmJPMSj6xSoIoosMW3mg0zPk8N--SaOy2uBf-Qhle3kBg44OJSY0q_7f33WYjgLp-8vpPoaCML2Q_Hd85iza0Yn1EwM1axGfXnDAX80w-y-6wSjrdVCGPO3XyV3tb8wGfSc_Ga5F7UFsKVZTm-Il4_DqPQXIXcCZtKk-i2qQ4Ksdaq_uuf4ZdOUHLiWth3zpvzGRw2n2A5gvRtESfHAS454ntt61c5aCLxkUhy04XYvhZtPsv1vSCOEcXxnmMGc11_wGQeZHodYdTRSBkSay_-jav3yaWzqswpZ3Q5BzFoZKHDFkcRftwICz7624T7fiC5iLnYIL6y8oqf-WMWoLf3JQ71b_5BR9eBU").
		Header("Content-type", "application/json").
		JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Txid      string `json:"txid, optional"`
		CreatedAt struct {
			Date         string `json:"date, optional"`
			TimezoneType int    `json:"timezone_type, optional"`
			Timezone     string `json:"timezone, optional"`
		} `json:"created_at, optional"`
		LineItems []struct {
			Name        string `json:"name, optional"`
			ItemId      string `json:"item_id, optional"`
			Description string `json:"description, optional"`
			Amount      string `json:"amount, optional"`
			Quantity    string `json:"quantity, optional"`
		} `json:"line_items, optional"`
		InvoiceAmount     float64     `json:"invoice_amount, optional"`
		InvoiceCurrency   string      `json:"invoice_currency, optional"`
		PaymentAmount     string      `json:"payment_amount, optional"`
		PaymentCurrency   string      `json:"payment_currency, optional"`
		CustomerUid       string      `json:"customer_uid, optional"`
		CustomerEmail     interface{} `json:"customer_email, optional"`
		ClientReferenceId interface{} `json:"client_reference_id, optional"`
		Status            string      `json:"status, optional"`
		TxHash            interface{} `json:"tx_hash, optional"`
		Rates             string      `json:"rates, optional"`
		EnablePromotion   bool        `json:"enable_promotion, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.PaymentAmount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Status == "completed" { //pending, completed, too_little, too_much, expired
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
