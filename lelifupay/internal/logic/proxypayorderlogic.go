package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/lelifupay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/lelifupay/internal/svc"
	"github.com/copo888/channel_app/lelifupay/internal/types"

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
	amount := utils.FloatMul(req.TransactionAmount, "100")
	amountStr := fmt.Sprintf("%.0f", amount)
	orderDate := time.Now().Format("20060102")
	orderTime := time.Now().Format("150405")
	timestamp := time.Now().Format("20060102150405")

	data := url.Values{}
	data.Set("txnType", "52")
	data.Set("txnSubType", "10")
	data.Set("secpVer", "icp3-1.1")
	data.Set("secpMode", "perm")
	data.Set("macKeyId", channel.MerId)
	data.Set("orderDate", orderDate)
	data.Set("orderTime", orderTime)
	data.Set("merId", channel.MerId)
	data.Set("orderId", req.OrderNo)
	data.Set("txnAmt", amountStr)
	data.Set("currencyCode", "156")
	data.Set("accName", req.ReceiptAccountName)
	data.Set("accNum", req.ReceiptAccountNumber)
	data.Set("bankNum", channelBankMap.MapCode)

	//data.Set("notifyUrl", "https://ac82-211-75-36-190.jp.ngrok.io/api/proxy-pay-call-back")
	data.Set("notifyUrl", l.svcCtx.Config.Server+"/api/proxy-pay-call-back")
	data.Set("timeStamp", timestamp)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("mac", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		RespCode string `json:"respCode"`
		RespMsg  string `json:"respMsg"`
		SecpVer  string `json:"secpVer"`
		SecpMode string `json:"secpMode"`
		MacKeyId string `json:"macKeyId"`
		OrderDate string `json:"orderDate"`
		OrderTime string `json:"orderTime"`
		MerId string `json:"merId"`
		ExtInfo string `json:"extInfo"`
		OrderId string `json:"orderId"`
		TxnId string `json:"txnId"`
		TxnAmt string `json:"txnAmt"`
		CurrencyCode string `json:"currencyCode"`
		TxnStatus string `json:"txnStatus"`
		TxnStatusDesc string `json:"txnStatusDesc"`
		Mac string `json:"mac"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.RespCode != "0000" {
		logx.WithContext(l.ctx).Errorf("代付渠道返回错误: %s: %s", channelResp.RespCode, channelResp.RespMsg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RespMsg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.TxnId,
		OrderStatus:    "",
	}

	return resp, nil
}