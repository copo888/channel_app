package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/koreapay/internal/payutils"
	"github.com/copo888/channel_app/koreapay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strings"

	"github.com/copo888/channel_app/koreapay/internal/svc"
	"github.com/copo888/channel_app/koreapay/internal/types"

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
	//channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	//if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
	//	logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	//} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
	//	logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	//}
	// 組請求參數
	//amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	//transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"

	content := struct {
		MerchId         string `json:"merchantId"`
		OrderId         string `json:"transactionId"`
		TransactionType string `json:"transactionType"`
		Currency        string `json:"currency"`
		Money           string `json:"amount"`
		NotifyUrl       string `json:"callback"`
		BankName        string `json:"bankName"`
		BankAcctName    string `json:"bankAcctName"`
		BankAccNum      string `json:"bankAccNum"`
	}{
		MerchId:         channel.MerId,
		OrderId:         req.OrderNo,
		TransactionType: "W",
		Currency:        "KRW",
		Money:           req.TransactionAmount,
		NotifyUrl:       notifyUrl,
		BankName:        req.ReceiptCardBankName,
		BankAcctName:    req.ReceiptAccountName,
		BankAccNum:      req.ReceiptAccountNumber,
	}
	paramsJson, _ := json.Marshal(content)
	paramsJsonStr := string(paramsJson)
	// 加簽
	aesData := payutils.AESEncrypt(strings.ReplaceAll(paramsJsonStr, " ", ""), []byte(channel.MerKey), l.svcCtx.Config.Channel.Pass1, l.svcCtx.Config.Channel.Pass2)
	encryptedString := base64.StdEncoding.EncodeToString(aesData)

	// 組請求參數 FOR JSON
	data := struct {
		MerchId string `json:"merchantId"`
		Message string `json:"message"`
	}{
		MerchId: channel.MerId,
		Message: encryptedString,
	}
	logx.Infof("paramsJsonStr: %s , data.Message(Encrypted): %s", paramsJsonStr, data.Message)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: channel.MerId,
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
		Status         string `json:"status"`
		Errors         string `json:"errors,optional"`
		TransactionId  string `json:"transactionId,optional"`
		ReferenceId    string `json:"referenceId,optional"`
		ReferenceNo    int64  `json:"referenceNo,optional"`
		WithdrawAmount string `json:"withdraw_amount,optional"`
		Signed         string `json:"signed,optional"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if strings.Index(channelResp.Errors, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Status, channelResp.Errors)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Errors)
	} else if strings.EqualFold(channelResp.Status, "fail") {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Status, channelResp.Errors)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Errors)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.ReferenceId,
		OrderStatus:    "",
	}

	return resp, nil
}
