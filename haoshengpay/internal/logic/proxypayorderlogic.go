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
	"github.com/copo888/channel_app/haoshengpay/internal/payutils"
	"github.com/copo888/channel_app/haoshengpay/internal/service"
	"github.com/copo888/channel_app/haoshengpay/internal/svc"
	"github.com/copo888/channel_app/haoshengpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	amountInt := int(amount)
	//timestamp := time.Now().Format("20060102150405")

	data := url.Values{}
	// 組請求參數 FOR JSON
	dataJs := struct {
		MchId           string `json:"mchId"`
		AppId           string `json:"appId"`
		MchTransOrderNo string `json:"mchTransOrderNo"`
		Currency        string `json:"currency"`
		Amount          string `json:"amount"`
		NotifyUrl       string `json:"notifyUrl"`
		BankCode        string `json:"bankCode"`
		BankName        string `json:"bankName"`
		AccountType     string `json:"accountType"`
		AccountName     string `json:"accountName"`
		AccountNo       string `json:"accountNo"`
		Province        string `json:"province"`
		City            string `json:"city"`
		TransVerifyCode string `json:"transVerifyCode"`
		Sign            string `json:"sign"`
	}{
		MchId:           channel.MerId,
		AppId:           "4672565e345b483583eaee7e0b51ffa2",
		MchTransOrderNo: req.OrderNo,
		Currency:        "cny",
		Amount:          fmt.Sprintf("%d", amountInt),
		NotifyUrl:       l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		//NotifyUrl:       "http://19218-host-header=localhost:19218/api/proxy-pay-call-back",
		BankCode: channelBankMap.MapCode,
		//BankCode:        "HPT00001",
		BankName: req.ReceiptCardBankName,
		//BankName:        "中国银行",
		AccountType:     "1",
		AccountName:     req.ReceiptAccountName,
		AccountNo:       req.ReceiptAccountNumber,
		Province:        req.ReceiptCardProvince,
		City:            req.ReceiptCardCity,
		TransVerifyCode: "r7HxdVrXJ7Rc",
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(dataJs, channel.MerKey, l.ctx)
	dataJs.Sign = sign
	b, err := json.Marshal(dataJs)
	if err != nil {
		fmt.Println("error:", err)
	}
	data.Set("params", string(b))

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
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
		RetCode         string  `json:"retCode"`
		RetMsg          string  `json:"retMsg, optional"`
		TransOrderId    string  `json:"transOrderId, optional"`
		MchId           int64   `json:"mchId, optional"`
		MchTransOrderNo string  `json:"mchTransOrderNo, optional"`
		Amount          float64 `json:"amount, optional"`
		Status          int     `json:"status, optional"`
		ChannelOrderNo  string  `json:"channelOrderNo, optional"`
		CreateTime      int     `json:"createTime, optional"`
		Sign            string  `json:"sign, optional"`
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

	if strings.Index(channelResp.RetMsg, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.RetCode, channelResp.RetMsg)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.RetMsg)
	} else if channelResp.RetCode != "SUCCESS" {
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
