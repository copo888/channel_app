package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/baisiangpay/internal/payutils"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/baisiangpay/internal/svc"
	"github.com/copo888/channel_app/baisiangpay/internal/types"

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
	//channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	//if err2 != nil || channelBankMap.MapCode == "" {
	//	logx.Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, err2.Error(), "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName)
	//}
	// 組請求參數
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 2, 64)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 組請求參數 FOR JSON
	data := struct {
		PayMerchantId string `json:"pay_merchant_id"`
		PayOrderId    string `json:"pay_order_id"`
		PayDatetime   string `json:"pay_datetime"`
		PayBankName   string `json:"pay_bank_name"`
		PayBankAcc    string `json:"pay_bank_acc"`
		PayBankOwner  string `json:"pay_bank_owner"`
		PayBankBranch string `json:"pay_bank_branch"`
		PayNotifyUrl  string `json:"pay_notify_url"`
		PayAmount     string `json:"pay_amount"`
		PaySign       string `json:"pay_sign"`
	}{
		PayMerchantId: channel.MerId,
		PayOrderId:    req.OrderNo,
		PayDatetime:   timestamp,
		PayBankName:   req.ReceiptCardBankName,
		PayBankAcc:    req.ReceiptCardBankCode,
		PayBankOwner:  req.ReceiptAccountName,
		PayBankBranch: req.ReceiptCardBranch,
		PayNotifyUrl:  l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		PayAmount:     transactionAmount,
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.PaySign = sign

	// 請求渠道
	logx.Infof("代付下单请求地址:%s,請求參數:%#v", channel.ProxyPayUrl, data)
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
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PayTransactionId string `json:"pay_transaction_id"`
		} `json:"data"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.Message != "success" {
		logx.Errorf("代付渠道返回错误: %s: %s", channelResp.Code, channelResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.PayTransactionId,
		OrderStatus:    "",
	}

	return resp, nil
}
