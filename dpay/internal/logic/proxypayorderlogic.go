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
	"github.com/copo888/channel_app/dpay/internal/payutils"
	"github.com/copo888/channel_app/dpay/internal/service"
	"github.com/copo888/channel_app/dpay/internal/svc"
	"github.com/copo888/channel_app/dpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

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
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	data := url.Values{}
	data.Set("cus_code", channel.MerId)
	data.Set("cus_order_sn", req.OrderNo)
	data.Set("payment_flag", "pay_webbk")
	data.Set("amount", transactionAmount)
	data.Set("notify_url", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	//data.Set("notify_url", "https://107c-211-75-36-190.jp.ngrok.io/api/proxy-pay-call-back")
	data.Set("bank_code", channelBankMap.MapCode)
	//data.Set("subsidiaryBank", req.ReceiptCardBankName)
	//data.Set("subbranch", req.ReceiptCardBranch)
	//data.Set("province", req.ReceiptCardProvince)
	//data.Set("city", req.ReceiptCardCity)
	data.Set("bank_account", req.ReceiptAccountNumber)
	data.Set("account_name", req.ReceiptAccountName)

	//data := struct {
	//	Partner string `json:"partner"`
	//	Service string `json:"service"`
	//	TradeNo string `json:"tradeNo"`
	//	Amount string `json:"amount"`
	//	NotifyUrl string `json:"notifyUrl"`
	//	BankCode string `json:"bankCode"`
	//	SubsidiaryBank string `json:"subsidiaryNank"`
	//	Subbranch string `json:"subbranch"`
	//	Province string `json:"province"`
	//	City string `json:"city"`
	//	BankCardNo string `json:"bankCardNo"`
	//	BankCardholder string `json:"bankCardHolder"`
	//	Sign string `json:"sign"`
	//}{
	//	Partner: channel.MerId,
	//	Service: "10201",
	//	TradeNo: req.OrderNo,
	//	Amount: transactionAmount,
	//	NotifyUrl: l.svcCtx.Config.Server+"/api/proxy-pay-call-back",
	//	BankCode: channelBankMap.MapCode,
	//	SubsidiaryBank: req.ReceiptCardBankName,
	//	Subbranch: req.ReceiptCardBranch,
	//	Province: req.ReceiptCardProvince,
	//	City: req.ReceiptCardCity,
	//	BankCardNo: req.ReceiptAccountNumber,
	//	BankCardholder: req.ReceiptAccountName,
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.Sign = sign

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
		Result  string `json:"result"`
		Status  int    `json:"status"`
		Message string `json:"message, optional"`
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
	if channelResp.Status == 410 || strings.Index(channelResp.Message, "代付金不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Status, channelResp.Message)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Message)
	} else if channelResp.Result != "success" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Result, channelResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	channelResp2 := struct {
		OrderInfo struct {
			OrderSn             string  `json:"order_sn"`
			CusOrderSn          string  `json:"cus_order_sn"`
			CurrencyType        string  `json:"currency_type"`
			OriginalAmount      float64 `json:"original_amount"`
			OrderAmount         float64 `json:"order_amount"`
			ExchangeAmount      float64 `json:"exchange_amount"`
			CusBeforeDeductions float64 `json:"cus_before_deductions"`
			CusAfterDeductions  float64 `json:"cus_after_deductions"`
		} `json:"order_info, optional"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp2); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp2.OrderInfo.OrderSn,
		OrderStatus:    "",
	}

	return resp, nil
}
