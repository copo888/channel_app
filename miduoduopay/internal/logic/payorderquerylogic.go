package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/miduoduopay/internal/payutils"
	"github.com/copo888/channel_app/miduoduopay/internal/svc"
	"github.com/copo888/channel_app/miduoduopay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	traceID string
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
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
	data := url.Values{}
	data.Set("MERCHANT_ID", channel.MerId)
	data.Set("VERSION", "1")
	data.Set("TRAN_CODE", req.ChannelOrderNo)

	// 組請求參數 FOR JSON
	//data := struct {
	//	MerchantId  string `json:"MERCHANT_ID"`
	//	TranCode  string `json:"TRAN_CODE"`
	//	Version string `json:"VERSION"`
	//	SignedMsg    string `json:"SIGNED_MSG"`
	//}{
	//	MerchantId:  channel.MerId,
	//	TranCode:  req.OrderNo,
	//	Version: "1",
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("SIGNED_MSG", sign)
	//data.SignedMsg = sign

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
		Code    string `json:"code"`
		Message string `json:"message"`
		Data struct{
			MerchantId string `json:"MERCHANT_ID"`
			SysCode    string `json:"SYS_CODE"`
			TranCode   string `json:"TRAN_CODE"`
			TranAmt    string `json:"TRAN_AMT"`
			Fee        string `json:"FEE"`
			Type       string `json:"TYPE"`
			Status     string `json:"STATUS"`
			SubmitTime string `json:"SUBMIT_TIME"`
			payTime    string `json:"PAY_TIME"`
			signedMsg  string `json:"SIGNED_MSG"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != "G_00001" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Data.TranAmt, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Data.Status == "SUCCESS" {
		orderStatus = "1"
	}else if strings.Index("FAILED,MERCHANT_TIMEOUT,CANCEL", channelResp.Data.Status) > -1 {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
