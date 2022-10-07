package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"
	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
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

	// 組請求參數 FOR JSON
	paramsStruct := struct {
		MerchantOrderId string `json:"merchantOrderId"`
	}{
		MerchantOrderId: req.OrderNo,
	}
	paramsJson, err := json.Marshal(paramsStruct)
	paramsJsonStr := string(paramsJson[:])

	_, params := payutils.AesEncrypt(paramsJsonStr, l.svcCtx.Config.AesKey)

	merchantNo, _ := strconv.ParseInt(channel.MerId, 10, 64)

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo int64  `json:"merchantNo"`
		Signature  string `json:"signature"`
		Params     string `json:"params"`
	}{
		MerchantNo: merchantNo,
		Params:     params,
		Signature:  payutils.Md5V(paramsJsonStr+channel.MerKey, l.ctx),
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%+v,支付原始參數:%s", channel.PayQueryUrl, data, paramsJsonStr)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, chnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderNo         string  `json:"orderNo"`
			OrderStatus     int64   `json:"orderStatus"`
			OrderAmount     float64 `json:"orderAmount"`
			PaidAmount      float64 `json:"paidAmount"`
			PlayerName      string  `json:"playerName"`
			MerchantOrderId string  `json:"merchantOrderId"`
			DepositName     string  `json:"depositName"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderStatus := "0"
	if channelResp.Data.OrderStatus == 1 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.Data.PaidAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
