package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/chiangshengpay/internal/payutils"
	"github.com/copo888/channel_app/chiangshengpay/internal/svc"
	"github.com/copo888/channel_app/chiangshengpay/internal/types"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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
	data := struct {
		UserId  string `json:"userId"`
		OrderId string `json:"orderId"`
		Type    string `json:"type"`
		Sign    string `json:"sign"`
	}{}
	data.UserId = channel.MerId
	data.OrderId = req.OrderNo
	data.Type = "pay"

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

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

	channelResp := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Code    int    `json:"code"`
		Result  struct {
			OrderId     string `json:"orderId"`
			OrderNo     string `json:"orderNo"`
			OrderStatus string `json:"orderStatus"`
			Amount      string `json:"amount"`
			UserId      string `json:"userId"`
			Sign        string `json:"sign"`
		} `json:"result"`
	}{}

	// 渠道回覆處理
	//channelResp2 := struct {
	//	OrderId string `json:"orderId"`
	//	State   string `json:"orderStatus"` //充值订单：1处理中；2成功；3失败
	//	Money   string `json:"amount"`
	//	Sign    string `json:"sign"`
	//	AppId   string `json:"appId"`
	//}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.Result, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.Result.OrderStatus == "2" { //4 失败	2 成功 	其他情况	未处理
		orderStatus = "1"
	} else if channelResp.Result.OrderStatus == "4" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: 0,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
