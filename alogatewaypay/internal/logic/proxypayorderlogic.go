package logic

import (
	"context"
	b64 "encoding/base64"
	"encoding/xml"
	"fmt"
	"github.com/copo888/channel_app/alogatewaypay/internal/payutils"
	"github.com/copo888/channel_app/alogatewaypay/internal/service"
	"github.com/copo888/channel_app/alogatewaypay/internal/svc"
	"github.com/copo888/channel_app/alogatewaypay/internal/types"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strings"
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
	merchantAccount := "901721"
	partnerControl := "051cd2494a6b8d2e69e05607deddb5ad"
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
	amounFloat := utils.FloatMul(req.TransactionAmount, "100")
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//encName := b64.URLEncoding.EncodeToString([]byte(req.ReceiptAccountName))

	data := url.Values{}
	data.Set("merchantaccount", merchantAccount)
	data.Set("merchantorder", req.OrderNo)
	data.Set("amount", fmt.Sprintf("%.f", amounFloat))
	data.Set("currency", "INR") //印度盧比
	data.Set("customername", b64.URLEncoding.EncodeToString([]byte(req.ReceiptAccountName)))
	data.Set("bankcode", channelBankMap.MapCode)
	data.Set("bankaccountnumber", req.ReceiptAccountNumber)

	//data.Set("subsidiaryBank", req.ReceiptCardBankName)
	//data.Set("subbranch", req.ReceiptCardBranch)
	//data.Set("province", req.ReceiptCardProvince)
	//data.Set("city", req.ReceiptCardCity)

	keys := []string{"merchantaccount", "merchantorder", "amount", "currency", "customername", "bankcode", "bankaccountnumber"}
	// 加簽
	sign := payutils.SortAndSignFromUrlValues_2(l.ctx, data, keys, partnerControl)
	data.Set("control", sign)
	data.Set("version", "11")
	data.Set("serverreturnurl", notifyUrl)
	data.Set("bankbranchaddress", "") //请商户传
	data.Set("customername", req.ReceiptAccountName)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: channel.MerId,
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
	res, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("代付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.DoCallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %s  Body: %s", res.Status, res.Body)
		msg := fmt.Sprintf("代付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
		service.DoCallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	Credit := struct {
		XMLName           xml.Name `xml:"credit"`
		Text              string   `xml:",chardata"`
		Transactionid     string   `xml:"transactionid"`
		Merchantaccount   string   `xml:"merchantaccount"`
		MerchantOrder     string   `xml:"merchant_order"`
		Amount            string   `xml:"amount"`
		Currency          string   `xml:"currency"`
		Customername      string   `xml:"customername"`
		Bankcode          string   `xml:"bankcode"`
		Bankaccountnumber string   `xml:"bankaccountnumber"`
		Status            string   `xml:"status"`
		Message           string   `xml:"message"`
		Control           string   `xml:"control"`
	}{}
	//defer res.Body.Close()
	//bodyBytes, _ := ioutil.ReadAll(res.Body())
	//stringBody := string(bodyBytes)
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status, string(res.Body()))

	if err := xml.Unmarshal(res.Body(), &Credit); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: merchantAccount,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", Credit)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if !strings.EqualFold(Credit.Status, "A0") {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %+v", Credit.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, Credit.Message)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: Credit.Transactionid,
		OrderStatus:    "",
	}

	return resp, nil
}
