package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/model"
	"github.com/copo888/channel_app/zhongshoupay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"

	"github.com/copo888/channel_app/zhongshoupay/internal/svc"
	"github.com/copo888/channel_app/zhongshoupay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {

	// 取得取道資訊
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannel(l.svcCtx.Config.ChannelCode)
	if err != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	// 取值
	merchantId := channel.MerId
	merchantKey := channel.MerKey
	orderNo := req.OrderNo

	// 組請求參數
	data := url.Values{}
	data.Set("partner", merchantId)
	data.Set("service", "10302")
	data.Set("outTradeNo", orderNo)

	// 加簽
	sign := payutils.SortAndSign2(data, merchantKey)
	data.Set("sign", sign)

	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	res, err := gozzle.Post(channel.PayQueryUrl).Timeout(10).Trace(span).Form(data)
	if err != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	}

	// 渠道回覆處理
	channelResp := struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Status  string `json:"status"` //0:处理中, 1:成功  2.失败
		Amount  string `json:"amount"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if !channelResp.Success {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, errParse.Error())
	}

	orderStatus := "0"
	if channelResp.Status == "1" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
