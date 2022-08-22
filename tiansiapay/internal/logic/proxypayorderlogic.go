package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"

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
	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	randomIp := utils.GetRandomIp()
	payAmount, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	deviceId := utils.GetRandomString(16, 0, 0)
	// 組請求參數 FOR JSON
	paramsStruct := struct {
		UserName        string  `json:"userName"`
		DeviceType      int64   `json:"deviceType"`
		DeviceId        string  `json:"deviceId"`
		LoginIp         string  `json:"loginIp"`
		MerchantOrderId string  `json:"merchantOrderId"`
		OrderType       int64   `json:"orderType"`
		NotifyUrl       string  `json:"notifyUrl"`
		PayAmount       float64 `json:"payAmount"`
		BankCode        string  `json:"bankCode"`
		BankNum         string  `json:"bankNum"`
		BankOwner       string  `json:"bankOwner"`
		BankAddress     string  `json:"bankAddress"`
	}{
		UserName:        req.PlayerId,
		DeviceType:      9,
		DeviceId:        deviceId,
		LoginIp:         randomIp,
		MerchantOrderId: req.OrderNo,
		OrderType:       0,
		PayAmount:       payAmount,
		NotifyUrl:       notifyUrl,
		BankCode:        channelBankMap.MapCode,
		BankNum:         req.ReceiptAccountNumber,
		BankOwner:       req.ReceiptAccountName,
		BankAddress:     req.ReceiptCardCity,
	}

	paramsJson, _ := json.Marshal(paramsStruct)
	paramsJsonStr := string(paramsJson[:])

	_, params := payutils.AesEncrypt(paramsJsonStr, l.svcCtx.Config.AesKey)

	merchantNo, _ := strconv.ParseInt(channel.MerId, 10, 64)

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo int64  `json:"merchantNo"`
		Signature  string `json:"signature"`
		Params     string `json:"params"`
	}{
		MerchantNo: merchantNo,
		Params:     params,
		Signature:  payutils.Md5V(paramsJsonStr+channel.MerKey, l.ctx),
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,代付請求參數:%+v,代付原始參數:%s", channel.PayUrl, data, paramsJsonStr)

	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() < 200 || ChannelResp.Status() >= 300 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderId   string  `json:"orderId"`
			PayAmount float64 `json:"payAmount"`
			BankCode  string  `json:"bankCode"`
		} `json:"data"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Code != 200 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.OrderId,
		OrderStatus:    "",
	}

	return resp, nil
}
