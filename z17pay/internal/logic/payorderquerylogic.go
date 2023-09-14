package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/z17pay/internal/payutils"
	"github.com/copo888/channel_app/z17pay/internal/svc"
	"github.com/copo888/channel_app/z17pay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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
	data := url.Values{}
	data.Set("account", channel.MerId)
	data.Set("tradeno", req.OrderNo)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("token", sign)

	url := channel.PayQueryUrl + "?account=" + channel.MerId + "&tradeno=" + req.OrderNo + "&token=" + sign
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", url, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Get(url).Timeout(20).Trace(span).Form(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, fmt.Sprintf("Error HTTP Status: %d, Body:%s", res.Status(), string(res.Body())))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Id              int     `json:"id, optional"`
		Status          int     `json:"status, optional"` //交易状态: 1:提交成功, 2:匹配成功等待收款, 3: 取消订单, 4:交易成功
		Type            int     `json:"type, optional"`
		Tradeno         string  `json:"tradeno, optional"`
		StorePlayerName string  `json:"store_player_name, optional"`
		StorePlayerAcct string  `json:"store_player_acct, optional"`
		Inname          string  `json:"inname, optional"`
		Inbankname      string  `json:"inbankname, optional"`
		Inbanknum       string  `json:"inbanknum, optional"`
		Location        string  `json:"location, optional"`
		Inbankfullname  string  `json:"inbankfullname, optional"`
		TransBankname   string  `json:"trans_bankname, optional"`
		TransName       string  `json:"trans_name, optional"`
		TransBanknum    string  `json:"trans_banknum, optional"`
		TransBankMsg    string  `json:"trans_bank_msg, optional"`
		Money           float64 `json:"money, optional"`
		Storemoney      float64 `json:"storemoney, optional"` //商户实际金额
		Storerate       float64 `json:"storerate, optional"`
		Comment         string  `json:"comment, optional"`
		Trans           int     `json:"trans, optional"`
		TransUrl        string  `json:"trans_url, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Status != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, fmt.Sprintf("%d", channelResp.Status))
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.Money, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.Status == 4 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.Storemoney,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
