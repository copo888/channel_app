package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/direpay/internal/payutils"
	"github.com/copo888/channel_app/direpay/internal/service"
	"github.com/copo888/channel_app/direpay/internal/svc"
	"github.com/copo888/channel_app/direpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
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
	if len(req.PlayerId) == 0 {
		logx.WithContext(l.ctx).Errorf("PlayerId不可为空 PlayerId:%s", req.PlayerId)
		return nil, errorx.New(responsex.INVALID_PLAYER_ID)
	}

	if len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("UserId不可为空 UserId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	if len(req.BankCode) == 0 {
		logx.WithContext(l.ctx).Errorf("BankCode不可为空 BankCode:%s", req.BankCode)
		return nil, errorx.New(responsex.BANK_CODE_EMPTY)
	}

	if len(req.BankAccount) == 0 {
		logx.WithContext(l.ctx).Errorf("BankAccount不可为空 BankAccount:%s", req.BankAccount)
		return nil, errorx.New(responsex.BANK_ACCOUNT_EMPTY)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	//transaction :=
	//	struct {
	//		Currency    string `json:"currency"`
	//		Amount      string `json:"amount"`
	//		Token       string `json:"token"`
	//		CallbackUrl string `json:"callback_url"`
	//		CustomerUid string `json:"customer_uid"`
	//	}{
	//		Currency:    "CNY",                 //占时固定CNY
	//		Amount:      req.TransactionAmount, //依法币数额作依据，不是token(BTC)
	//		Token:       "BTC",
	//		CallbackUrl: notifyUrl,
	//		CustomerUid:  req.UserId, //请填 客户独特编号
	//	}
	//byteTrans, err := json.Marshal(transaction)
	//if err != nil {
	//	logx.WithContext(l.ctx).Errorf("序列化错误:+%v", transaction)
	//	return nil, errorx.New("序列化错误", err.Error())
	//}
	//encryptedData, AesErr := payutils.AesEcrypt(byteTrans, []byte(channel.MerKey))
	//if AesErr != nil {
	//	logx.WithContext(l.ctx).Errorf("加密错误:%s", AesErr.Error())
	//	return nil, errorx.New("加密错误", AesErr.Error())
	//}

	//data := url.Values{}
	//data.Set("data", string(encryptedData))
	//data.Set("platform", channel.MerId)
	//data.Set("lang", "en")

	data := struct {
		Command       string `json:"command"`
		HashCode      string `json:"hashCode"`
		CallbackUrl   string `json:"callback_url"`
		RedirectUrl   string `json:"redirect_url"`
		Currency      string `json:"currency"`
		Method        string `json:"method"`
		CustomerUid   string `json:"customer_uid"` //客户独特编号
		DepositorName string `json:"depositor_name"`
		Amount        string `json:"amount"`
		TxId          string `json:"txid, optional"`
		BankCode      string `json:"bank_code, optional"`
		BankAccount   string `json:"bank_account, optional"` //(当currency是THB, method是card时需填入)
	}{
		Command:       "fiat_payment",
		HashCode:      payutils.GetSign("fiat_payment" + channel.MerKey),
		CallbackUrl:   notifyUrl,
		RedirectUrl:   req.PageUrl,
		Currency:      "THB",
		Method:        req.ChannelPayType, //qr
		CustomerUid:   req.PlayerId,
		DepositorName: req.UserId,
		TxId:          req.OrderNo,
		Amount:        req.TransactionAmount,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		ChannelCode:     channel.Code,
		MerchantOrderNo: req.MerchantOrderNo,
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

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).
		Header("Authorization", "Bearer "+l.svcCtx.Config.AccessToken).
		Header("Content-type", "application/json").
		JSON(data)
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

		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if (res.Status() < 400 && res.Status() >= 300) || res.Status() < 200 {
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
		Txid     string `json:"txid, optional"`
		Currency string `json:"currency, optional"`
		Amount   string `json:"amount, optional"`
		PayUrl   string `json:"pay_url, optional"`
		Method   string `json:"method, optional"`
		Status   string `json:"status, optional"`
		Message  string `json:"message, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if strings.Index(channelResp.Status, "success") <= -1 {
		// 寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          channelResp.Message,
			TraceId:          l.traceID,
			ChannelErrorCode: channelResp.Status,
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	} else {
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

	}
	// 若需回傳JSON 請自行更改
	//if strings.EqualFold(req.JumpType, "json") {
	//	isCheckOutMer := false // 自組收銀台回傳 true
	//	if req.MerchantId == "ME00015"{
	//		isCheckOutMer = true
	//	}
	//}

	//invoiceAmount, err := channelResp.InvoiceAmount.Float64()
	//
	//// 返回json
	//receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoBTCVO{
	//	OrderNo:         req.OrderNo,
	//	CustomerUid:     channelResp.CustomerUid,
	//	InvoiceAmount:   invoiceAmount,
	//	InvoiceCurrency: channelResp.InvoiceCurrency, //法币
	//	PaymentAmount:   channelResp.PaymentAmount,   //加密货币
	//	PaymentToken:    channelResp.PaymentToken,
	//	Rates:           channelResp.Rates,
	//	PaymentAddress:  channelResp.PaymentAddress,
	//})

	//if err3 != nil {
	//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	//}

	return &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.PayUrl,
		ChannelOrderNo: channelResp.Txid,
	}, nil

	return
}
