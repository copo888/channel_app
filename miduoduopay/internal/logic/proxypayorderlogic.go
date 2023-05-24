package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/miduoduopay/internal/payutils"
	"github.com/copo888/channel_app/miduoduopay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/miduoduopay/internal/svc"
	"github.com/copo888/channel_app/miduoduopay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s,orderNo: %s, ProxyPayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
		logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
		logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	}
	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	notifyUrl := l.svcCtx.Config.Server+"/api/proxy-pay-call-back"
	//notifyUrl = "https://b345-211-75-36-190.ngrok-free.app/api/proxy-pay-call-back"
	timestamp := time.Now().Format("20060102150405")

	if amountFloat != math.Trunc(amountFloat) {
		logx.WithContext(l.ctx).Errorf("金额必须为整数, 请求金额:%s", req.TransactionAmount)
		return nil, errorx.New(responsex.INVALID_AMOUNT)
	}
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 0, 64)

	data := url.Values{}
	data.Set("MERCHANT_ID", channel.MerId)
	data.Set("BANK_ACCOUNT_NAME", req.ReceiptAccountName)
	data.Set("BANK_ACCOUNT_NO", req.ReceiptAccountNumber)
	data.Set("BANK_CODE", channelBankMap.MapCode)
	data.Set("NOTIFY_URL", notifyUrl)
	data.Set("PAY_TYPE", "AP001")
	data.Set("SUBMIT_TIME", timestamp)
	data.Set("TRANSACTION_NUMBER", req.OrderNo)
	data.Set("TRANSACTION_AMOUNT", transactionAmount)
	data.Set("VERSION", "1")

	//data := struct {
	//	MerchantId string `json:"MERCHANT_ID"`
	//	BankAccountName string `json:"BANK_ACCOUNT_NAME"`
	//	BankAccountNo string `json:"BANK_ACCOUNT_NO"`
	//	BankCode string `json:"BANK_CODE"`
	//	NotifyUrl string `json:"NOTIFY_URL"`
	//	PayType string `json:"PAY_TYPE"`
	//	SubmitTime string `json:"SUBMIT_TIME"`
	//	TransactionNumber string `json:"TRANSACTION_NUMBER"`
	//	TransactionAmount string `json:"TRANSACTION_AMOUNT"`
	//	Version string `json:"VERSION"`
	//	SignedMsg string `json:"SIGNED_MSG"`
	//}{
	//	MerchantId: channel.MerId,
	//	Version: "1",
	//	TransactionNumber: req.OrderNo,
	//	TransactionAmount:  req.TransactionAmount,
	//	NotifyUrl: notifyUrl,
	//	BankCode: channelBankMap.MapCode,
	//	BankAccountNo: req.ReceiptAccountNumber,
	//	BankAccountName: req.ReceiptAccountName,
	//	PayType: "AP001",
	//	SubmitTime: timestamp,
	//
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("SIGNED_MSG", sign)
	//data.SignedMsg = sign
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", data),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%+v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code string `json:"code"`
		Message string `json:"message"`
		Data struct{
			TransactionId string `json:"transactionId"`
			TransactionAmount float64 `json:"transactionAmount"`
			MerchantId string `json:"merchantId"`
			PayType string `json:"payType"`
			Postscript string `json:"postscript"`
		} `json:"data"`
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
		Content:   fmt.Sprintf("%+v", channelResp),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if strings.Index(channelResp.Message, "NOT_ENOUGH_AVAILABLE_BALANCE") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Code, channelResp.Message)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Message)
	} else if channelResp.Code != "G_00005" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Code, channelResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.TransactionId,
		OrderStatus:    "",
	}

	return resp, nil
}
