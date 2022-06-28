package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/vp8pay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/copo888/channel_app/vp8pay/internal/svc"
	"github.com/copo888/channel_app/vp8pay/internal/types"

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

	logx.Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	if err2 != nil || channelBankMap.MapCode == "" {
		logx.Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName)
	}
	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	// 組請求參數 FOR JSON
	data := struct {
		AccountNumber string `json:"account_number"`
		OutTradeNo    string `json:"out_trade_no"`
		BankId        string `json:"bank_id"`
		BankOwner     string `json:"bank_owner"`
		Amount        string `json:"amount"`
		CallbackUrl   string `json:"callback_url"`
		Sign          string `json:"sign"`
	}{
		AccountNumber: req.ReceiptAccountNumber,
		OutTradeNo:    req.OrderNo,
		BankId:        channelBankMap.MapCode,
		BankOwner:     req.ReceiptCardBankName,
		Amount:        transactionAmount,
		CallbackUrl:   l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey + "4PEKQ6viEaWxF8k1arBkVGOF4Xw0Ipp5rXUEF2jVnY7MeJHRFO32WcGUYKKq")
	data.Sign = sign

	// 請求渠道
	logx.Infof("代付下单请求地址:%s,請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).
		Header("Accept", "application/json").
		Header("Authorization", "Bearer "+channel.MerKey).
		Timeout(10).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			TradeNo    string `json:"trade_no"`
			OutTradeNo string `json:"out_trade_no"`
			Amount     float64 `json:"amount"`
		} `json:"data"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if !channelResp.Success {
		logx.Errorf("代付渠道返回错误: %s: %s", channelResp.Success, channelResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.TradeNo,
		OrderStatus:    "",
	}

	return resp, nil
}
