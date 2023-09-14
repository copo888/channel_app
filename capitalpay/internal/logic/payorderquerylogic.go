package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/capitalpay/internal/payutils"
	"github.com/copo888/channel_app/capitalpay/internal/svc"
	"github.com/copo888/channel_app/capitalpay/internal/types"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

type Params struct{
	Product string `json:"product"`
	MerchantRef string `json:"merchant_ref"`
	SystemRef string `json:"system_ref"`
	Amount string `json:"amount"`
	PayAmount string `json:"pay_amount"`
	Status int64 `json:"status"` // 0:unpaid 1:paid
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
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	timestamp := time.Now().Unix()
	t := strconv.FormatInt(timestamp, 10)
	// 組請求參數
	data := url.Values{}
	if req.OrderNo != "" {
		data.Set("trade_no", req.OrderNo)
	}
	if req.ChannelOrderNo != "" {
		data.Set("order_no", req.ChannelOrderNo)
	}
	data.Set("merchant_no", channel.MerId)
	data.Set("timestamp", t)
	data.Set("sign_type", "MD5")

	params := struct {
		MerchantRefs []string `json:"merchant_refs"`
	}{}

	params.MerchantRefs = append(params.MerchantRefs, req.OrderNo)

	paramsJs, err := json.Marshal(params)
	if err != nil {
		return nil,err
	}

	data.Set("params", string(paramsJs))

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
	signString := channel.MerId + string(paramsJs) + "MD5" + t + channel.MerKey
	sign := payutils.GetSign(signString)
	logx.WithContext(l.ctx).Info("加签参数: ", signString)
	logx.WithContext(l.ctx).Info("签名字串: ", sign)
	data.Set("sign", sign)

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() > 201 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Code  int `json:"code"`
		Timestamp int64 `json:"timestamp"`
		Message string `json:"message"`
		Params string `json:"params"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	var p []Params

	err4 := json.Unmarshal([]byte(channelResp.Params), &p)
	if err4  != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err4.Error())
	}
	orderAmount, errParse := strconv.ParseFloat(p[0].Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if p[0].Status == 1 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
