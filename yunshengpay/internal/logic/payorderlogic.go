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
	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

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
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	orderTime := time.Now().Format("2006-01-02 15:04:05")
	randomID := utils.GetRandomString(10, utils.ALL, utils.MIX)

	// 組請求參數
	dataInit := struct {
		MerchId      string `json:"merchantId"`
		UserId       string `json:"userId"`
		OrderId      string `json:"orderId"`
		Time         string `json:"orderTime"`
		TerminalType string `json:"terminalType"`
		Amount       string `json:"amount"`
		Payer        string `json:"payer"`
		NotifyUrl    string `json:"notifyUrl"`
		PayType      string `json:"payWith"`
		Nonce        string `json:"nonce"`
		Timestamp    string `json:"timestamp"`
		UseCounter   bool   `json:"useCounter"`
	}{
		MerchId:      channel.MerId,
		UserId:       channel.MerId,
		OrderId:      req.OrderNo,
		Time:         orderTime,
		TerminalType: "PC",
		Amount:       req.TransactionAmount, //两位小数
		Payer:        req.UserId,
		NotifyUrl:    notifyUrl,
		PayType:      req.ChannelPayType,
		Nonce:        randomID,
		Timestamp:    timestamp,
		UseCounter:   true,
	}
	dataBytes, err := json.Marshal(dataInit)

	encryptContent := utils.EnPwdCode(string(dataBytes), channel.MerKey)

	// 組請求參數 FOR JSON
	reqObj := struct {
		Id   string `json:"id"`
		Data string `json:"data"`
	}{
		Id:   channel.MerId,
		Data: encryptContent,
	}
	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", reqObj)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}


	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%#v", channel.PayUrl, reqObj)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(10).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(10).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(reqObj)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	response := utils.DePwdCode(string(res.Body()), channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    int     `json:"code"`
		Msg     string  `json:"msg, optional"`
		Success bool    `json:"success, optional"`
		TransId string  `json:"transId, optional"` //：支付公司单号
		OrderId string  `json:"orderId, optional"`
		Amount  float64 `json:"amount, optional"`  //：实际金额
		Url     string  `json:"url, optional"`     //：收银台地址
		Account string  `json:"account, optional"` //：银行卡号
		Bank    string  `json:"bank, optional"`    //：银行编码
		Branch  string  `json:"branch, optional"`
		Holder  string  `json:"holder, optional"` //：持卡人姓名
	}{}

	// 返回body 轉 struct
	//if err = res.DecodeJSON(&channelResp); err != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	//}

	if err = json.Unmarshal([]byte(response), &channelResp); err != nil {
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
	if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
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
		PayPageInfo:    channelResp.Url,
		ChannelOrderNo: "",
	}

	return
}
