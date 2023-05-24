package logic

import (
	"context"
	"crypto/aes"
	json "encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/feibaopay/internal/payutils"
	"github.com/copo888/channel_app/feibaopay/internal/svc"
	"github.com/copo888/channel_app/feibaopay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
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

	iv := "c11fa9ed92344d9d"

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
	////if req.ChannelOrderNo != "" {
	////	data.Set("order_no", req.ChannelOrderNo)
	////}
	//data.Set("appid", channel.MerId)
	//data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON
	data := struct {
		MerchantSlug     string `json:"merchant_slug"`
		MerchantOrderNum string `json:"merchant_order_num"`
	}{
		MerchantSlug: channel.MerId,
	}
	if req.OrderNo != "" {
		data.MerchantOrderNum = req.OrderNo
	}

	out, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	reqData := struct {
		MerchantSlug string `json:"merchant_slug"`
		Data         string `json:"data"`
	}{
		MerchantSlug: channel.MerId,
	}
	// 加簽
	sign := payutils.GetSignAES256CBC(string(out), channel.MerKey, iv, aes.BlockSize, l.ctx)
	reqData.Data = sign

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, reqData)

	span := trace.SpanFromContext(l.ctx)
	//res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(reqData)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Code             int    `json:"code"`
		Msg              string `json:"msg, optional"`
		MerchantSlug     string `json:"merchant_slug"`
		MerchantOrderNum string `json:"merchant_order_num"`
		Action           string `json:"action"`
		Order            string `json:"order"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	desOrder := struct {
		Amount            string `json:"amount"`
		Gateway           string `json:"gateway"`
		Status            string `json:"status"`
		MerchantOrderNum  string `json:"merchant_order_num"`
		MerchantOrderTime string `json:"merchant_order_time"`
	}{}

	desString, errDecode := payutils.AES256Decode(channelResp.Order, channel.MerKey, iv)

	if errDecode != nil {
		return nil, errDecode
	}

	fmt.Println(desString)
	dd := strings.ReplaceAll(desString, "\b", "")
	dby := []byte(dd)
	errj := json.Unmarshal(dby, &desOrder)
	//errj := json.Unmarshal(dby,&desOrder)
	if errj != nil {
		return nil, errj
	}

	orderAmount, errParse := strconv.ParseFloat(desOrder.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if desOrder.Status == "success" {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
