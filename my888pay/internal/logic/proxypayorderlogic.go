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
	"github.com/copo888/channel_app/my888pay/internal/payutils"
	"github.com/copo888/channel_app/my888pay/internal/service"
	"github.com/copo888/channel_app/my888pay/internal/svc"
	"github.com/copo888/channel_app/my888pay/internal/types"
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
	channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
		logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
		logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	}

	if len(req.ReceiptAccountName) == 0 {
		logx.WithContext(l.ctx).Errorf("UserId不可为空 userId:%s", req.ReceiptAccountName)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	if len(req.PlayerId) == 0 {
		logx.WithContext(l.ctx).Errorf("PlayerId不可为空 playerId:%s", req.PlayerId)
		return nil, errorx.New(responsex.INVALID_PLAYER_ID)
	}

	if len(req.ReceiptAccountNumber) == 0 {
		logx.WithContext(l.ctx).Errorf("BankAccount不可为空 BankAccount:%s", req.ReceiptAccountNumber)
		return nil, errorx.New(responsex.INVALID_BANK_NO)
	}

	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	//transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	data := struct {
		PlatformOrderNumber string  `json:"platform_order_number"`
		Account             string  `json:"account"`
		Username            string  `json:"username"`
		Phone               string  `json:"phone"`
		RemittanceBank      string  `json:"remittance_bank"`
		RemittanceAccount   string  `json:"remittance_account"`
		Amount              float64 `json:"amount"`
		ApiKey              string  `json:"api_key"`
		CallbackUrl         string  `json:"callback_url"`
	}{
		PlatformOrderNumber: req.OrderNo,
		Account:             req.PlayerId,
		Username:            req.ReceiptAccountName,
		RemittanceBank:      channelBankMap.MapCode,
		RemittanceAccount:   req.ReceiptAccountNumber,
		Amount:              amountFloat,
		ApiKey:              channel.MerKey,
		CallbackUrl:         l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		Phone:               "01012345678", //占时写死，当渠道有规定要会员手机时
	}

	// 加簽
	// 将原始数据序列化为JSON字符串
	jsonData, err := json.Marshal(data)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("%+v", err)
	}
	// 解码密钥
	hashKey, hashErr := payutils.DecodeBase64Key(l.svcCtx.Config.HashKey)
	if hashErr != nil {
		logx.WithContext(l.ctx).Errorf("密钥解码失败:", err)
	}
	// 加簽
	encrypedData, errEnc := payutils.Encrypt(string(jsonData), hashKey)
	if errEnc != nil {
		logx.WithContext(l.ctx).Errorf("加密失败: %+v", errEnc)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_DF,
		Content:         fmt.Sprintf("%+v", data),
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%+v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).
		Header("Content-Type", "text/plain").
		Header("X-Api-Key", channel.MerKey).
		Timeout(20).Trace(span).Body([]byte(encrypedData))

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
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
	}
	//else if ChannelResp.Status() != 200 {
	//	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	//	msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
	//	service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
	//
	//	//寫入交易日志
	//	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
	//		MerchantNo:       req.MerchantId,
	//		MerchantOrderNo:  req.MerchantOrderNo,
	//		ChannelCode:      channel.Code,
	//		OrderNo:          req.OrderNo,
	//		LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
	//		LogSource:        constants.API_DF,
	//		Content:          string(ChannelResp.Body()),
	//		TraceId:          l.traceID,
	//		ChannelErrorCode: strconv.Itoa(ChannelResp.Status()),
	//	}); err != nil {
	//		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	//	}
	//
	//	return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	//}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code int `json:"code"`
		Data struct {
			OrderNumber         string  `json:"order_number"`
			PlatformOrderNumber string  `json:"platform_order_number"`
			Status              int     `json:"status"`
			Amount              float64 `json:"amount"`
			Account             string  `json:"account"`
			Username            string  `json:"username"`
			Bank                string  `json:"bank"`
			BankAccount         string  `json:"bank_account"`
			Url                 string  `json:"url"`
		} `json:"data"`
		ErrorText string `json:"error_text, optional"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	if strings.Index(channelResp.ErrorText, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %d: %s", channelResp.Code, channelResp.ErrorText)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.ErrorText)
	} else if channelResp.Code != 200 || ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %d: %s", channelResp.Code, channelResp.ErrorText)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			MerchantOrderNo:  req.MerchantOrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          channelResp.ErrorText,
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(channelResp.Code),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorText)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.RESPONSE_FROM_CHANNEL,
		LogSource:       constants.API_DF,
		Content:         fmt.Sprintf("%+v", channelResp),
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.OrderNumber,
		OrderStatus:    "",
	}

	return resp, nil
}
