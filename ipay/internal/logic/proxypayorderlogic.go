package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/ipay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/ipay/internal/svc"
	"github.com/copo888/channel_app/ipay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	timestamp := time.Now().Format(time.RFC3339)
	// 組請求參數 FOR JSON
	params := struct {
		AccountName      string `json:"account_name"`
		MerchantOrderId  string `json:"merchant_order_id"`
		TotalAmount      string `json:"total_amount"`
		Timestamp        string `json:"timestamp"`
		NotifyUrl        string `json:"notify_url"`
		BankName         string `json:"bank_name"`
		BankProvinceName string `json:"bank_province_name"`
		BankCityName     string `json:"bank_city_name"`
		BankAccouNo      string `json:"bank_account_no"`
		BankAccountType  string `json:"bank_account_type"`
		BankAccountName  string `json:"bank_account_name"`
	}{
		AccountName:      channel.MerId,
		MerchantOrderId:  req.OrderNo,
		TotalAmount:      req.TransactionAmount,
		Timestamp:        timestamp,
		NotifyUrl:        l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		BankName:         req.ReceiptCardBankName,
		BankProvinceName: req.ReceiptCardProvince,
		BankCityName:     req.ReceiptCardCity,
		BankAccouNo:      req.ReceiptAccountNumber,
		BankAccountType:  "corporate",
		BankAccountName:  req.ReceiptAccountName,
	}

	// 加簽
	paramsJson, _ := json.Marshal(params)
	signature := payutils.GetSign2(paramsJson, l.svcCtx.PrivateKey)
	// 組請求參數
	data := url.Values{}
	data.Set("data", string(paramsJson))
	data.Set("signature", signature)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() == 403 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, fmt.Sprintf("Error HTTP Status: %d, %s", ChannelResp.Status(), string(ChannelResp.Body())))
	} else if ChannelResp.Status() < 200 && ChannelResp.Status() >= 300 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d, %s", ChannelResp.Status(), string(ChannelResp.Body())))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		ID string `json:"id"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.ID,
		OrderStatus:    "",
	}

	return resp, nil
}
