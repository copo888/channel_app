package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/bitpayz/internal/payutils"
	"github.com/copo888/channel_app/bitpayz/internal/service"
	"github.com/copo888/channel_app/bitpayz/internal/svc"
	"github.com/copo888/channel_app/bitpayz/internal/types"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"github.com/skip2/go-qrcode"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"
)

type PayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s,orderNo: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	if len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	} else if len(req.BankAccount) == 0 {
		logx.WithContext(l.ctx).Errorf("bankAccount不可为空 userId:%s", req.BankAccount)
		return nil, errorx.New(responsex.BANK_ACCOUNT_EMPTY)
	}

	channelBankMap, err2 := model.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.BankCode)
	if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
		logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode+"，渠道Map名称: "+channelBankMap.MapCode)
	} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
		logx.WithContext(l.ctx).Errorf("银行代码: %s, 渠道银行代码: %s", req.BankCode, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode+"，渠道Map名称: "+channelBankMap.MapCode)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	timestamp := time.Now().UnixMilli()
	amountF, _ := strconv.ParseFloat(req.TransactionAmount, 64)

	// 組請求參數 FOR JSON
	data := struct {
		ClientId          string  `json:"clientId"`
		MerchantId        string  `json:"merchantId"`
		TransactionId     string  `json:"transactionId"`
		BankAccountNumber string  `json:"bankAccountNumber"`
		BankName          string  `json:"bankName"`
		Name              string  `json:"name"`
		Amount            float64 `json:"amount"`
		CallbackUrl       string  `json:"callbackUrl"`
		Type              string  `json:"type"`
		Timeout           int     `json:"timeout"`
		Signature         string  `json:"signature"`
		Timestamp         int64   `json:"timestamp"`
	}{
		ClientId:          "nHUxQbHgEu",
		MerchantId:        channel.MerId,
		TransactionId:     req.OrderNo,
		BankAccountNumber: req.BankAccount,
		BankName:          channelBankMap.MapCode,
		Name:              req.UserId,
		Amount:            amountF,
		CallbackUrl:       notifyUrl,
		Type:              "QR",
		Timeout:           30, //Minutes
		Timestamp:         timestamp,
	}
	// 加簽
	sign, err := payutils.GetSign_HMAC_SHA256(channel.MerId, "nHUxQbHgEu", channel.MerKey, timestamp)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("签名错误: %s", err.Error())
	}
	data.Signature = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         data,
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Header("x-api-key", "825c850d-cc4b-410e-b3a4-b1fc3d898d79").Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		service.DoCallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          string(res.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(res.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Status  string `json:"status"`
		code    int    `json:"code, optional"`
		Message string `json:"message"`
		Data    struct {
			ClientId          string  `json:"clientId, optional"`
			MerchantId        string  `json:"merchantId, optional"`
			ReferenceId       string  `json:"referenceId, optional"`
			TransactionId     string  `json:"transactionId, optional"`
			Status            string  `json:"status, optional"`
			Amount            float64 `json:"amount, optional"`
			DepositAmount     float64 `json:"depositAmount, optional"`
			Qrcode            string  `json:"qrcode, optional"`
			BankAccountNumber string  `json:"bankAccountNumber, optional"`
			BankAccountName   string  `json:"bankAccountName, optional"`
			BankName          string  `json:"bankName, optional"`
			BankCode          string  `json:"bankCode, optional"`
			PromptpayNumber   string  `json:"promptpayNumber, optional"`
			ExpireDate        string  `json:"expireDate, optional"`
			CustomerData      struct {
				BankAccountNumber string `json:"bankAccountNumber, optional"`
				BankName          string `json:"bankName, optional"`
				Name              string `json:"name, optional"`
			} `json:"customerData, optional"`
		} `json:"data, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Status != "success" {
		// 寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          fmt.Sprintf("%+v", channelResp),
			TraceId:          l.traceID,
			ChannelErrorCode: fmt.Sprintf("%d", channelResp.code),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		ChannelCode:     channel.Code,
		MerchantOrderNo: req.MerchantOrderNo,
		OrderNo:         req.OrderNo,
		LogType:         constants.RESPONSE_FROM_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         fmt.Sprintf("%+v", channelResp),
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if QrCodeBase64, err := generateQRCodeBase64(channelResp.Data.Qrcode); err != nil {
		logx.WithContext(l.ctx).Errorf("生成二维码错误: %s", err.Error())
	} else {
		channelResp.Data.Qrcode = QrCodeBase64
	}

	// 若需回傳JSON 請自行更改
	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
		CardName:   channelResp.Data.BankAccountName,
		CardNumber: channelResp.Data.BankAccountNumber,
		BankName:   channelResp.Data.BankName,
		BankBranch: "",
		Amount:     channelResp.Data.DepositAmount,
		Link:       "",
		QrCode:     channelResp.Data.Qrcode,
		Currency:   req.Currency,
		Remark:     "",
	})

	if err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	return &types.PayOrderResponse{
		PayPageType:    "json",
		PayPageInfo:    string(receiverInfoJson),
		ChannelOrderNo: "",
		IsCheckOutMer:  true, // 自組收銀台回傳 true
	}, nil

	return
}

func generateQRCodeBase64(content string) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	imgBase64 := base64.StdEncoding.EncodeToString(png)
	return "data:image/png;base64," + imgBase64, nil
}
