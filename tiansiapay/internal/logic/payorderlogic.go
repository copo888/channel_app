package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"
	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
)

type PayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	randomIp := utils.GetRandomIp()
	payAmount, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	deviceId := utils.GetRandomString(16,0,0)
	// 組請求參數 FOR JSON
	paramsStruct := struct {
		UserName         string  `json:"userName"`
		DeviceType       int64   `json:"deviceType"`
		DeviceId         string  `json:"deviceId"`
		LoginIp          string  `json:"loginIp"`
		MerchantOrderId  string  `json:"merchantOrderId"`
		DepositNotifyUrl string  `json:"depositNotifyUrl"`
		PayAmount        float64 `json:"payAmount"`
		DepositName      string  `json:"depositName"`
	}{
		UserName:         req.PlayerId,
		DeviceType:       9,
		DeviceId:         payutils.Md5V(deviceId, l.ctx),
		LoginIp:          randomIp,
		MerchantOrderId:  req.OrderNo,
		DepositNotifyUrl: notifyUrl,
		PayAmount:        payAmount,
		DepositName:      req.UserId,
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
		Params: params,
		Signature: payutils.Md5V(paramsJsonStr + channel.MerKey, l.ctx),
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v,支付原始參數:%s", channel.PayUrl, data, paramsJsonStr)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() < 200 || res.Status() >= 300 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    int64 `json:"code"`
		Msg     string `json:"msg"`
		Data struct {
			Url       string `json:"url"`
		} `json:"data"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Data.Url,
		ChannelOrderNo: "",
	}

	return
}
