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
	"github.com/copo888/channel_app/indiapay/internal/payutils"
	"github.com/copo888/channel_app/indiapay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"

	"github.com/copo888/channel_app/indiapay/internal/svc"
	"github.com/copo888/channel_app/indiapay/internal/types"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s,orderNo: %s, ProxyPayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	if req.Remark == "" {
		logx.WithContext(l.ctx).Errorf("remark不可为空 remark:%s", req.Remark)
		return nil, errorx.New(responsex.INVALID_PARAMETER, "IFSC is required")
	}

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
	floatNum, err := strconv.ParseFloat(req.TransactionAmount, 64)
	if err != nil {
		fmt.Println("Error converting string to float:", err)
		return nil, err
	}

	data := url.Values{}
	data.Set("mer_id", channel.MerId)
	data.Set("order_id", req.OrderNo)
	data.Set("amount", fmt.Sprintf("%.0f", floatNum)) //整數，不要小數
	data.Set("notify", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	data.Set("gateway", "1") //1 : 銀行卡 2: UPI
	data.Set("user", req.ReceiptAccountName)
	data.Set("account", req.ReceiptAccountNumber)
	data.Set("bank_code", req.Remark) //填入IFSC channelBankMap.MapCode
	data.Set("bank", req.ReceiptCardBankName)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("sign", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		ChannelCode: channel.Code,
		OrderNo:     req.OrderNo,
		LogType:     constants.DATA_REQUEST_CHANNEL,
		LogSource:   constants.API_DF,
		Content:     fmt.Sprintf("%+v", data),
		TraceId:     l.traceID,
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
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		//不管渠道网路错误或者其他错误，一律不反失败单，持续处于待处理，直到等渠道回调成功或失败，或者手动回调
		resp := &types.ProxyPayOrderResponse{
			ChannelOrderNo: "CHN_" + req.OrderNo,
			OrderStatus:    "",
		}
		return resp, nil
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo: req.MerchantId,
			//MerchantOrderNo: req.OrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          string(ChannelResp.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(ChannelResp.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code      int         `json:"code, optional"`
		Data      interface{} `json:"data, optional"`
		Message   string      `json:"message, optional"`
		OrderCode string      `json:"order_code, optional"`
		Amount    string      `json:"amount, optional"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	if strings.Index(channelResp.Message, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %d: %s", channelResp.Code, channelResp.Message)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Message)
	} else if channelResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %d: %s", channelResp.Code, channelResp.Message)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo: req.MerchantId,
			//MerchantOrderNo: req.OrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          fmt.Sprintf("%+v", channelResp),
			TraceId:          l.traceID,
			ChannelErrorCode: fmt.Sprintf("%d", channelResp.Code),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		ChannelCode: channel.Code,
		OrderNo:     req.OrderNo,
		LogType:     constants.RESPONSE_FROM_CHANNEL,
		LogSource:   constants.API_DF,
		Content:     fmt.Sprintf("%+v", channelResp),
		TraceId:     l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.OrderCode,
		OrderStatus:    "",
	}

	return resp, nil
}
