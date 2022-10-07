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
	"github.com/copo888/channel_app/yibipay/internal/payutils"
	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
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
	var currency string
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
	//notifyUrl := "http://28fa-211-75-36-190.ngrok.io/api/pay-call-back"
	timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	aesKey := "qHp8VxRtzQ7HpBfE"
	if strings.EqualFold("USDT", req.Currency) {
		currency = "1"
	}

	// 組請求參數 FOR JSON
	dataInit := struct {
		Money       string `json:"amount"`
		NotifyUrl   string `json:"callBackUrl"`
		CoinType    string `json:"coinType"` //链类型2=ERC20 4=TRC20
		Currency    string `json:"currency"` //1 USDT  -1 CNY
		OrderId     string `json:"depositOrderId"`
		MerchId     string `json:"merchantCode"`
		PayType     string `json:"payType"`     //1 :直接跳转到付款页面，配合现金支付方式 一起使用，这个参数有值时，paymentType 必须有值 2 :直接跳转到扫码转币页面
		PaymentType string `json:"paymentType"` //1 银行卡
		Remark      string `json:"remark"`
		ReturnUrl   string `json:"returnUrl"`
		Telephone   string `json:"telephone"`
		Time        string `json:"timestamp"`
		UserCode    string `json:"userCode"`
	}{
		Money:       req.TransactionAmount,
		NotifyUrl:   notifyUrl,
		CoinType:    "4",
		Currency:    currency,
		OrderId:     req.OrderNo,
		MerchId:     channel.MerId,
		PayType:     "2",
		PaymentType: "2",
		ReturnUrl:   notifyUrl,
		Remark:      "remark",
		Telephone:   "13301110000",
		Time:        timestamp,
		UserCode:    randomID,
	}
	dataBytes, err := json.Marshal(dataInit)
	params := utils.EnPwdCode(string(dataBytes), aesKey)
	sign := payutils.SortAndSignSHA256FromObj(dataInit, channel.MerKey)
	logx.WithContext(l.ctx).Infof("加签原串:%s，Encryption: %s，Signature: %s", string(dataBytes)+channel.MerKey, params, sign)
	data := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params"`    //参数密文
		Signature    string `json:"signature"` //参数签名(params + md5key)
	}{
		MerchantCode: channel.MerId,
		Params:       params,
		Signature:    sign,
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params, optional"`
		Sign         string `json:"signature"`
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
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	paramsDecode := utils.DePwdCode(channelResp.Params, aesKey)
	logx.WithContext(l.ctx).Infof("paramsDecode: %s", paramsDecode)
	channelResp2 := struct {
		Code string `json:"code"`
		Data struct {
			PayUrl string `json:"payUrl,optional"`
		} `json:"data,optional"`
		Message string `json:"message,optional"`
	}{}

	if err = json.Unmarshal([]byte(paramsDecode), &channelResp2); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp2)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if channelResp2.Code != "200" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp2.Message)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp2.Data.PayUrl,
		ChannelOrderNo: "",
	}

	return
}
