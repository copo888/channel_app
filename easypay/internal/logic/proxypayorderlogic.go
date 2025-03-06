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
	"github.com/copo888/channel_app/easypay/internal/payutils"
	"github.com/copo888/channel_app/easypay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/easypay/internal/svc"
	"github.com/copo888/channel_app/easypay/internal/types"

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
	appSecret := "17c44b5028794d8c9fc3e79c13b985ee"
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
	//transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//notifyUrl = "https://e46c-211-75-36-190.jp.ngrok.io/api/proxy-pay-call-back"
	dateTime := time.Now().Format("2006-01-02 15:04:05")
	timestamp := time.Now().Unix()

	//data := url.Values{}
	//data.Set("partner", channel.MerId)
	//data.Set("service", "10201")
	//data.Set("tradeNo", req.OrderNo)
	//data.Set("amount", transactionAmount)
	//data.Set("notifyUrl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	//data.Set("bankCode", channelBankMap.MapCode)
	//data.Set("subsidiaryBank", req.ReceiptCardBankName)
	//data.Set("subbranch", req.ReceiptCardBranch)
	//data.Set("province", req.ReceiptCardProvince)
	//data.Set("city", req.ReceiptCardCity)
	//data.Set("bankCardNo", req.ReceiptAccountNumber)
	//data.Set("bankCardholder", req.ReceiptAccountName)

	data := struct {
		MerchantNo  string  `json:"merchantNo"`
		OrderNo     string  `json:"orderNo"`
		Amount      float64 `json:"amount"`
		Name        string  `json:"name"`
		BankName    string  `json:"bankName"`
		BankAccount string  `json:"bankAccount"`
		Mobile      string  `json:"mobile"`
		Datetime    string  `json:"datetime"`
		NotifyUrl   string  `json:"notifyUrl"`
		ReverseUrl  string  `json:"reverseUrl"`
		Extra       string  `json:"extra"`
		Time        int64   `json:"time"`
		AppSecret   string  `json:"appSecret"`
		Sign        string  `json:"sign"`
	}{
		MerchantNo:  channel.MerId,
		OrderNo:     req.OrderNo,
		Amount:      amountFloat,
		Name:        req.ReceiptAccountName,
		BankName:    channelBankMap.MapCode,
		BankAccount: req.ReceiptAccountNumber,
		Datetime:    dateTime,
		NotifyUrl:   notifyUrl,
		Time:        timestamp,
		AppSecret:   appSecret,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

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
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    int     `json:"code"`
		Text    string  `json:"text"`
		TradeNo string  `json:"tradeNo"`
		OrderNo string  `json:"orderNo"`
		Amount  float64 `json:"amount"`
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

	if strings.Index(channelResp.Text, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Code, channelResp.Text)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Text)
	} else if channelResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Code, channelResp.Text)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Text)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: string(channelResp.TradeNo),
		OrderStatus:    "",
	}

	return resp, nil
}
