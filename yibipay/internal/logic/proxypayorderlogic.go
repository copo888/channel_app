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
	"github.com/copo888/channel_app/yibipay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"

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

	aesKey := "qHp8VxRtzQ7HpBfE"
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
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
	var currency string
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	//notifyUrl := "http://28fa-211-75-36-190.ngrok.io/api/api/proxy-pay-call-back"
	if strings.EqualFold("USDT", channel.CurrencyCode) { //1 USDT  -1 CNY
		currency = "1"
	}

	dataInit := &Data{
		Amount:           transactionAmount,
		CallBackUrl:      l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		CallToken:        randomID,
		Currency:         currency, //1 USDT  -1 CNY
		MerchantCode:     channel.MerId,
		Timestamp:        timeStamp,
		UserCode:         channel.MerId,
		WalletAddress:    req.ReceiptAccountNumber,
		WithdrawOrderId:  req.OrderNo,
		WithdrawQueryUrl: "http://bedf-211-75-36-190.ngrok.io",
	}
	dataBytes, err := json.Marshal(dataInit)
	if err != nil {
		logx.Errorf("序列化失败: %s", err.Error())
	}
	params := utils.EnPwdCode(string(dataBytes), aesKey)
	sign := payutils.SortAndSignSHA256FromObj(dataInit, channel.MerKey)
	logx.WithContext(l.ctx).Infof("加签原串:%s，Encryption: %s，Signature: %s", string(dataBytes)+channel.MerKey, params, sign)

	data := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params"`    //参数密文
		Signature    string `json:"signature"` //参数签名(params + md5key)
	}{
		MerchantCode: channel.MerId,
		Params:       params,
		Signature:    sign,
	}

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
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).JSON(data)

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
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params,optional"`
		Sign         string `json:"signature"`
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

	paramsDecode := utils.DePwdCode(channelResp.Params, aesKey)
	logx.WithContext(l.ctx).Infof("paramsDecode: %s", paramsDecode)
	channelResp2 := struct {
		Code string `json:"code"`
		Data struct {
			Success bool `json:"success"`
		} `json:"data,optional"`
		Message string `json:"message,optional"`
	}{}
	if err = json.Unmarshal([]byte(paramsDecode), &channelResp2); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", channelResp2)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if channelResp2.Code != "200" || channelResp2.Data.Success != true {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp2.Message, channelResp2.Data)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp2.Message)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: "CHN_" + req.OrderNo,
		OrderStatus:    "",
	}
	return resp, nil
}

type Data struct {
	Amount           string `json:"amount"`
	CallBackUrl      string `json:"callBackUrl"`
	CallToken        string `json:"callToken"`
	Currency         string `json:"currency"`
	MerchantCode     string `json:"merchantCode"`
	MerchantId       string `json:"merchantId"`
	Remark           string `json:"remark"`
	Timestamp        string `json:"timestamp"`
	UserCode         string `json:"userCode"`
	WalletAddress    string `json:"walletAddress"`
	WithdrawOrderId  string `json:"withdrawOrderId"` //商户生成的唯一提款请求订单号
	WithdrawQueryUrl string `json:"withdrawQueryUrl"`
}
