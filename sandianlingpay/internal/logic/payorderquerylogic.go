package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/sandianlingpay/internal/payutils"
	"github.com/copo888/channel_app/sandianlingpay/internal/svc"
	"github.com/copo888/channel_app/sandianlingpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"

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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	data := url.Values{}
	if req.OrderNo != "" {
		data.Set("mhtorderno", req.OrderNo)
	}

	data.Set("opmhtid", channel.MerId)
	data.Set("random", randomID)

	// 組請求參數 FOR JSON
	//data := struct {
	//	merchId  string
	//	orderId  string
	//	time     string
	//	signType string
	//	sign     string
	//}{
	//	merchId:  channel.MerId,
	//	orderId:  req.OrderNo,
	//	time:     timestamp,
	//	signType: "MD5",
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	url := channel.PayQueryUrl + "?mhtorderno=" + req.OrderNo + "&opmhtid=" + channel.MerId + "&random=" + randomID + "&sign=" + sign
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", url, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Get(url).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Code   int    `json:"rtCode"`
		Msg    string `json:"msg"`
		Result struct {
			Accno          string `json:"accno"`
			Attach         string `json:"attach"`
			Currency       string `json:"currency"`
			Fromip         string `json:"fromip"`
			Lastnotifytime string `json:"lastnotifytime"`
			Notifystatus   int    `json:"notifystatus"`
			Notifyurl      string `json:"notifyurl"`
			Orderamount    int    `json:"orderamount"`
			Ordertime      string `json:"ordertime"`
			Paidamount     int    `json:"paidamount"`
			Payername      string `json:"payername"`
			Pforderno      string `json:"pforderno"`
			Settletime     string `json:"settletime"`
			Status         int    `json:"status"`
		} `json:"result, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	amountStr := strconv.Itoa(channelResp.Result.Orderamount)
	orderAmount := utils.FloatDiv(amountStr, "100")

	//orderAmount, errParse := strconv.ParseFloat(channelResp.Money, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.Result.Status == 1 {
		orderStatus = "1"
	} else if channelResp.Result.Status == 2 {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
