package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/txpay/internal/txpayutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxProxyPayOrderQueryLogic {
	return TxProxyPayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxProxyPayOrderQueryLogic) TxProxyPayOrderQuery(req *types.TxProxyPayOrderQueryRequest) (resp *types.OrderResponse, err error) {

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannel(l.svcCtx.Config.ChannelCode)
	if err != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	merchantId := channel.MerId
	merchantKey := channel.MerKey
	orderNo := req.OrderNo
	remark := ""

	source := "merchant_id=" + merchantId + "&merchant_order_id=" + orderNo + "&key=" + merchantKey
	sign := txpayutils.GetSign(source)

	data := url.Values{}
	data.Set("merchant_id", merchantId)
	data.Set("merchant_order_id", orderNo)
	data.Set("remark", remark)
	data.Set("sign", sign)

	span := trace.SpanFromContext(l.ctx)
	res, err := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).Form(data)
	if err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	}

	proxyResp := struct {
		PayMessage int    `json:"pay_message"`
		PayResult  string `json:"pay_result"`
	}{}

	if err = res.DecodeJSON(&proxyResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if proxyResp.PayMessage != 1 {
		return nil, errorx.New(responsex.ORDER_NUMBER_NOT_EXIST, proxyResp.PayResult)
	}

	return nil, nil
}
