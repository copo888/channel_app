package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/mlbpay_php/internal/payutils"
	"github.com/copo888/channel_app/mlbpay_php/internal/svc"
	"github.com/copo888/channel_app/mlbpay_php/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
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
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	//data := url.Values{}
	//if req.OrderNo != "" {
	//	data.Set("trade_no", req.OrderNo)
	//}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	//data.Set("appid", channel.MerId)
	//data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON
	data := struct {
		MerchNo string `json:"merchNo"`
		OrderNo string `json:"orderNo"`
	}{
		MerchNo: channel.MerId,
		OrderNo: req.OrderNo,
	}

	reqData := struct {
		Sign        string `json:"sign"`
		Context     []byte `json:"context"`
		EncryptType string `json:"encryptType"`
	}{}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)

	// 加簽 JSON
	out, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	source := string(out) + channel.MerKey
	sign := payutils.GetSign(source)
	logx.WithContext(l.ctx).Info("sign加签参数: ", source)
	logx.WithContext(l.ctx).Info("context加签参数: ", string(out))
	reqData.EncryptType = "MD5"
	reqData.Sign = sign
	reqData.Context = out

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(reqData)
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
		Code int    `json:"code"`
		Msg  string `json:"msg, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	// 渠道回覆處理
	channelResp2 := struct {
		Sign    string `json:"sign,optional"`
		Context []byte `json:"context,optional"`
	}{}

	if err = res.DecodeJSON(&channelResp2); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	respCon := struct {
		MerchNo    string `json:"merchNo"`
		OrderNo    string `json:"orderNo"`
		OutChannel string `json:"outChannel"`
		BusinessNo string `json:"businessNo"`
		OrderState string `json:"orderState"`
		Amount     string `json:"amount"`
		RealAmount string `json:"realAmount"`
		Remark     string `json:"remark"`
	}{}

	json.Unmarshal(channelResp2.Context, &respCon)
	logx.WithContext(l.ctx).Errorf("支付订单查询渠道返回参数解密: %s", respCon)
	orderAmount, errParse := strconv.ParseFloat(respCon.RealAmount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if respCon.OrderState == "1" {
		orderStatus = "1"
	} else if respCon.OrderState == "2" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
