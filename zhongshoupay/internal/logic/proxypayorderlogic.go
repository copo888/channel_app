package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	_ "github.com/copo888/channel_app/common/model"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/zhongshoupay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"

	"github.com/copo888/channel_app/zhongshoupay/internal/svc"
	"github.com/copo888/channel_app/zhongshoupay/internal/types"

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

	//TODO 測試渠道返回
	testResp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: "ChannelOrderNoTEST",
		OrderStatus:    "",
	}
	return testResp, nil

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	if err2 != nil || channelBankMap.MapCode == "" {
		logx.Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, err2.Error(), "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName)
	}
	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	data := url.Values{}
	data.Set("partner", channel.MerId)
	data.Set("service", "10201")
	data.Set("tradeNo", req.OrderNo)
	data.Set("amount", transactionAmount)
	data.Set("notifyUrl", channel.ApiUrl+"/api/proxy-pay-call-back")
	data.Set("bankCode", channelBankMap.MapCode)
	data.Set("subsidiaryBank", req.ReceiptCardBankName)
	data.Set("subbranch", req.ReceiptCardBranch)
	data.Set("province", req.ReceiptCardProvince)
	data.Set("city", req.ReceiptCardCity)
	data.Set("bankCardNo", req.ReceiptAccountNumber)
	data.Set("bankCardholder", req.ReceiptAccountName)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	logx.Infof("代付下单请求地址:%s,代付請求參數:%#v", channel.ProxyPayUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).Form(data)
	logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	if ChnErr != nil {
		logx.Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}

	// 渠道回覆處理
	channelResp := struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		TradeId string `json:"tradeId"` //渠道訂單號
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.TradeId,
		OrderStatus:    "",
	}

	return resp, nil
}
