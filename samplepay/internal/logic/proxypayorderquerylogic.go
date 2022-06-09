package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/samplepay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strings"
	"time"

	"github.com/copo888/channel_app/samplepay/internal/svc"
	"github.com/copo888/channel_app/samplepay/internal/types"

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

	logx.Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	data := url.Values{}
	data.Set("partner", channel.MerId)
	data.Set("service", "10301")
	data.Set("outTradeNo", req.OrderNo)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	logx.Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(10).Trace(span).Form(data)
	logx.Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	if ChnErr != nil {
		logx.Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelQueryResp := struct {
		Success    bool   `json:"success"`
		Msg        string `json:"msg"`
		OutTradeNo string `json:"outTradeNo"`
		Amount     string `json:"amount"`
		Status     string `json:"status"`
		StatusStr  string `json:"statusStr"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Msg)
	}
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Status == "1" {
		orderStatus = "20"
	} else if strings.Index("2,3,5", channelQueryResp.Status) > -1 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	resp.Status = 1
	//resp.CallBackStatus =
	resp.OrderStatus = orderStatus
	resp.ChannelReplyDate = time.Now().Format("2006-01-02 15:04:05")
	//resp.ChannelCharge =

	return resp, nil
}