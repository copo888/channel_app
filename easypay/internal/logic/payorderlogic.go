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
	"github.com/copo888/channel_app/easypay/internal/payutils"
	"github.com/copo888/channel_app/easypay/internal/service"
	"github.com/copo888/channel_app/easypay/internal/svc"
	"github.com/copo888/channel_app/easypay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
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
	appSecret := "17c44b5028794d8c9fc3e79c13b985ee"
	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	var channelBankMap *typesX.BankCodeMapX
	if req.BankCode != "" {
		channelBankMap, err = model.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.BankCode)
		if err != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
			logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err.Error())
			return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "银行名称: "+channelBankMap.BankName, "渠道Map名称: "+channelBankMap.MapCode)
		} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
			logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.BankCode, channelBankMap.BankName, channelBankMap.MapCode)
			return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "银行名称: "+channelBankMap.BankName, "渠道Map名称: "+channelBankMap.MapCode)
		}
	}

	/** UserId 必填時使用 **/
	if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://e46c-211-75-36-190.jp.ngrok.io/api/pay-call-back"
	dateTime := time.Now().Format("2006-01-02 15:04:05")
	timestamp := time.Now().Unix()
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	channelNo, errI := strconv.ParseInt(req.ChannelPayType, 10, 64)
	if errI != nil {
		return nil, errI
	}

	// 組請求參數
	//data := url.Values{}
	//data.Set("merchantNo", channel.MerId)
	//data.Set("orderNo", req.OrderNo)
	//data.Set("amount", req.TransactionAmount)
	//data.Set("datetime", dateTime)
	//data.Set("time", string(timestamp))
	//data.Set("notifyUrl", notifyUrl)
	//data.Set("UserNo", "")
	//data.Set("Discount", "")
	//data.Set("Extra","")

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo string  `json:"merchantNo"`
		Amount     float64 `json:"amount"`
		OrderNo    string  `json:"orderNo"`
		UserNo     string  `json:"userNo"`
		Discount   float64 `json:"discount"`
		Datetime   string  `json:"datetime"`
		Time       int64   `json:"time"`
		NotifyUrl  string  `json:"notifyUrl"`
		AppSecret  string  `json:"appSecret"`
		PayeeName  string  `json:"payeeName"`
		ChannelNo  int64   `json:"channelNo"`
		BankName   string  `json:"bankName"`
		Extra      string  `json:"extra"`
		Sign       string  `json:"sign"`
	}{
		MerchantNo: channel.MerId,
		Amount:     amountFloat,
		OrderNo:    req.OrderNo,
		Time:       timestamp,
		NotifyUrl:  notifyUrl,
		Datetime:   dateTime,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)
	//data.Set("appSecret", appSecret)
	//data.Set("payeeName ", randomID)
	//data.Set("channelNo ", req.ChannelPayType)
	//if req.ChannelPayType != "4" {
	//	data.Set("bankName ", randomID)
	//}
	if req.BankCode != "" {
		data.BankName = channelBankMap.MapCode
	}
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign
	data.AppSecret = appSecret
	data.PayeeName = req.UserId
	data.ChannelNo = channelNo

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

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

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
		Code           int     `json:"code"`
		Text           string  `json:"text"`
		TradeNo        string  `json:"tradeNo"`
		OrderNo        string  `json:"orderNo"`
		UserNo         int32   `json:"userNo"`
		UserName       string  `json:"userName"`
		ChannelNo      int     `json:"channelNo"`
		Amount         float64 `json:"amount"`
		Discount       float64 `json:"discount"`
		Lucky          float64 `json:"lucky"`
		Payable        float64 `json:"payable"`
		TargetUrl      string  `json:"targetUrl"`
		Qrcode         string  `json:"qrcode"`
		PayeeAccNumber string  `json:"payeeAccNumber"`
		KfNo           string  `json:"kfNo"`
		Extra          string  `json:"extra"`
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
	if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Text)
	}

	// 若需回傳JSON 請自行更改
	//if strings.EqualFold(req.JumpType, "json") {
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
	//		IsCheckOutMer:  false, // 自組收銀台回傳 true
	//	}, nil
	//}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.TargetUrl,
		ChannelOrderNo: "",
	}

	return
}
