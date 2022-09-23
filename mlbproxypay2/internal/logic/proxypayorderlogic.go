package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/mlbproxypay2/internal/payutils"
	"github.com/copo888/channel_app/mlbproxypay2/internal/svc"
	"github.com/copo888/channel_app/mlbproxypay2/internal/types"
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

	// 組返回給backOffice 的代付返回物件(測試四)
	//return &types.ProxyPayOrderResponse{
	//	ChannelOrderNo: "TESTTRADEID_000011111",
	//	OrderStatus:    "",
	//}, nil

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)
	APP_SECRET := "tk_eTSdhXkaCAnbfskt6GA"

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
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 0, 64)

	data := url.Values{}
	data.Set("merchantNo", channel.MerId)
	data.Set("orderNo", req.OrderNo)
	data.Set("amount", transactionAmount)
	data.Set("name", req.ReceiptAccountName)
	data.Set("bankName", channelBankMap.MapCode)
	data.Set("bankAccount", req.ReceiptAccountNumber)
	data.Set("datetime", utils.GetDateTimeSring(utils.YYYYMMddHHmmss2))
	data.Set("notifyUrl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	data.Set("time", strconv.FormatInt(utils.GetCurrentMilliSec(), 10))
	data.Set("extra", "")
	data.Set("reverseUrl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	data.Set("mobile", "")

	// 加簽
	sign := payutils.SortAndSignFromUrlValues_SHA256(data, channel.MerKey)
	data.Set("sign", sign)
	data.Set("appSecret", APP_SECRET)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,代付請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    int     `json:"code"`
		Msg     string  `json:"text"`
		TradeId string  `json:"tradeNo"` //渠道訂單號
		OrderNo string  `json:"orderNo"` //訂單號
		Amount  float64 `json:"amount"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if strings.Index(channelResp.Msg, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Msg)
	} else if channelResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.TradeId,
		OrderStatus:    "",
	}

	return resp, nil
}
