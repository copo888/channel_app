package logic

import (
	"context"
	"github.com/copo888/channel_app/common/types"
	"github.com/copo888/channel_app/common/utils"
	"net/url"

	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxProxyPayOrderQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTxProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TxProxyPayOrderQueryLogic {
	return &TxProxyPayOrderQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TxProxyPayOrderQueryLogic) TxProxyPayOrderQuery(in *txpay.TxProxyPayQueryRequest) (*txpay.TxProxyPayQueryResponse, error) {
	logx.Info("代付查询订单ProxyPayQuery", in)

	channel := &types.ChannelData{}
	l.svcCtx.MyDB.Table("ch_channels").Where("code = CHN000124").Take(&channel)

	merchantId := channel.MerId
	merchantKey := channel.MerKey
	orderNo := in.OrderNo
	remark := ""

	source := "merchant_id=" + merchantId + "&merchant_order_id=" + orderNo + "&key=" + merchantKey
	sign := utils.GetSign(source)

	data := url.Values{}
	data.Set("merchant_id", merchantId)
	data.Set("merchant_order_id", orderNo)
	data.Set("remark", remark)
	data.Set("sign", sign)

	logx.Info("加签原串:{} 加签后字串:{}", source, sign)
	logx.Info("代付下单请求地址:{} 代付請求參數:{}", channel.ProxyPayQueryUrl, data)

	resp, err := utils.SubmitForm(channel.ProxyPayQueryUrl, data, l.ctx)

	if err != nil {
		logx.Error(err)
		return nil, err
	}

	if err != nil {
		logx.Error(err)
		return nil, err
	}
	TxProxyPayQueryResp := &txpay.TxProxyPayQueryResponse{
		Code: string(resp.Status()),
		Msg:  string(resp.Body()),
	}

	return TxProxyPayQueryResp, nil
}
