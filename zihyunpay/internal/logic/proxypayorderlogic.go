package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/zihyunpay/internal/payutils"
	"github.com/copo888/channel_app/zihyunpay/internal/service"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/zihyunpay/internal/svc"
	"github.com/copo888/channel_app/zihyunpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
		logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
		logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName, "渠道Map名称: "+channelBankMap.MapCode)
	}
	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	timestamp := time.Now().Format("20060102150405")

	var jsnoData []struct {
		Fxddh     string `json:"fxddh"`
		Fxdate    string `json:"fxdate"`
		Fxfee     string `json:"fxfee"`
		Fxbody    string `json:"fxbody"`
		Fxname    string `json:"fxname"`
		Fxaddress string `json:"fxaddress"`
	}
	jsnoData = append(jsnoData, struct {
		Fxddh     string `json:"fxddh"`
		Fxdate    string `json:"fxdate"`
		Fxfee     string `json:"fxfee"`
		Fxbody    string `json:"fxbody"`
		Fxname    string `json:"fxname"`
		Fxaddress string `json:"fxaddress"`
	}{
		Fxddh:     req.OrderNo,
		Fxdate:    timestamp,
		Fxfee:     transactionAmount,
		Fxbody:    req.ReceiptAccountNumber,
		Fxname:    req.ReceiptAccountName,
		Fxaddress: channelBankMap.MapCode,
	})

	infoJson, jsonErr := json.Marshal(jsnoData)

	if jsonErr != nil {
		return nil, errorx.New(responsex.DECODE_JSON_ERROR, jsonErr.Error())
	}
	fxaction := "repay"
	data := url.Values{}
	data.Set("fxid", channel.MerId)
	data.Set("fxaction", fxaction)
	data.Set("fxbody", string(infoJson))
	// 加簽
	signSource := channel.MerId + fxaction + string(infoJson) + channel.MerKey
	sign := payutils.GetSign(signSource)
	logx.Info("加签参数: ", signSource)
	logx.Info("签名字串: ", sign)

	data.Set("fxnotifyurl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	data.Set("fxsign", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%+v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", ChannelResp.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		FxStatus json.Number `json:"fxstatus"`
		FxMsg    string      `json:"fxmsg"`
		FxBody   string      `json:"fxbody"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if strings.Index(channelResp.FxMsg, "余额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %s: %s", channelResp.FxStatus.String(), channelResp.FxMsg)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.FxMsg)
	} else if channelResp.FxStatus.String() != "1" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.FxStatus.String(), channelResp.FxMsg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.FxMsg)
	}

	var bodyResp []struct {
		FxStatus json.Number `json:"fxstatus"`
		FxCode   string      `json:"fxcode"`
	}

	if err := json.Unmarshal([]byte(channelResp.FxBody), &bodyResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	if bodyResp[0].FxStatus.String() != "1" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", bodyResp[0].FxStatus.String(), bodyResp[0].FxCode)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, bodyResp[0].FxCode)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: "CHN_" + req.OrderNo,
		OrderStatus:    "",
	}

	return resp, nil
}
