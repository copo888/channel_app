package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/mlbpay_php/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"

	"github.com/copo888/channel_app/mlbpay_php/internal/svc"
	"github.com/copo888/channel_app/mlbpay_php/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	traceID string
}

func NewProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderQueryLogic {
	return ProxyPayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
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

	//data := url.Values{}
	//data.Set("partner", channel.MerId)
	//data.Set("service", "10301")
	//data.Set("outTradeNo", req.OrderNo)

	// 組請求參數 FOR JSON
	data := struct {
		MerchNo  string `json:"merchNo"`
		OrderNo  string `json:"orderNo"`
	}{
		MerchNo :  channel.MerId,
		OrderNo :  req.OrderNo,
	}

	reqData := struct {
		Sign string `json:"sign"`
		Context []byte `json:"context"`
		EncryptType string `json:"encryptType"`
	}{}

	// 加簽
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

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).JSON(reqData)

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
		Code    int `json:"code"`
		Msg     string `json:"msg, optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.Code, channelQueryResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Msg)
	}

	channelQueryResp2 := struct {
		Sign    string `json:"sign,optional"`
		Context []byte `json:"context,optional"`
	}{}

	// 返回body 轉 struct
	if err3 := ChannelResp.DecodeJSON(&channelQueryResp2); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}

	respCon := struct {
		MerchNo string `json:"merchNo"`
		OrderNo string `json:"orderNo"`
		OutChannel string `json:"outChannel"`
		BusinessNo string `json:"businessNo"`
		OrderState string `json:"orderState"`
		Amount string `json:"amount"`
	}{}

	json.Unmarshal(channelQueryResp2.Context,&respCon)

	logx.WithContext(l.ctx).Errorf("代付订单查询渠道返回参数解密: %+v", respCon)

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if respCon.OrderState == "1" {
		orderStatus = "20"
	} else if strings.Index("2,4", respCon.OrderState) > -1 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	return &types.ProxyPayOrderQueryResponse{
		Status: 1,
		//CallBackStatus: ""
		OrderStatus:      orderStatus,
		ChannelReplyDate: time.Now().Format("2006-01-02 15:04:05"),
		//ChannelCharge =
	}, nil
}
