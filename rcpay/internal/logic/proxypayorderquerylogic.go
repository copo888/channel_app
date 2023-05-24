package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/rcpay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/rcpay/internal/svc"
	"github.com/copo888/channel_app/rcpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderQueryLogic {
	return ProxyPayOrderQueryLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayOrderQueryLogic) ProxyPayOrderQuery(req *types.ProxyPayOrderQueryRequest) (resp *types.ProxyPayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)
	timestamp := time.Now().Unix()

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	//data := url.Values{}
	//data.Set("partner", channel.MerId)
	//data.Set("service", "10301")
	//data.Set("outTradeNo", req.OrderNo)

	data := struct {
		Username    string `json:"username"`
		OrderNumber string `json:"order_number"`
		Timestamp   string `json:"timestamp"`
		Sign        string `json:"sign"`
	}{
		Username:    channel.MerId,
		OrderNumber: req.OrderNo,
		Timestamp:   fmt.Sprintf("%v", timestamp),
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)
	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

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
		HttpStatusCode int    `json:"http_status_code"`
		ErrorCode      int    `json:"error_code"`
		Message        string `json:"message"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.HttpStatusCode >= 200 && channelQueryResp.HttpStatusCode < 300 {
	} else {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %i: %s", channelQueryResp.HttpStatusCode, channelQueryResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Message)
	}

	channelResp2 := struct {
		Data struct {
			Amount            string `json:"amount"`
			ChannelCode       string `json:"channel_code"`
			ConfirmedAt       string `json:"confirmed_at"`
			CreatedAt         string `json:"created_at"`
			Currency          string `json:"currency"`
			CurrencyAmount    string `json:"currency_amount"`
			Fee               string `json:"fee"`
			MatchedAt         string `json:"matched_at"`
			NotifiedAt        string `json:"notified_at"`
			NotifyUrl         string `json:"notify_url"`
			OrderNumber       string `json:"order_number"`
			Rate              string `json:"rate"`
			Status            int    `json:"status"`
			SystemOrderNumber string `json:"system_order_number"`
			ToWalletAddress   string `json:"to_wallet_address"`
			Sign              string `json:"sign"`
		} `json:"data"`
	}{}

	// 返回body 轉 struct
	if err = ChannelResp.DecodeJSON(&channelResp2); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelResp2.Data.Status == 4 || channelResp2.Data.Status == 5 {
		orderStatus = "20"
	} else if channelResp2.Data.Status == 6 || channelResp2.Data.Status == 7 || channelResp2.Data.Status == 8 {
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
