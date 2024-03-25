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
	"github.com/copo888/channel_app/lepay/internal/payutils"
	"github.com/copo888/channel_app/lepay/internal/service"
	"github.com/copo888/channel_app/lepay/internal/svc"
	"github.com/copo888/channel_app/lepay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

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
	var branch string
	if req.ReceiptCardBranch == "" {
		branch = req.ReceiptCardBankName
	} else {
		branch = req.ReceiptCardBranch
	}

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
		Bank           string `json:"BANK"`
		BankBranch     string `json:"BANK_BRANCH"`
		CardNumber     string `json:"CARD_NUMBER"`
		CardHolderName string `json:"CARD_HOLDER_NAME"`
		City           string `json:"CITY"`
		Province       string `json:"PROVINCE"`
	}

	type Data struct {
		MerchantNumber  string              `json:"merchant_number"`
		Amount          string              `json:"amount"`
		Sign            string              `json:"sign"`
		NotifyUrl       string              `json:"notify_url"`
		OrderNumber     string              `json:"order_number"`
		BankAccountInfo BankAccountInfoData `json:"bank_account_info"`
	}

	// 組請求參數 FOR JSON
	data := Data{
		MerchantNumber: channel.MerId,
		OrderNumber:    req.OrderNo,
		NotifyUrl:      notifyUrl,
		Amount:         transactionAmount,
		BankAccountInfo: BankAccountInfoData{
			Bank:           channelBankMap.MapCode,
			BankBranch:     branch,
			CardNumber:     req.ReceiptAccountNumber,
			CardHolderName: req.ReceiptAccountName,
			City:           req.ReceiptCardCity,
			Province:       req.ReceiptCardProvince,
		},
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		ChannelCode: channel.Code,
		OrderNo:     req.OrderNo,
		LogType:     constants.DATA_REQUEST_CHANNEL,
		LogSource:   constants.API_DF,
		Content:     fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,代付請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).JSON(data)
	//ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).Form(data)
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChannelResp.Body())
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, string(ChannelResp.Body()))
	} else if ChannelResp.Status() < 200 && ChannelResp.Status() >= 300 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		msg := fmt.Sprintf("代付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, ChannelResp.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo: req.MerchantId,
			//MerchantOrderNo: req.OrderNo,
			ChannelCode:      channel.Code,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_DF,
			Content:          string(ChannelResp.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(ChannelResp.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		SystemOrderNumber string `json:"system_order_number"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		ChannelCode: channel.Code,
		OrderNo:     req.OrderNo,
		LogType:     constants.RESPONSE_FROM_CHANNEL,
		LogSource:   constants.API_DF,
		Content:     fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.SystemOrderNumber,
		OrderStatus:    "",
	}

	return resp, nil
}
