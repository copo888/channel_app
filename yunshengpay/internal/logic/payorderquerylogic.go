package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

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
	randomID := utils.GetRandomString(10, utils.ALL, utils.MIX)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
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
	dataInit := struct {
		MerchId   string `json:"merchantId"`
		OrderId   string `json:"orderId"`
		Nonce     string `json:"nonce"`
		TimeStamp string `json:"timestamp"`
	}{
		MerchId:   channel.MerId,
		OrderId:   req.OrderNo,
		TimeStamp: timestamp,
		Nonce:     randomID,
	}
	dataBytes, err := json.Marshal(dataInit)
	encryptContent := utils.EnPwdCode(string(dataBytes), channel.MerKey)
	// 組請求參數 FOR JSON
	reqObj := struct {
		Id   string `json:"id"`
		Data string `json:"data"`
	}{
		Id:   channel.MerId,
		Data: encryptContent,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, reqObj)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(10).Trace(span).JSON(reqObj)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(10).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	response := utils.DePwdCode(string(res.Body()), channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)
	// 渠道回覆處理
	channelResp := struct {
		Code  int     `json:"code"`
		Msg   string  `json:"msg"`
		State int     `json:"status"` //0为订单成功完成，1：创建，11：取消,6:未到账，7：预完成，8：超时取消。
		Money float64 `json:"amount"`
	}{}

	if err = json.Unmarshal([]byte(response), &channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.Money, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.State == 0 {
		orderStatus = "1"
	} else if channelResp.State == 11 || channelResp.State == 6 || channelResp.State == 8 {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.Money,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
