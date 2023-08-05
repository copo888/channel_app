package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/leimapay/internal/payutils"
	"github.com/copo888/channel_app/leimapay/internal/svc"
	"github.com/copo888/channel_app/leimapay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
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
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	//data := url.Values{}
	//if req.OrderNo != "" {
	//	data.Set("trade_no", req.OrderNo)
	//}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	//data.Set("appid", channel.MerId)
	//data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON
	data := struct {
		MerchId  string `json:"merchantCode"`
		OrderId  string `json:"merchantOrderId"`
		Sign     string `json:"sign"`
	}{
		MerchId:  channel.MerId,
		OrderId:  req.OrderNo,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("mac", sign)

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)
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
		Result bool `json:"result"`
		ErrorMsg struct{
			Code string `json:"code"`
			ErrorMsg string `json:"errorMsg"`
			Descript string `json:"descript"`
		} `json:"errorMsg, optional"`
		Data struct{
			GamerOrderId string `json:"gamerOrderId"`
			MerchantOrderId string `json:"merchantOrderId"`
			CurrencyCode string `json:"currencyCode"`
			PaymentTypeCode string `json:"paymentTypeCode"`
			Amount string `json:"amount"`
			Remark string `json:"remark"`
			Status string `json:"status"`
		} `json:"data, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Result != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg.ErrorMsg)
	}

	//orderAmount := utils.FloatDiv(channelResp.TxnAmt, "100")

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Data.Status == "Success" {
		orderStatus = "1"
	}else if channelResp.Data.Status == "Failed" || channelResp.Data.Status == "Unpaid" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
