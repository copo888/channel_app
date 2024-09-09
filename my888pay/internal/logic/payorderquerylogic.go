package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/my888pay/internal/payutils"
	"github.com/copo888/channel_app/my888pay/internal/svc"
	"github.com/copo888/channel_app/my888pay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%+v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).
		Header("Content-Type", "text/plain").
		Header("X-Api-Key", channel.MerKey).
		Timeout(20).Trace(span).Body([]byte(encrypedData))

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
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

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorText)
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.AmountPaid, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.Data.Status == 2 { //1:未完成 2:完成 3:警示 4:退款 5:取消
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: channelResp.Data.AmountReceivable,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
