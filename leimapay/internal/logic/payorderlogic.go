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
	"github.com/copo888/channel_app/leimapay/internal/payutils"
	"github.com/copo888/channel_app/leimapay/internal/service"
	"github.com/copo888/channel_app/leimapay/internal/svc"
	"github.com/copo888/channel_app/leimapay/internal/types"
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
	if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}
	//if len(req.JumpType) == 0 {
	//	logx.WithContext(l.ctx).Errorf("JumpType不可为空 JumpType不可为空:%s", req.JumpType)
	//	return nil, errorx.New(responsex.INVALID_PARAMETER)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://f632-211-75-36-190.ngrok-free.app/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	amount, _ := strconv.ParseFloat(req.TransactionAmount, 64)

	// 組請求參數
	//data := url.Values{}
	//data.Set("merchId", channel.MerId)
	//data.Set("money", req.TransactionAmount)
	//data.Set("userId", randomID)
	//data.Set("orderId", req.OrderNo)
	//data.Set("time", timestamp)
	//data.Set("notifyUrl", notifyUrl)
	//data.Set("payType", req.ChannelPayType)
	//data.Set("reType", "LINK")
	//data.Set("signType", "MD5")

	// 組請求參數 FOR JSON
	data := struct {
		MerchantCode     string  `json:"merchantCode"`
		MerchantOrderId  string  `json:"merchantOrderId"`
		CurrencyCode     string  `json:"currencyCode"`
		PaymentTypeCode  string  `json:"paymentTypeCode"`
		Amount           float64 `json:"amount"`
		SuccessUrl       string  `json:"successUrl"`
		Mp               string  `json:"mp"`
		MerchantMemberId string  `json:"merchantMemberId"`
		MerchantMemberIp string  `json:"merchantMemberIp"`
		PayerName        string  `json:"payerName"`
		Sign             string  `json:"sign"`
	}{
		MerchantCode:     channel.MerId,
		MerchantOrderId:  req.OrderNo,
		CurrencyCode:     req.Currency,
		PaymentTypeCode:  req.ChannelPayType,
		Amount:           amount,
		SuccessUrl:       notifyUrl,
		Mp:               "deposit",
		MerchantMemberId: randomID,
		MerchantMemberIp: ip,
		PayerName:        req.UserId,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
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
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Result   bool `json:"result"`
		ErrorMsg struct {
			Code     int    `json:"code"`
			ErrorMsg string `json:"errorMsg"`
			Descript string `json:"descript"`
		} `json:"errorMsg, optional"`
		Data struct {
			GamerOrderId      string `json:"gamerOrderId"`
			Amount            string `json:"amount"`
			BankName          string `json:"bankName"`
			BankAccountNumber string `json:"bankAccountNumber"`
			BankAccountName   string `json:"bankAccountName"`
			Sign              string `json:"sign"`
		} `json:"data, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Result != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg.ErrorMsg+":"+channelResp.ErrorMsg.Descript)
	}

	isCheckOutMer := true
	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		isCheckOutMer = false
	}
	//isCheckOutMer := true // 自組收銀台回傳 true
	//if req.MerchantId == "ME00015" {
	//	isCheckOutMer = true
	//}
	amount, err2 := strconv.ParseFloat(channelResp.Data.Amount, 64)
	if err2 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
	}
	// 返回json
	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
		CardName:   channelResp.Data.BankAccountName,
		CardNumber: channelResp.Data.BankAccountNumber,
		BankName:   channelResp.Data.BankName,
		BankBranch: "",
		Amount:     amount,
		Link:       "",
		Remark:     "",
	})
	if err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	return &types.PayOrderResponse{
		PayPageType:    "json",
		PayPageInfo:    string(receiverInfoJson),
		ChannelOrderNo: "",
		IsCheckOutMer:  isCheckOutMer,
	}, nil

	//resp = &types.PayOrderResponse{
	//	PayPageType:    "url",
	//	PayPageInfo:    channelResp.PayUrl,
	//	ChannelOrderNo: "",
	//}

	return
}
