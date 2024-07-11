package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/bcpay/internal/payutils"
	"github.com/copo888/channel_app/bcpay/internal/svc"
	"github.com/copo888/channel_app/bcpay/internal/types"
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
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjYxYWRhNzM3ZWNmMzIwMjE3ZmVlYzUyZDIzNDgyNTkwOTI1YjAyNjI2YzY1MjAwMDk4ODc1ZmY2NzI2N2FkMjdkNGY3MmQ5NmVkYzgzNDY4In0.eyJhdWQiOiIxIiwianRpIjoiNjFhZGE3MzdlY2YzMjAyMTdmZWVjNTJkMjM0ODI1OTA5MjViMDI2MjZjNjUyMDAwOTg4NzVmZjY3MjY3YWQyN2Q0ZjcyZDk2ZWRjODM0NjgiLCJpYXQiOjE3MjA2ODcxMjQsIm5iZiI6MTcyMDY4NzEyNCwiZXhwIjoxNzIxMjkxOTI0LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.loWg5juLro4FtuwOY-5ui_D1CQ_IcCZjOo3pYE-3cEcZSTbuQ9WCoQfJvL7daofBsBh8CkiHeVtQRx3S9QEUD_edVv5J83uHpyCJaMN3wvIE8K1DbBbhbWEK6WHl46bLHr4Akj-wZzUd8cx10OXBZFq6v5uZiQ73V-GJqP3NhufcXU1p10KbCSwsYiyqNd8F6-4p6Bmg5YucriQ5jM7KXxkTwBb09RNf4B7f2p2_QCw8YpbcwM6IDhdslUHwgRmHwf1fxESvw4im6Vjd6yfVcyjnex9jlItv_dkibvVd5Z-iTDz4_DzM8y1OlXiqRJD55dj0Y6gl5mCDb7RrZIpDC9NHuzdcL0GfetYLEM0hWazBAULPypRsJ79a2RhEvfopoPgkntv10mQmQ1U3X9vo3wRDZoUqfWdiQ2Xy-1kW0Cdg7CM2bSQExmKvdGxj1CVB8fSqFZqBVP4vrXzdQ40hS29rPoEZTg2VqDRK8AKgQjSFr4USaiq-bMq680Ok_y3kaadmEII86o8JtuBeDVKIdYImBsN8QNfIUezoAPgEmFvTGU9fDw4J69CaFWp9anTDCSCKNuRmcs7cZPK7Kz3WVjN35eMTInP2GTDCX-H8tYuzXESKIIrliONkYqGGU4JJ9Lhb11WMg3NJUpTQ3HPFm3tyCgrgMA8--QI0XkLfp50eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjYxYWRhNzM3ZWNmMzIwMjE3ZmVlYzUyZDIzNDgyNTkwOTI1YjAyNjI2YzY1MjAwMDk4ODc1ZmY2NzI2N2FkMjdkNGY3MmQ5NmVkYzgzNDY4In0.eyJhdWQiOiIxIiwianRpIjoiNjFhZGE3MzdlY2YzMjAyMTdmZWVjNTJkMjM0ODI1OTA5MjViMDI2MjZjNjUyMDAwOTg4NzVmZjY3MjY3YWQyN2Q0ZjcyZDk2ZWRjODM0NjgiLCJpYXQiOjE3MjA2ODcxMjQsIm5iZiI6MTcyMDY4NzEyNCwiZXhwIjoxNzIxMjkxOTI0LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.loWg5juLro4FtuwOY-5ui_D1CQ_IcCZjOo3pYE-3cEcZSTbuQ9WCoQfJvL7daofBsBh8CkiHeVtQRx3S9QEUD_edVv5J83uHpyCJaMN3wvIE8K1DbBbhbWEK6WHl46bLHr4Akj-wZzUd8cx10OXBZFq6v5uZiQ73V-GJqP3NhufcXU1p10KbCSwsYiyqNd8F6-4p6Bmg5YucriQ5jM7KXxkTwBb09RNf4B7f2p2_QCw8YpbcwM6IDhdslUHwgRmHwf1fxESvw4im6Vjd6yfVcyjnex9jlItv_dkibvVd5Z-iTDz4_DzM8y1OlXiqRJD55dj0Y6gl5mCDb7RrZIpDC9NHuzdcL0GfetYLEM0hWazBAULPypRsJ79a2RhEvfopoPgkntv10mQmQ1U3X9vo3wRDZoUqfWdiQ2Xy-1kW0Cdg7CM2bSQExmKvdGxj1CVB8fSqFZqBVP4vrXzdQ40hS29rPoEZTg2VqDRK8AKgQjSFr4USaiq-bMq680Ok_y3kaadmEII86o8JtuBeDVKIdYImBsN8QNfIUezoAPgEmFvTGU9fDw4J69CaFWp9anTDCSCKNuRmcs7cZPK7Kz3WVjN35eMTInP2GTDCX-H8tYuzXESKIIrliONkYqGGU4JJ9Lhb11WMg3NJUpTQ3HPFm3tyCgrgMA8--QI0XkLfp50").
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
