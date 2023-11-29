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
	"github.com/copo888/channel_app/jyuhueipay/internal/payutils"
	"github.com/copo888/channel_app/jyuhueipay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/jyuhueipay/internal/svc"
	"github.com/copo888/channel_app/jyuhueipay/internal/types"

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
	//channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	//if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
	//	logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	//} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
	//	logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	//}
	// 組請求參數
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	transactionAmount := strconv.FormatFloat(amount, 'f', 0, 64)
	timestamp := time.Now().Format("20060102150405")

	data := url.Values{}
	data.Set("charset", "1")
	data.Set("language", "1")
	data.Set("version", "v1.0")
	data.Set("signType", "3")
	data.Set("timestamp", timestamp)

	data.Set("accessType", "1")
	data.Set("merchantId", channel.MerId)
	data.Set("merchantNme", channel.MerId)
	data.Set("orderNo", req.OrderNo)
	//data.Set("bankNo", channelBankMap.MapCode)
	data.Set("bankNo", "104100000004")
	data.Set("acctNo", req.ReceiptAccountNumber)
	data.Set("acctName", url.QueryEscape(req.ReceiptAccountName))
	data.Set("acctType", "0")
	data.Set("acctPhone", "00000000000")
	data.Set("ccy", "CNY")
	data.Set("transAmt", transactionAmount)
	data.Set("transUsage", "proxyPay")
	data.Set("notifyUrl", url.QueryEscape(l.svcCtx.Config.Server+"/api/proxy-pay-call-back"))

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, l.svcCtx.PrivateKey, l.ctx)
	data.Set("signMsg", sign)
	//data.Set("acctNo", req.ReceiptAccountNumber)
	data.Set("acctName", req.ReceiptAccountName)
	data.Set("notifyUrl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		ChannelCode: channel.Code,
		OrderNo:     req.OrderNo,
		LogType:     constants.DATA_REQUEST_CHANNEL,
		LogSource:   constants.API_DF,
		Content:     fmt.Sprintf("%+v", data),
		TraceId:     l.traceID,
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

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo: req.MerchantId,
			//MerchantOrderNo: req.OrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          string(ChannelResp.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(ChannelResp.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Ccy         string  `json:"ccy,optional"`
		AcctNo      string  `json:"acctNo,optional"`
		RspMsg      string  `json:"rspMsg,optional"`
		SignType    string  `json:"signType,optional"`
		AcctType    string  `json:"acctType,optional"`
		TransStatus string  `json:"transStatus,optional"`
		Version     string  `json:"version,optional"`
		Timestamp   string  `json:"timestamp,optional"`
		BankNo      string  `json:"bankNo,optional"`
		PayFee      float64 `json:"payFee,optional"`
		AcctName    string  `json:"acctName,optional"`
		TransTime   string  `json:"transTime,optional"`
		MerchantId  string  `json:"merchantId,optional"`
		NotifyUrl   string  `json:"notifyUrl,optional"`
		OrderId     string  `json:"orderId,optional"`
		TransUsage  string  `json:"transUsage,optional"`
		MerchantNme string  `json:"merchantNme,optional"`
		Charset     string  `json:"charset,optional"`
		TransAmt    float64 `json:"transAmt,optional"`
		RspCod      string  `json:"rspCod,optional"`
		AcctPhone   string  `json:"acctPhone,optional"`
		OrderNo     string  `json:"orderNo,optional"`
		SignMsg     string  `json:"signMsg,optional"`
		Language    string  `json:"language,optional"`
		AccessType  string  `json:"accessType,optional"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	if strings.Index(channelResp.RspMsg, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.RspCod, channelResp.RspMsg)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.RspMsg)
	} else if channelResp.RspCod != "01000000" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.RspCod, channelResp.RspMsg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo: req.MerchantId,
			//MerchantOrderNo: req.OrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          fmt.Sprintf("%+v", channelResp),
			TraceId:          l.traceID,
			ChannelErrorCode: channelResp.RspCod,
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RspMsg)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		ChannelCode: channel.Code,
		OrderNo:     req.OrderNo,
		LogType:     constants.RESPONSE_FROM_CHANNEL,
		LogSource:   constants.API_DF,
		Content:     fmt.Sprintf("%+v", channelResp),
		TraceId:     l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.OrderId,
		OrderStatus:    "",
	}

	return resp, nil
}
