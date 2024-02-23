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
	"github.com/copo888/channel_app/jxwpay/internal/payutils"
	"github.com/copo888/channel_app/jxwpay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/jxwpay/internal/svc"
	"github.com/copo888/channel_app/jxwpay/internal/types"

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
	timestamp := time.Now().Format("20060102150405")
	ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//notifyUrl = "https://f599-211-75-36-190.jp.ngrok.io/api/proxy-pay-call-back"

	headMap := make(map[string]string)
	headMap["sid"] = channel.MerId
	headMap["timestamp"] = timestamp
	headMap["nonce"] = randomID
	headMap["url"] = "/payfor/trans"

	data := url.Values{}
	data.Set("out_trade_no", req.OrderNo)
	data.Set("amount", transactionAmount)
	data.Set("notify_url", notifyUrl)
	data.Set("bank_name", channelBankMap.MapCode)
	data.Set("sub_bank", req.ReceiptCardBranch)
	data.Set("bank_province", req.ReceiptCardProvince)
	data.Set("bank_city", req.ReceiptCardCity)
	data.Set("card_no", req.ReceiptAccountNumber)
	data.Set("bank_account", req.ReceiptAccountName)
	data.Set("currency", "CNY")
	data.Set("send_ip", ip)
	data.Set("attach", "")

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
	headSource := payutils.JoinStringsInASCII(headMap, "", false, false, "")
	bodyDataMap := payutils.CovertUrlValuesToMap(data)
	bodySource := payutils.JoinStringsInASCII(bodyDataMap, "", false, true, "")
	newSource := headSource + bodySource + channel.MerKey
	newSign := payutils.GetSign(newSource)
	logx.WithContext(l.ctx).Info("加签参数: ", newSource)
	logx.WithContext(l.ctx).Info("签名字串: ", newSign)
	headMap["sign"] = newSign
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
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Headers(headMap).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", ChannelResp.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code       string `json:"code"`
		Msg        string `json:"msg, optional"`
		TradeNo    string `json:"trade_no, optional"`
		OutTradeNo string `json:"out_trade_no, optional"`
		Sign       string `json:"sign, optional"`
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

	if strings.Index(channelResp.Msg, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Msg)
	} else if channelResp.Code != "1000" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.TradeNo,
		OrderStatus:    "",
	}

	return resp, nil
}