package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/mashangpay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"

	"github.com/copo888/channel_app/mashangpay/internal/svc"
	"github.com/copo888/channel_app/mashangpay/internal/types"

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

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (resp *types.ProxyPayOrderResponse, err error) {

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
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//notifyUrl := "http://5d66-211-75-36-190.ngrok.io/api/proxy-pay-call-back"

	data := struct {
		MerchId           string `json:"merchantCode"`
		OrderId           string `json:"merchantOrderId"`
		BankCode          string `json:"bankCode"`
		BankAccountName   string `json:"bankAccountName"`
		BankAccountNumber string `json:"bankAccountNumber"`
		Branch            string `json:"branch"` //无资料时,可填入银行名称
		Province          string `json:"province"`
		City              string `json:"city"`
		Amount            string `json:"amount"` //单位;元,精确到元,
		NotifyUrl         string `json:"successUrl"`
		Sign              string `json:"sign"`
	}{
		MerchId:           channel.MerId,
		OrderId:           req.OrderNo,
		BankCode:          channelBankMap.MapCode,
		BankAccountName:   req.ReceiptAccountName,
		BankAccountNumber: req.ReceiptAccountNumber,
		Branch:            req.ReceiptCardBankName,
		Province:          req.ReceiptCardProvince,
		City:              req.ReceiptCardCity,
		Amount:            transactionAmount,
		NotifyUrl:         notifyUrl,
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

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
	Res, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if Res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", Res.Status(), string(Res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", Res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", Res.Status(), string(Res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Result   bool `json:"result"`
		ErrorMsg struct {
			Code     int    `json:"code"`
			ErrorMsg string `json:"errorMsg"`
			Descript string `json:"descript"`
		} `json:"errorMsg,optional"`
		Data struct {
			GamerOrderId string `json:"gamerOrderId"`
			HttpUrl      string `json:"httpUrl"`
			HttpsUrl     string `json:"httpsUrl"`
		} `json:"data,optional"`
	}{}

	if err = Res.DecodeJSON(&channelResp); err != nil {
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

	if strings.Index(channelResp.ErrorMsg.Descript, "代付钱包金额不足") > -1 {
		logx.WithContext(l.ctx).Errorf("代付渠提单道返回错误: %t: %s", channelResp.Result, channelResp.ErrorMsg.Descript)
		return nil, errorx.New(responsex.INSUFFICIENT_IN_AMOUNT, channelResp.ErrorMsg.Descript)
	} else if channelResp.Result != true {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.Result, channelResp.ErrorMsg.Descript)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg.Descript)
	}

	//組返回給backOffice 的代付返回物件

	return &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.GamerOrderId,
		OrderStatus:    "",
	}, nil
}
