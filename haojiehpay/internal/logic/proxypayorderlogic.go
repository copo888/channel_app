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
	"github.com/copo888/channel_app/haojiehpay/internal/payutils"
	"github.com/copo888/channel_app/haojiehpay/internal/svc"
	"github.com/copo888/channel_app/haojiehpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %#v", l.svcCtx.Config.ProjectName, req)

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
	//amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//notifyUrl = "https://dc98-211-75-36-190.jp.ngrok.io/api/proxy-pay-call-back"
	proxyKey := "ffvaaBVvLFDW" // 代付验证码
	appId := "145fd6be4e5b4187839447579d70a984"
	//transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	data := url.Values{}
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

	//mchId, _ := strconv.Atoi(channel.MerId)
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	amountInt := int(amount)
	// 組請求參數 FOR JSON
	dataJs := struct {
		MchId string `json:"mchId"`
		AppId string `json:"appId"`
		MchTransOrderNo string `json:"mchTransOrderNo"`
		Currency string `json:"currency"`
		Amount string `json:"amount"`
		NotifyUrl string `json:"notifyUrl"`
		BankCode string `json:"bankCode"`
		BankName string `json:"bankName"`
		AccountType string `json:"accountType"`
		AccountName string `json:"accountName"`
		AccountNo string `json:"accountNo"`
		Province string `json:"province"`
		City  string `json:"city"`
		Param2 string `json:"param2"`
		Sign  string `json:"sign"`
	}{
		MchId: channel.MerId,
		AppId: appId,
		MchTransOrderNo: req.OrderNo,
		Currency: "cny",
		Amount: fmt.Sprintf("%d", amountInt),
		NotifyUrl: notifyUrl,
		BankCode: req.ReceiptCardBankCode,
		BankName: req.ReceiptCardBankName,
		AccountType: "1",
		AccountName: req.ReceiptAccountName,
		AccountNo: req.ReceiptAccountNumber,
		Province: req.ReceiptCardProvince,
		City: req.ReceiptCardCity,
		Param2: proxyKey,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(dataJs, channel.MerKey)
	dataJs.Sign = sign
	b, err := json.Marshal(dataJs)
	if err != nil {
		fmt.Println("error:", err)
	}
	data.Set("params", string(b))

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
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		RetCode string `json:"retCode"`
		RetMsg string `json:"retMsg, optional"`
		TransOrderId string `json:"transOrderId, optional"`
		MchId int64 `json:"mchId, optional"`
		MchTransOrderNo string `json:"mchTransOrderNo, optional"`
		Amount float64 `json:"amount, optional"`
		Status int `json:"status, optional"`
		ChannelOrderNo string `json:"channelOrderNo, optional"`
		CreateTime int `json:"createTime, optional"`
		Sign string `json:"sign, optional"`
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

	if channelResp.RetCode != "SUCCESS" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.RetCode, channelResp.RetMsg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RetMsg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.TransOrderId,
		OrderStatus:    "",
	}

	return resp, nil
}
