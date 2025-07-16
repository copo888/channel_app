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
	"github.com/copo888/channel_app/papayapay/internal/service"
	"github.com/copo888/channel_app/papayapay/internal/svc"
	"github.com/copo888/channel_app/papayapay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"

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

	data := struct {
		CurrencyCode            string  `json:"currencyCode"`
		FundOutPaymentReference string  `json:"fundOutPaymentReference"`
		FundOutDescription      string  `json:"fundOutDescription"`
		AccountName             string  `json:"accountName"`
		AccountNumber           string  `json:"accountNumber"`
		BankCode                string  `json:"bankCode"`
		Amount                  float64 `json:"amount"`
	}{
		CurrencyCode:            "THB",
		FundOutPaymentReference: req.OrderNo,
		FundOutDescription:      "payout",
		Amount:                  amountFloat,
		BankCode:                channelBankMap.MapCode,
		AccountNumber:           req.ReceiptAccountNumber,
		AccountName:             req.ReceiptAccountName,
	}

	// 加簽
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey,l.ctx)
	//data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_DF,
		Content:         fmt.Sprintf("%+v", data),
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%+v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).
		Headers(map[string]string{
			"Content-Type":     "application/json",
			"Accecpt":          "application/json",
			"transactiontoken": channel.MerKey,
		}).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: \n", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
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
	}
	//else if ChannelResp.Status() != 200 {
	//	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	//	msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
	//	service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
	//
	//	//寫入交易日志
	//	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
	//		MerchantNo:       req.MerchantId,
	//		MerchantOrderNo:  req.MerchantOrderNo,
	//		ChannelCode:      channel.Code,
	//		OrderNo:          req.OrderNo,
	//		LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
	//		LogSource:        constants.API_DF,
	//		Content:          string(ChannelResp.Body()),
	//		TraceId:          l.traceID,
	//		ChannelErrorCode: strconv.Itoa(ChannelResp.Status()),
	//	}); err != nil {
	//		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	//	}
	//
	//	return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d\n", ChannelResp.Status()))
	//}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		StatusCode int `json:"statusCode"`
		Data       struct {
			FundOutStatus string `json:"fundOutStatus"`
			Message       string `json:"message"`
			Data          struct {
				Type                    string      `json:"type"`
				CurrencyCode            string      `json:"currencyCode"`
				FundOutPaymentReference string      `json:"fundOutPaymentReference"`
				FundOutStatus           string      `json:"fundOutStatus"`
				AccountNumber           string      `json:"accountNumber"`
				BankCode                string      `json:"bankCode"`
				Amount                  int         `json:"amount"`
				TransactionRef1         string      `json:"transactionRef1"`
				ServiceFee              interface{} `json:"serviceFee"`
				IsPaid                  bool        `json:"isPaid"`
				FundOutCallbackStatus   string      `json:"fundOutCallbackStatus"`
				FundOutDescription      string      `json:"fundOutDescription"`
				CreatedDate             string      `json:"createdDate"`
				UpdatedDate             string      `json:"updatedDate"`
			} `json:"data"`
		} `json:"data"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	if strings.Index(channelResp.Data.Message, "Insufficient Balance") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s\n", channelResp.Data.FundOutStatus, channelResp.Data.Message)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Data.Message)
	} else if channelResp.StatusCode != 200 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s\n", channelResp.Data.FundOutStatus, channelResp.Data.Message)
		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			MerchantOrderNo:  req.MerchantOrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          fmt.Sprintf("%+v", channelResp),
			TraceId:          l.traceID,
			ChannelErrorCode: channelResp.Data.Message,
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s\n", err)
		}
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Data.Message)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.RESPONSE_FROM_CHANNEL,
		LogSource:       constants.API_DF,
		Content:         fmt.Sprintf("%+v", channelResp),
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s\n", err)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.Data.TransactionRef1,
		OrderStatus:    "",
	}

	return resp, nil
}
