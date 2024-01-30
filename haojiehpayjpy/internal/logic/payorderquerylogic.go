package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/haojiehpayjpy/internal/payutils"
	"github.com/copo888/channel_app/haojiehpayjpy/internal/svc"
	"github.com/copo888/channel_app/haojiehpayjpy/internal/types"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %#v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	appId := "145fd6be4e5b4187839447579d70a984"
	// 組請求參數
	data := url.Values{}
	//if req.OrderNo != "" {
	//	data.Set("trade_no", req.OrderNo)
	//}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	//data.Set("appid", channel.MerId)
	//data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON
	dataJs := struct {
		MchId         string `json:"mchId"`
		AppId         string `json:"appId"`
		MchOrderNo    string `json:"mchOrderNo"`
		ExecuteNotify string `json:"executeNotify"`
		Sign          string `json:"sign"`
	}{
		MchId:         channel.MerId,
		AppId:         appId,
		MchOrderNo:    req.OrderNo,
		ExecuteNotify: "false",
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(dataJs, channel.MerKey)
	dataJs.Sign = sign

	b, err := json.Marshal(dataJs)
	if err != nil {
		fmt.Println("error:", err)
	}
	data.Set("params", string(b))
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
		RetCode        string `json:"retCode"`
		RetMsg         string `json:"RetMsg, optional"`
		MchId          string `json:"mchId, optional"`
		AppId          string `json:"appId, optional"`
		ProductId      string `json:"productId, optional"`
		PayOrderId     string `json:"payOrderId, optional"`
		MchOrderNo     string `json:"mchOrderNo, optional"`
		Amount         string `json:"amount, optional"`
		Currency       string `json:"currency, optional"`
		Status         string `json:"status, optional"`
		ChannelUser    string `json:"channelUser, optional"`
		ChannelOrderNo string `json:"channelOrderNo, optional"`
		ChannelAttach  string `json:"channelAttach, optional"`
		PaySuccTime    string `json:"paySuccTime, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.RetCode != "SUCCESS" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RetMsg)
	}

	amountF, _ := strconv.ParseFloat(channelResp.Amount, 64)
	orderAmount := utils.FloatDivF(amountF, 100) // 單位:元

	orderStatus := "0"
	if channelResp.Status == "2" || channelResp.Status == "3" { //支付状态,0-订单生成,1-支付中,2- 支付成功,3-业务处理完成(支付成功),5-支付失败
		orderStatus = "1"
	} else if channelResp.Status == "5" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
