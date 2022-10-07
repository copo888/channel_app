package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderQueryLogic {
	return ProxyPayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderQueryLogic) ProxyPayOrderQuery(req *types.ProxyPayOrderQueryRequest) (resp *types.ProxyPayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 組請求參數 FOR JSON
	paramsStruct := struct {
		MerchantOrderId string `json:"merchantOrderId"`
	}{
		MerchantOrderId: req.OrderNo,
	}
	paramsJson, err := json.Marshal(paramsStruct)
	paramsJsonStr := string(paramsJson[:])

	_, params := payutils.AesEncrypt(paramsJsonStr, l.svcCtx.Config.AesKey)

	merchantNo, _ := strconv.ParseInt(channel.MerId, 10, 64)

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo int64  `json:"merchantNo"`
		Signature  string `json:"signature"`
		Params     string `json:"params"`
	}{
		MerchantNo: merchantNo,
		Params:     params,
		Signature:  payutils.Md5V(paramsJsonStr+channel.MerKey, l.ctx),
	}

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelQueryResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderNo         string  `json:"orderNo"`
			OrderStatus     int64   `json:"orderStatus"`
			OrderAmount     float64 `json:"orderAmount"`
			PaidAmount      float64 `json:"paidAmount"`
			PlayerName      string  `json:"playerName"`
			MerchantOrderId string  `json:"merchantOrderId"`
			DepositName     string  `json:"depositName"`
		} `json:"data"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Code != 200 {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.Code, channelQueryResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Msg)
	}
	//渠道订单状态 0=待支付 1=成功 2=失败
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Data.OrderStatus == 1 {
		orderStatus = "20"
	} else if channelQueryResp.Data.OrderStatus == 2 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	return &types.ProxyPayOrderQueryResponse{
		Status:           1,
		ChannelOrderNo:   channelQueryResp.Data.OrderNo,
		OrderStatus:      orderStatus,
		ChannelReplyDate: time.Now().Format("2006-01-02 15:04:05"),
		//ChannelCharge =
	}, nil
}
