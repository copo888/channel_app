package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/my888pay/internal/payutils"
	"github.com/copo888/channel_app/my888pay/internal/service"
	"github.com/copo888/channel_app/my888pay/internal/svc"
	"github.com/copo888/channel_app/my888pay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
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

	channelBankMap, err2 := model.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.BankCode)
	if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
		logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "银行名称: "+"渠道Map名称: "+channelBankMap.MapCode)
	} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
		logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.BankCode, channelBankMap.BankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "银行名称: "+channelBankMap.BankName, "渠道Map名称: "+channelBankMap.MapCode)
	}

	/** UserId 必填時使用 **/
	if len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("UserId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	if len(req.PlayerId) == 0 {
		logx.WithContext(l.ctx).Errorf("PlayerId不可为空 playerId:%s", req.PlayerId)
		return nil, errorx.New(responsex.INVALID_PLAYER_ID)
	}

	if len(req.BankAccount) == 0 {
		logx.WithContext(l.ctx).Errorf("BankAccount不可为空 BankAccount:%s", req.BankAccount)
		return nil, errorx.New(responsex.INVALID_BANK_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	amount, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	//ip := utils.GetRandomIp()

	// 組請求參數 FOR JSON
	data := struct {
		PlatformOrderNumber string  `json:"platform_order_number"`
		Account             string  `json:"account"`
		Username            string  `json:"username"`
		RemittanceBank      string  `json:"remittance_bank"`
		RemittanceAccount   string  `json:"remittance_account"`
		Amount              float64 `json:"amount"`
		ApiKey              string  `json:"api_key"`
		CallbackUrl         string  `json:"callback_url"`
		Phone               string  `json:"phone"`
	}{
		PlatformOrderNumber: req.OrderNo,
		Account:             req.PlayerId,           //会员平台的帳號
		Username:            req.UserId,             //真实姓名
		RemittanceBank:      channelBankMap.MapCode, //User Withdrawal Bank Code
		RemittanceAccount:   req.BankAccount,        //会员银行账号
		Amount:              amount,
		ApiKey:              channel.MerKey,
		CallbackUrl:         notifyUrl,
		Phone:               "01012345678", //req.Phone, 占时写死，当渠道有规定要会员手机时
	}

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
		LogSource:       constants.API_ZF,
		Content:         data,
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v,加密后资料%+v", channel.PayUrl, data, encrypedData)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).
		Header("Content-Type", "text/plain").
		Header("X-Api-Key", channel.MerKey).
		Timeout(20).Trace(span).Body([]byte(encrypedData))

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
	}
	//else if res.Status() != 200 {
	//	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	//	msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
	//	service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
	//
	//	//寫入交易日志
	//	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
	//		MerchantNo:       req.MerchantId,
	//		ChannelCode:      channel.Code,
	//		MerchantOrderNo:  req.MerchantOrderNo,
	//		OrderNo:          req.OrderNo,
	//		LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
	//		LogSource:        constants.API_ZF,
	//		Content:          string(res.Body()),
	//		TraceId:          l.traceID,
	//		ChannelErrorCode: strconv.Itoa(res.Status()),
	//	}); err != nil {
	//		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	//	}
	//
	//	return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	//}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code int `json:"code"`
		Data struct {
			Url string `json:"url, optional"`
		} `json:"data, optional"`
		ErrorText string `json:"error_text, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Code != 200 {
		// 寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          channelResp.ErrorText,
			TraceId:          l.traceID,
			ChannelErrorCode: fmt.Sprintf("%d", channelResp.Code),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorText)
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

	// 若需回傳JSON 請自行更改
	//if strings.EqualFold(req.JumpType, "json") {
	//	isCheckOutMer := false // 自組收銀台回傳 true
	//	if req.MerchantId == "ME00015"{
	//		isCheckOutMer = true
	//	}
	//	amount, err2 := strconv.ParseFloat(channelResp.Money, 64)
	//	if err2 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
	//	}
	//	// 返回json
	//	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
	//		CardName:   channelResp.PayInfo.Name,
	//		CardNumber: channelResp.PayInfo.Card,
	//		BankName:   channelResp.PayInfo.Bank,
	//		BankBranch: channelResp.PayInfo.Subbranch,
	//		Amount:     amount,
	//		Link:       "",
	//		Remark:     "",
	//	})
	//	if err3 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	//	}
	//	return &types.PayOrderResponse{
	//		PayPageType:    "json",
	//		PayPageInfo:    string(receiverInfoJson),
	//		ChannelOrderNo: "",
	//		IsCheckOutMer:  isCheckOutMer, // 自組收銀台回傳 true
	//	}, nil
	//}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Data.Url,
		ChannelOrderNo: "",
	}

	return
}
