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
	"github.com/copo888/channel_app/sinhongpay/internal/payutils"
	"github.com/copo888/channel_app/sinhongpay/internal/service"
	"github.com/copo888/channel_app/sinhongpay/internal/svc"
	"github.com/copo888/channel_app/sinhongpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

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

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://f599-211-75-36-190.jp.ngrok.io/api/pay-call-back"
	timestamp := time.Now().Format("20060102150405")
	ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	attach := struct {
		RealName string `json:"real_name"`
	}{
		RealName: req.UserId,
	}

	infoJson, jsonErr := json.Marshal(attach)

	if jsonErr != nil {
		return nil, errorx.New(responsex.DECODE_JSON_ERROR, jsonErr.Error())
	}

	// 組請求參數
	headMap := make(map[string]string)
	headMap["sid"] = channel.MerId
	headMap["timestamp"] = timestamp
	headMap["nonce"] = randomID
	headMap["url"] = "/pay/qrorder"

	data := url.Values{}
	data.Set("amount", transactionAmount)
	data.Set("out_trade_no", req.OrderNo)
	data.Set("channel", req.ChannelPayType)
	data.Set("notify_url", notifyUrl)
	data.Set("currency", "CNY")
	data.Set("send_ip", ip)
	data.Set("attach", string(infoJson))
	data.Set("return_url", "")

	// 組請求參數 FOR JSON
	//data := struct {
	//	MerchId   string `json:"merchId"`
	//	Money     string `json:"money"`
	//	OrderId   string `json:"orderId"`
	//	Time      string `json:"time"`
	//	NotifyUrl string `json:"notifyUrl"`
	//	PayType   string `json:"payType"`
	//	sign      string `json:"sign"`
	//}{
	//	MerchId:   channel.MerId,
	//	Money:     req.TransactionAmount,
	//	OrderId:   req.OrderNo,
	//	Time:      timestamp,
	//	NotifyUrl: notifyUrl,
	//	PayType:   req.ChannelPayType,
	//}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	headSource := payutils.JoinStringsInASCII(headMap, "", false, false, "")
	bodyDataMap := payutils.CovertUrlValuesToMap(data)
	bodySource := payutils.JoinStringsInASCII(bodyDataMap, "", false, true, "")
	newSource := headSource + bodySource + channel.MerKey
	newSign := payutils.GetSign(newSource)
	logx.WithContext(l.ctx).Info("加签参数: ", newSource)
	logx.WithContext(l.ctx).Info("签名字串: ", newSign)
	headMap["sign"] = newSign
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data}); err != nil {
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
	res, ChnErr := gozzle.Post(channel.PayUrl).Headers(headMap).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code       string `json:"code"`
		Msg        string `json:"msg, optional"`
		OutTradeNo string `json:"out_trade_no, optional"`
		Sign       string `json:"sign,optional"`
		TradeNo    string `json:"trade_no,optional"`
		PayUrl     string `json:"pay_url, optional"`
	}{}

	PayUrl := struct {
		PayUrl  string `json:"pay_url,optional"`
		PayJson struct {
			QrcodeUrl   string `json:"qrcode_url,optional"`
			BankName    string `json:"bank_name,optional"`
			AccountName string `json:"account_name,optional"`
			CardNumber  string `json:"card_number,optional"`
			SubBank     string `json:"sub_bank,optional"`
			PayAmount   string `json:"pay_amount,optional"`
		} `json:"pay_json,optional"`
	}{}

	//PayJson := struct {
	//	PayJson struct{
	//		QrcodeUrl string `json:"qrcode_url,optional"`
	//		BankName string `json:"bank_name,optional"`
	//		AccountName string `json:"account_name,optional"`
	//		CardNumber string `json:"card_number,optional"`
	//		SubBank string `json:"sub_bank,optional"`
	//		PayAmount string `json:"pay_amount,optional"`
	//	} `json:"pay_json,optional"`
	//}{}

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
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Code != "1000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	err2 := json.Unmarshal([]byte(channelResp.PayUrl), &PayUrl)

	if err2 != nil {
		return nil, errorx.New(responsex.DECODE_JSON_ERROR, jsonErr.Error())
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		amount, err2 := strconv.ParseFloat(PayUrl.PayJson.PayAmount, 64)
		if err2 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
		}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   PayUrl.PayJson.AccountName,
			CardNumber: PayUrl.PayJson.CardNumber,
			BankName:   PayUrl.PayJson.BankName,
			BankBranch: PayUrl.PayJson.SubBank,
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
			IsCheckOutMer:  false, // 自組收銀台回傳 true
		}, nil
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    PayUrl.PayUrl,
		ChannelOrderNo: "",
	}

	return
}
