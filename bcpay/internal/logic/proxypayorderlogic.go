package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/bcpay/internal/payutils"
	"github.com/copo888/channel_app/bcpay/internal/service"
	"github.com/copo888/channel_app/bcpay/internal/svc"
	"github.com/copo888/channel_app/bcpay/internal/types"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s,orderNo: %s, ProxyPayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	if len(req.PlayerId) == 0 {
		logx.WithContext(l.ctx).Errorf("PlayerId不可为空 PlayerId:%s", req.PlayerId)
		return nil, errorx.New(responsex.INVALID_PLAYER_ID)
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

	//请求渠道API
	var fiatAmount string
	if req.Currency == "USDT" {
		fiatAmount = req.TransactionAmount
	} else {
		var fiatAmountF float64
		var rateErr error
		if fiatAmountF, rateErr = payutils.GetCryptoRate(&types.ExchangeInfo{
			Url:          channel.PayUrl,
			Currency:     "USDT",       //
			Token:        req.Currency, //商户
			CryptoAmount: req.TransactionAmount,
		}, &l.ctx); rateErr != nil {
			return nil, errorx.New(responsex.INVALID_PARAMETER, rateErr.Error())
		}
		fiatAmount = strconv.FormatFloat(fiatAmountF, 'f', -1, 64)
	}

	data := struct {
		Command          string `json:"command"`
		HashCode         string `json:"hashCode"`
		TxId             string `json:"txid, optional"`
		Amount           string `json:"amount"`
		Currency         string `json:"currency"`
		WithdrawalAmount string `json:"withdrawal_amount"` //加密货币金额
		WithdrawalToken  string `json:"withdrawal_token"`  //加密货币
		Address          string `json:"address"`           //取款的加密货币钱包地址
		CallbackUrl      string `json:"callback_url"`
		CustomerUid      string `json:"customer_uid"` //客户独特编号
	}{
		Command:          "partner_withdraw",
		HashCode:         payutils.GetSign("partner_withdraw" + channel.MerKey),
		TxId:             req.OrderNo,
		Amount:           fiatAmount, //这里要依照法币数额去换crypto, //法币金额
		Currency:         "TRC20-USDT",
		WithdrawalAmount: req.TransactionAmount, //加密金额
		WithdrawalToken:  channel.CurrencyCode,
		Address:          req.ReceiptAccountNumber,
		CallbackUrl:      l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		CustomerUid:      req.PlayerId,
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
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
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjYxYWRhNzM3ZWNmMzIwMjE3ZmVlYzUyZDIzNDgyNTkwOTI1YjAyNjI2YzY1MjAwMDk4ODc1ZmY2NzI2N2FkMjdkNGY3MmQ5NmVkYzgzNDY4In0.eyJhdWQiOiIxIiwianRpIjoiNjFhZGE3MzdlY2YzMjAyMTdmZWVjNTJkMjM0ODI1OTA5MjViMDI2MjZjNjUyMDAwOTg4NzVmZjY3MjY3YWQyN2Q0ZjcyZDk2ZWRjODM0NjgiLCJpYXQiOjE3MjA2ODcxMjQsIm5iZiI6MTcyMDY4NzEyNCwiZXhwIjoxNzIxMjkxOTI0LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.loWg5juLro4FtuwOY-5ui_D1CQ_IcCZjOo3pYE-3cEcZSTbuQ9WCoQfJvL7daofBsBh8CkiHeVtQRx3S9QEUD_edVv5J83uHpyCJaMN3wvIE8K1DbBbhbWEK6WHl46bLHr4Akj-wZzUd8cx10OXBZFq6v5uZiQ73V-GJqP3NhufcXU1p10KbCSwsYiyqNd8F6-4p6Bmg5YucriQ5jM7KXxkTwBb09RNf4B7f2p2_QCw8YpbcwM6IDhdslUHwgRmHwf1fxESvw4im6Vjd6yfVcyjnex9jlItv_dkibvVd5Z-iTDz4_DzM8y1OlXiqRJD55dj0Y6gl5mCDb7RrZIpDC9NHuzdcL0GfetYLEM0hWazBAULPypRsJ79a2RhEvfopoPgkntv10mQmQ1U3X9vo3wRDZoUqfWdiQ2Xy-1kW0Cdg7CM2bSQExmKvdGxj1CVB8fSqFZqBVP4vrXzdQ40hS29rPoEZTg2VqDRK8AKgQjSFr4USaiq-bMq680Ok_y3kaadmEII86o8JtuBeDVKIdYImBsN8QNfIUezoAPgEmFvTGU9fDw4J69CaFWp9anTDCSCKNuRmcs7cZPK7Kz3WVjN35eMTInP2GTDCX-H8tYuzXESKIIrliONkYqGGU4JJ9Lhb11WMg3NJUpTQ3HPFm3tyCgrgMA8--QI0XkLfp50").
		Header("Content-type", "application/json").
		JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo:  req.MerchantId,
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

		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() > 300 || res.Status() < 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			//MerchantNo: req.MerchantId,
			//MerchantOrderNo: req.OrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          string(res.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(res.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d，%s", res.Status(), string(res.Body())))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Message string `json:"message"`
		Txid    string `json:"txid"`
		Amount  string `json:"amount"`
		Token   string `json:"token"`
		Status  string `json:"status"`
		Warning string `json:"warning"`
	}{}

	if err := res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	if strings.Index(channelResp.Message, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.Status, channelResp.Message)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.Message)
	} else if res.Status() >= 300 || res.Status() < 200 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %d: %s", res.Status(), channelResp.Message)

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
			ChannelErrorCode: fmt.Sprintf("%d", res.Status()),
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
		ChannelOrderNo: "CNH_" + req.OrderNo,
		OrderStatus:    "",
	}

	return resp, nil
}
