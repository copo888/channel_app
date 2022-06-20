package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/vcpay2/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/vcpay2/internal/svc"
	"github.com/copo888/channel_app/vcpay2/internal/types"

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
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	// 組請求參數 FOR JSON
	data := struct {
		Timestamp     string `json:"timestamp"`
		Amount        string `json:"amount"`
		AppKey        string `json:"appKey"`
		BankName      string `json:"bankName"`
		BankBranch    string `json:"bankBranch"`
		AccountName   string `json:"accountName"`
		AccountNumber string `json:"accountNumber"`
		OrderID       string `json:"orderID"`
		Sign          string `json:"sign"`
	}{
		Timestamp:     timestamp,
		Amount:        transactionAmount,
		AppKey:        channel.MerId,
		BankName:      req.ReceiptCardBankName,
		BankBranch:    req.ReceiptCardBranch,
		AccountName:   req.ReceiptAccountName,
		AccountNumber: req.ReceiptAccountNumber,
		OrderID:       req.OrderNo,
	}

	// 加簽 JSON
	source := timestamp + "&" + data.Amount + "&" + data.AppKey + "&" + data.BankName + "&" +
		data.AccountName + "&" + data.AccountNumber + "&" + data.OrderID + "&" + channel.MerKey
	sign := payutils.GetSign(source)
	logx.Info("加签参数: ", source)
	logx.Info("签名字串: ", sign)
	data.Sign = sign

	// 請求渠道
	logx.Infof("代付下单请求地址:%s,代付請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Success bool   `json:"success"`
		Msg     string `json:"message"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: "",
		OrderStatus:    "",
	}

	return resp, nil
}
