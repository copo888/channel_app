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
	"github.com/copo888/channel_app/kumopay1881/internal/payutils"
	"github.com/copo888/channel_app/kumopay1881/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"

	"github.com/copo888/channel_app/kumopay1881/internal/svc"
	"github.com/copo888/channel_app/kumopay1881/internal/types"

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
	req.ReceiptAccountName = strings.TrimSpace(req.ReceiptAccountName)
	data := struct {
		OutTradeNo    string `json:"out_trade_no"`
		BankId        string `json:"bank_id"`
		BankOwner     string `json:"bank_owner"`
		AccountNumber string `json:"account_number"`
		Amount        string `json:"amount"`
		CallbackUrl   string `json:"callback_url"`
		Sign          string `json:"sign"`
	}{
		OutTradeNo:    req.OrderNo,
		BankId:        channelBankMap.MapCode,
		BankOwner:     req.ReceiptAccountName,
		AccountNumber: req.ReceiptAccountNumber,
		Amount:        transactionAmount,
		CallbackUrl:   l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey+channel.MerId)
	data.Sign = sign

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
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Header("Authorization", "Bearer "+channel.MerKey).
		Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		if strings.Index(ChnErr.Error(), "connection reset by peer") > -1 {

			logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
			msg := fmt.Sprintf("代付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
			service.CallLineSendURL(l.ctx, l.svcCtx, msg)

			//組返回給backOffice 的代付返回物件
			resp := &types.ProxyPayOrderResponse{
				ChannelOrderNo: "",
				OrderStatus:    "",
			}
			return resp, nil
		}
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
		Success bool        `json:"success"`
		Message string      `json:"message"`
		Errors  interface{} `json:"errors"`
		Data    struct {
			TradeNo       string  `json:"trade_no"`
			OutTradeNo    string  `json:"out_trade_no"`
			Amount        float64 `json:"amount"`
			BankId        string  `json:"bank_id"`
			AccountNumber string  `json:"account_number"`
			BankOwner     string  `json:"bank_owner"`
			CallbackUrl   string  `json:"callback_url"`
			State         string  `json:"state"`
		} `json:"data"`
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

	if channelResp.Success != true {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Success, channelResp.Message)
		message := channelResp.Message
		if channelResp.Errors != nil {
			message += fmt.Sprintf(": %s", channelResp.Errors)
		}
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, message)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.TradeNo,
		OrderStatus:    "",
	}

	return resp, nil
}
