package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/lepay/internal/payutils"
	"github.com/copo888/channel_app/lepay/internal/svc"
	"github.com/copo888/channel_app/lepay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)
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

	type Data struct {
		MerchantNumber string `json:"merchant_number"`
		Sign           string `json:"sign"`
		OrderNumber    string `json:"merchant_order_number"`
	}

	// 組請求參數 FOR JSON
	data := Data{
		MerchantNumber: channel.MerId,
		OrderNumber:    req.OrderNo,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).JSON(data)
	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelQueryResp := struct {
		OutTradeNo string `json:"system_order_number"`
		Amount     string `json:"amount"`
		Status     int64  `json:"status"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Status == 2 || channelQueryResp.Status == 6 {
		orderStatus = "20"
	} else if channelQueryResp.Status == 4 || channelQueryResp.Status == 5 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	resp = &types.ProxyPayOrderQueryResponse{}
	resp.Status = 1
	//resp.CallBackStatus =
	resp.OrderStatus = orderStatus
	resp.ChannelOrderNo = channelQueryResp.OutTradeNo
	resp.ChannelReplyDate = time.Now().Format("2006-01-02 15:04:05")
	//resp.ChannelCharge =

	return resp, nil
}
