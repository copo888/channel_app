package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"

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
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	//orderTime := time.Now().Format("2006-01-02 15:04:05")
	nonce := utils.GetRandomString(10, utils.ALL, utils.MIX)
	//data := url.Values{}
	//data.Set("partner", channel.MerId)
	//data.Set("service", "10301")
	//data.Set("outTradeNo", req.OrderNo)

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
		Nonce:     nonce,
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

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryUrl, reqObj)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(10).Trace(span).JSON(reqObj)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	response := utils.DePwdCode(string(res.Body()), channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelQueryResp := struct {
		Code       int     `json:"code"`
		Msg        string  `json:"msg"`
		MerchantId string  `json:"merchantId"`
		OrderId    string  `json:"orderId"`
		TransId    string  `json:"transId"`
		Fee        float64 `json:"fee"`
		Amount     float64 `json:"amount"`
		Status     int     `json:"status"` //(0, '成功') (2, '处理中'), (11, '取消'), (7, '撤单')
		StatusStr  string  `json:"description"`
	}{}

	if err = json.Unmarshal([]byte(response), &channelQueryResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelQueryResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.Code, channelQueryResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Msg)
	}
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Status == 0 {
		orderStatus = "20"
	} else if channelQueryResp.Status == 11 || channelQueryResp.Status == 7 {
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
