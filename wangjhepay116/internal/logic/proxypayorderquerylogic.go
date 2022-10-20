package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/wangjhepay116/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"

	"github.com/copo888/channel_app/wangjhepay116/internal/svc"
	"github.com/copo888/channel_app/wangjhepay116/internal/types"

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
	timestamp := time.Now().Unix()

	// 組請求參數 FOR JSON
	data := struct {
		Mid     string `json:"mid"`
		Time    int64  `json:"time"`
		OrderNo string `json:"order_no"`
		Sign    string `json:"sign"`
	}{
		Mid:     channel.MerId,
		OrderNo: req.OrderNo,
		Time:    timestamp,
	}

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Header("Authorization", "api-key "+l.svcCtx.Config.ApiKey.ChannelKey).Timeout(20).Trace(span).JSON(data)

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
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderNo      string `json:"order_no"`
			No           string `json:"no"`
			Amount       float64 `json:"amount"`
			ActualAmount float64 `json:"actual_amount"`
			Fee          float64 `json:"fee"`
			CreatedTime  int64  `json:"created_time"`
			DepositTime  int64  `json:"deposit_time"`
			NotifyTime   int64  `json:"notify_time"`
			Status       string `json:"status"`
		} `json:"data"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Code != 200 {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.Code, channelQueryResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Message)
	}
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Data.Status == "succeeded" { // 创建:created 支付中:inprogress 成功: succeeded 失败:failed 超时过期:expired
		orderStatus = "20"
	} else if strings.Index("failed,expired", channelQueryResp.Data.Status) > -1 {
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
