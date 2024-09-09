package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/my888pay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/my888pay/internal/svc"
	"github.com/copo888/channel_app/my888pay/internal/types"

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

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 組請求參數 FOR JSON
	data := struct {
		OrderNumber         string `json:"order_number"`
		PlatformOrderNumber string `json:"platform_order_number"`
		ApiKey              string `json:"api_key"`
	}{
		//OrderNumber:         req.OrderNo,
		PlatformOrderNumber: req.OrderNo,
		ApiKey:              channel.MerKey,
	}

	// 加簽
	// 将原始数据序列化为JSON字符串
	jsonData, err := json.Marshal(data)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("%+v", err)
	}
	// 解码密钥
	hashKey, hashErr := payutils.DecodeBase64Key(l.svcCtx.Config.HashKey)
	if hashErr != nil {
		logx.WithContext(l.ctx).Errorf("密钥解码失败:", err)
	}
	// 加簽
	encrypedData, errEnc := payutils.Encrypt(string(jsonData), hashKey)
	if errEnc != nil {
		logx.WithContext(l.ctx).Errorf("加密失败: %+v", errEnc)
	}

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.PayQueryUrl).
		Header("Content-Type", "text/plain").
		Header("X-Api-Key", channel.MerKey).
		Timeout(20).Trace(span).Body([]byte(encrypedData))

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
		Code int `json:"code"`
		Data struct {
			Account             string  `json:"account"`
			Username            string  `json:"username"`
			OrderNumber         string  `json:"order_number"`
			PlatformOrderNumber string  `json:"platform_order_number"`
			AmountReceivable    float64 `json:"amount_receivable"`
			AmountReceive       float64 `json:"amount_receive"`
			AmountPayable       float64 `json:"amount_payable"`
			AmountPaid          float64 `json:"amount_paid"`
			AmountAdjustment    float64 `json:"amount_adjustment"`
			Fee                 float64 `json:"fee"`
			DepositFee          float64 `json:"deposit_fee"`
			RakePercent         float64 `json:"rake_percent"`
			RemittanceBank      string  `json:"remittance_bank"`
			RemittanceAccount   string  `json:"remittance_account"`
			Status              int     `json:"status"` //1: Incomplete 2: Completed 3: Alert 4: Refund 5: Cancel
			CreatedAt           string  `json:"created_at"`
			UpdatedAt           string  `json:"updated_at"`
		} `json:"data"`
		ErrorText string `json:"error_text, optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Code != 200 {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %d: %s", channelQueryResp.Code, channelQueryResp.ErrorText)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.ErrorText)
	}
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Data.Status == 20 {
		orderStatus = "20"
	} else if channelQueryResp.Data.Status == 30 || channelQueryResp.Data.Status == 31 {
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
