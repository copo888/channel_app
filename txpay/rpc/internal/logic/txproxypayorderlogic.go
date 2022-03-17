package logic

import (
	"context"
	_ "crypto/md5"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"
	"github.com/zeromicro/go-zero/core/logx"
	"io/ioutil"
	_ "net/http"
	"net/url"
)

type TxProxyPayOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTxProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TxProxyPayOrderLogic {
	return &TxProxyPayOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TxProxyPayOrderLogic) TxProxyPayOrder(in *txpay.TxProxyPayOrderRequest) (*txpay.TxProxyPayOrderResponse, error) {
	logx.Info("代付订单ChannelPayOrder:", in)
	var channel = txpay.Channel{}
	TxProxyPayOrderResp := &txpay.TxProxyPayOrderResponse{}
	code := "CHN000124" //需要預設redis app 啟動就把渠道資料暫存
	err := l.svcCtx.MyDB.Table("ch_channels").Where("code = ?", code).Find(channel).Error
	if err != nil {
		TxProxyPayOrderResp.Code = "EX001"
		TxProxyPayOrderResp.Msg = err.Error()
		return TxProxyPayOrderResp, err
	}

	merchantId := channel.MerId
	merchantKey := channel.MerKey
	orderNo := in.OrderNo
	amount := in.TransactionAmount
	bankCode := in.ReceiptCardBankCode
	accountName := in.ReceiptAccountName
	accountNo := in.ReceiptAccountNumber

	bankAddress := in.ReceiptCardProvince + "," + in.ReceiptCardBranch
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
	sign := utils.GetSign(source)
	data.Set("sign", sign)

	logx.Info("加签原串:{} 加签后字串:{}", source, sign)
	logx.Info("代付下单请求地址:{} 代付請求參數:{}", channel.ProxyPayUrl, data)

	resp, err := utils.SubmitForm(channel.ProxyPayUrl, data)

	if err != nil {
		logx.Error(err)
		return nil, err
	}
	logx.Info(resp.Status)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logx.Error(err)
		return nil, err
	}

	TxProxyPayOrderResp.Code = resp.Status
	TxProxyPayOrderResp.Msg = string(body)

	return TxProxyPayOrderResp, nil
}
