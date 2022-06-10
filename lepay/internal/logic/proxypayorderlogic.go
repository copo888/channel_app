package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/lepay/internal/payutils"
	"github.com/copo888/channel_app/lepay/internal/svc"
	"github.com/copo888/channel_app/lepay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

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

	logx.Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	if err2 != nil || channelBankMap.MapCode == "" {
		logx.Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
		return nil, errorx.New(responsex.BANK_CODE_INVALID, err2.Error(), "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName)
	}
	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	notifyUrl := l.svcCtx.Config.Server + "/api/proxy-pay-call-back"
	//notifyUrl = "http://9e7c-211-75-36-190.ngrok.io/api/proxy-pay-call-back"
	//data := url.Values{}
	//data.Set("partner", channel.MerId)
	//data.Set("service", "10201")
	//data.Set("tradeNo", req.OrderNo)
	//data.Set("amount", transactionAmount)
	//data.Set("notifyUrl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	//data.Set("bankCode", channelBankMap.MapCode)
	//data.Set("subsidiaryBank", req.ReceiptCardBankName)
	//data.Set("subbranch", req.ReceiptCardBranch)
	//data.Set("province", req.ReceiptCardProvince)
	//data.Set("city", req.ReceiptCardCity)
	//data.Set("bankCardNo", req.ReceiptAccountNumber)
	//data.Set("bankCardholder", req.ReceiptAccountName)

	type BankAccountInfoData struct {
		Bank string `json:"BANK"`
		BankBranch string `json:"BANK_BRANCH"`
		CardNumber string `json:"CARD_NUMBER"`
		CardHolderName string `json:"CARD_HOLDER_NAME"`
		City string `json:"CITY"`
		Province string `json:"PROVINCE"`
	}

	type Data struct {
		MerchantNumber string `json:"merchant_number"`
		Amount         string `json:"amount"`
		Sign           string `json:"sign"`
		NotifyUrl      string `json:"notify_url"`
		OrderNumber    string `json:"order_number"`
		BankAccountInfo BankAccountInfoData `json:"bank_account_info"`
	}

	// 組請求參數 FOR JSON
	data := Data{
		MerchantNumber: channel.MerId,
		OrderNumber: req.OrderNo,
		NotifyUrl: notifyUrl,
		Amount: transactionAmount,
		BankAccountInfo: BankAccountInfoData{
			Bank: channelBankMap.MapCode,
			BankBranch: req.ReceiptCardBranch,
			CardNumber: req.ReceiptAccountNumber,
			CardHolderName: req.ReceiptAccountName,
			City: req.ReceiptCardCity,
			Province: req.ReceiptCardProvince,
		},
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	// 請求渠道
	logx.Infof("代付下单请求地址:%s,代付請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).JSON(data)
	//ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).Form(data)
	logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	if ChnErr != nil {
		logx.Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		SystemOrderNumber string   `json:"system_order_number"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.SystemOrderNumber,
		OrderStatus:    "",
	}

	return resp, nil
}
