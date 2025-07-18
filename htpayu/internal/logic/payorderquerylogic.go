package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/htpayu/internal/payutils"
	"github.com/copo888/channel_app/htpayu/internal/svc"
	"github.com/copo888/channel_app/htpayu/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	// 組請求參數
	data := url.Values{}
	data.Set("mhtorderno", req.OrderNo)
	data.Set("opmhtid", channel.MerId)
	data.Set("random", randomID)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("sign", sign)

	queryUrl := channel.PayQueryUrl + "?mhtorderno=" + req.OrderNo + "&opmhtid=" + channel.MerId + "&random=" + randomID + "&sign=" + sign
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", queryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Get(queryUrl).Timeout(20).Trace(span).Do()

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, chnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		RtCode int    `json:"rtCode"`
		Msg    string `json:"msg"`
		Result struct {
			Pforderno      string  `json:"pforderno,optional"`
			Orderamount    float64 `json:"orderamount,optional"`
			Paidamount     float64 `json:"paidamount,optional"`
			Currency       string  `json:"currency,optional"`
			Payername      string  `json:"payername,optional"`
			Paytype        string  `json:"paytype,optional"`
			Accno          string  `json:"accno,optional"`
			Attach         string  `json:"attach,optional"`
			Note           string  `json:"note,optional"`
			Ordertime      string  `json:"ordertime,optional"`
			Status         int     `json:"status,optional"`
			Settletime     string  `json:"settletime,optional"`
			Notifyurl      string  `json:"notifyurl,optional"`
			Notifystatus   int     `json:"notifystatus,optional"`
			Lastnotifytime string  `json:"lastnotifytime,optional"`
			Reference      string  `json:"reference,optional"`
			FromIP         string  `json:"fromIP,optional"`
		} `json:"result,optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.RtCode != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	orderStatus := "0"
	if channelResp.Result.Status == 1 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.Result.Orderamount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
