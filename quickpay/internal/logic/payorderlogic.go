package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/constants/redisKey"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/quickpay/internal/payutils"
	"github.com/copo888/channel_app/quickpay/internal/svc"
	"github.com/copo888/channel_app/quickpay/internal/types"
	"strconv"
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
	channelBankMap, err2 := model.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.BankCode)
	if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
		logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "渠道Map名称: "+channelBankMap.MapCode)
	} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
		logx.WithContext(l.ctx).Errorf("银行代码: %s,渠道银行代码: %s", req.BankCode, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "渠道Map名称: "+channelBankMap.MapCode)
	}

	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://effa-211-75-36-190.jp.ngrok.io/api/pay-call-back"

	timestamp := time.Now()
	tf := timestamp.Format("2006-01-02 03:04:05PM")
	tfs := timestamp.Format("20060102150405")
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)

	//ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	//data := url.Values{}
	//data.Set("Merchant", channel.MerId)
	//data.Set("MerchantTrxRef", req.OrderNo)
	//data.Set("Currency", req.Currency)
	//data.Set("Customer", randomID)
	//data.Set("Amount", transactionAmount)
	//data.Set("DateTime",tf)
	//data.Set("SuccessURI", req.PageUrl)
	//data.Set("FailedURI","http://dev.copo.pro/#/home")
	//data.Set("BackURI", notifyUrl)
	//data.Set("Bank", channelBankMap.MapCode)

	// 組請求參數 FOR JSON
	data := struct {
		RequestUrl     string `json:"requestUrl"`
		Merchant       string `json:"Merchant"`
		MerchantTrxRef string `json:"MerchantTrxRef"`
		Currency       string `json:"Currency"`
		Customer       string `json:"Customer"`
		Amount         string `json:"Amount"`
		DateTime       string `json:"DateTime"`
		SuccessURI     string `json:"SuccessURI"`
		FailedURI      string `json:"FailedURI"`
		BackURI        string `json:"BackURI"`
		Bank           string `json:"Bank"`
		Key            string `json:"Key"`
	}{
		RequestUrl:     channel.PayUrl,
		Merchant:       channel.MerId,
		Amount:         req.TransactionAmount,
		MerchantTrxRef: req.OrderNo,
		Currency:       req.Currency,
		Customer:       randomID,
		DateTime:       tf,
		SuccessURI:     req.PageUrl,
		FailedURI:      req.PageFailedUrl,
		BackURI:        notifyUrl,
		Bank:           channelBankMap.MapCode,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	source := channel.MerId + randomID + transactionAmount + req.Currency + tfs + channel.MerKey
	sign := payutils.GetSign(source)
	data.Key = sign
	logx.WithContext(l.ctx).Info("加签参数: ", source)
	logx.WithContext(l.ctx).Info("签名字串: ", sign)
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
	//span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)
	//
	//if ChnErr != nil {
	//	logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
	//	msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
	//	service.CallLineSendURL(l.ctx, l.svcCtx, msg)
	//	return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	//}else if res.Status() != 200 {
	//	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), res.Body)
	//	msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
	//	service.CallLineSendURL(l.ctx, l.svcCtx, msg)
	//	return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	//}

	//// 渠道回覆處理 [請依照渠道返回格式 自定義]
	//channelResp := struct {
	//	Code    string `json:"code"`
	//	Msg     string `json:"msg, optional"`
	//	Sign    string `json:"sign"`
	//	Money   string `json:"money"`
	//	OrderId string `json:"orderId"`
	//	PayUrl  string `json:"payUrl"`
	//	PayInfo struct {
	//		Name       string `json:"name"`
	//		Card       string `json:"card"`
	//		Bank       string `json:"bank"`
	//		Subbranch  string `json:"subbranch"`
	//		ExpiringAt string `json:"expiring_at"`
	//	} `json:"payInfo"`
	//}{}

	// 返回body 轉 struct
	//if err = res.DecodeJSON(&channelResp); err != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	//}

	//寫入交易日志
	//if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
	//	MerchantNo: req.MerchantId,
	//	//MerchantOrderNo: req.OrderNo,
	//	OrderNo:   req.OrderNo,
	//	LogType:   constants.RESPONSE_FROM_CHANNEL,
	//	LogSource: constants.API_ZF,
	//	Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
	//	logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	//}

	// 渠道狀態碼判斷
	//if channelResp.Code != "0000" {
	//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	//}

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

	// 资料转JSON
	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, errorx.New(responsex.DECODE_JSON_ERROR)
	}
	// 存 Redis
	redisKey := redisKey.CACHE_PAY_ORDER_CHANNEL_REDIRECT + req.OrderNo
	logx.WithContext(l.ctx).Infof("redisKey : ", redisKey)
	if err = l.svcCtx.RedisClient.Set(l.ctx, redisKey, dataJson, 15*time.Minute).Err(); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION)
	}

	url := fmt.Sprintf("%s/#/redirectPage?orderNo=%s", l.svcCtx.Config.FrontEndDomain, req.OrderNo)

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    url,
		ChannelOrderNo: "",
	}

	return
}
