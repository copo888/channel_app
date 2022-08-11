package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/txpay/internal/txpayutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxProxyPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxProxyPayOrderLogic {
	return TxProxyPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxProxyPayOrderLogic) TxProxyPayOrder(req *types.TxProxyPayOrderRequest) (resp *types.OrderResponse, err error) {

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannel(l.svcCtx.Config.ChannelCode)
	if err != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	merchantId := channel.MerId
	merchantKey := channel.MerKey
	orderNo := req.OrderNo
	amount := req.TransactionAmount
	bankCode := req.ReceiptCardBankCode
	accountName := req.ReceiptAccountName
	accountNo := req.ReceiptAccountNumber

	bankAddress := req.ReceiptCardProvince + "," + req.ReceiptCardBranch
	userId := utils.GetRandomString(10, 0, 0)
	ip := utils.GetRandomIp()

	data := url.Values{}

	data.Set("merchant_id", merchantId)
	data.Set("merchant_order_id", orderNo)
	data.Set("user_level", "0")
	data.Set("user_credit_level", "-9_9")
	data.Set("payType", "912")
	data.Set("pay_amt", amount) //两位小数
	data.Set("notify_url", "")
	data.Set("return_url", "")
	data.Set("bank_code", bankCode)
	data.Set("bank_num", accountNo)
	data.Set("bank_owner", accountName)
	data.Set("bank_address", bankAddress)
	data.Set("user_id", userId)
	data.Set("user_ip", ip)
	data.Set("member_account", userId)
	data.Set("remark", "")

	source := "merchant_id=" + merchantId + "&merchant_order_id=" + orderNo + "&pay_type=" + "912" + "&pay_amt=" + amount + "&notify_url=" + "" +
		"&return_url=" + "" + "&bank_code=" + bankCode + "&bank_num=" + accountNo + "&bank_owner=" + accountName + "&bank_address=" + bankAddress +
		"&remark=" + "" + "&key=" + merchantKey

	sign := txpayutils.GetSign(source)
	data.Set("sign", sign)

	span := trace.SpanFromContext(l.ctx)
	res, err := gozzle.Post(channel.ProxyPayUrl).Timeout(20).Trace(span).Form(data)
	if err != nil {

	}

	proxyResp := struct {
		PayMessage int    `json:"pay_message"`
		PayResult  string `json:"pay_result"`
	}{}

	if err = res.DecodeJSON(&proxyResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if proxyResp.PayMessage != 1 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, proxyResp.PayResult)
	}

	return nil, nil
}
