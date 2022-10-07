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
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"

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
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	orderTime := time.Now().Format("2006-01-02 15:04:05")
	nonce := utils.GetRandomString(10, utils.ALL, utils.MIX)

	dataInit := struct {
		MerchId        string  `json:"merchantId"`
		UserId         string  `json:"userId"`
		OrderId        string  `json:"orderId"`
		OrderTime      string  `json:"orderTime"`
		Amount         float64 `json:"amount"`
		BankCardNo     string  `json:"account"`
		BankCardholder string  `json:"holder"`
		BankCode       string  `json:"bank"`
		BankName       string  `json:"bankName"`
		NotifyUrl      string  `json:"notifyUrl"`
		Nonce          string  `json:"nonce"`
		Timestamp      string  `json:"timestamp"`
	}{
		MerchId:        channel.MerId,
		UserId:         channel.MerId,
		OrderId:        req.OrderNo,
		OrderTime:      orderTime,
		Amount:         amountFloat,
		BankCardNo:     req.ReceiptAccountNumber,
		BankCardholder: req.ReceiptAccountName,
		BankCode:       channelBankMap.MapCode,
		BankName:       req.ReceiptCardBankName,
		NotifyUrl:      l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		Nonce:          nonce,
		Timestamp:      timestamp,
	}
	dataBytes, err := json.Marshal(dataInit)

	encryptContent := utils.EnPwdCode(string(dataBytes), channel.MerKey)
	// 組請求參數 FOR JSON
	reqObj := struct {
		Id   string `json:"id"`
		Data string `json:"data"`
	}{
		Id:   channel.MerId,
		Data: encryptContent,
	}
	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", reqObj)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%#v", channel.ProxyPayUrl, reqObj)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).JSON(reqObj)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	response := utils.DePwdCode(string(res.Body()), channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		TradeId string `json:"transId"` //渠道訂單號
	}{}

	if err = json.Unmarshal([]byte(response), &channelResp); err != nil {
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

	if channelResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.TradeId,
		OrderStatus:    "",
	}

	return resp, nil
}
