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
	"github.com/copo888/channel_app/indiapay/internal/payutils"
	"github.com/copo888/channel_app/indiapay/internal/service"
	"github.com/copo888/channel_app/indiapay/internal/svc"
	"github.com/copo888/channel_app/indiapay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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

	if req.SourceIp == "" {
		logx.WithContext(l.ctx).Errorf("SourceIp不可为空 SourceIp:%s", req.SourceIp)
		return nil, errorx.New(responsex.INVALID_PARAMETER, "UserIp is required")
	}

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	floatNum, err := strconv.ParseFloat(req.TransactionAmount, 64)
	if err != nil {
		fmt.Println("Error converting string to float:", err)
		return
	}

	// 組請求參數
	data := url.Values{}
	data.Set("mer_id", channel.MerId)
	data.Set("order_id", req.OrderNo)
	data.Set("gateway", req.ChannelPayType)
	data.Set("amount", fmt.Sprintf("%.0f", floatNum)) //整數，不要小數
	data.Set("callback", req.PageUrl)
	data.Set("notify", notifyUrl)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("player_ip", req.SourceIp)
	data.Set("sign", sign)
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey,l.ctx)
	//data.sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:  req.MerchantId,
		ChannelCode: channel.Code,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data,
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{222
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
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
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
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
		Code      int    `json:"code"`
		Data      string `json:"data"`
		Message   string `json:"message"`
		Upi       string `json:"upi"`
		Amount    string `json:"amount"`
		Token     string `json:"token"`
		OrderCode string `json:"order_code"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Code != 0 {
		// 寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
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
		MerchantNo:  req.MerchantId,
		ChannelCode: channel.Code,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 若需回傳JSON 請自行更改22
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
		PayPageInfo:    channelResp.Data,
		ChannelOrderNo: "",
	}

	return
}
