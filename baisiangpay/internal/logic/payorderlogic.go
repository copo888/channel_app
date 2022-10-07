package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/baisiangpay/internal/payutils"
	"github.com/copo888/channel_app/baisiangpay/internal/svc"
	"github.com/copo888/channel_app/baisiangpay/internal/types"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	// 檢查 userId
	if req.PayType == "YK" && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID, "userId : "+req.UserId)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 組請求參數 FOR JSON
	data := struct {
		PayMerchantId string `json:"pay_merchant_id"`
		PayOrderId    string `json:"pay_order_id"`
		PayDatetime   string `json:"pay_datetime"`
		PaySubject    string `json:"pay_subject"`
		PayMethod     string `json:"pay_method"`
		PayRealName   string `json:"pay_real_name"`
		PayNotifyUrl  string `json:"pay_notify_url"`
		PayAmount     string `json:"pay_amount"`
		PaySign       string `json:"pay_sign"`
	}{
		PayMerchantId: channel.MerId,
		PayOrderId:    req.OrderNo,
		PayDatetime:   timestamp,
		PaySubject:    "COPO",
		PayMethod:     req.ChannelPayType,
		PayRealName:   req.UserId,
		PayNotifyUrl:  notifyUrl,
		PayAmount:     req.TransactionAmount,
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.PaySign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%#v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PayTransactionId string `json:"pay_transaction_id"`
			PayUrl           string `json:"pay_url"`
			PayBankName      string `json:"pay_bank_name"`
			PayBankBranch    string `json:"pay_bank_branch"`
			PayBankNo        string `json:"pay_bank_no"`
			PayBankOwner     string `json:"pay_bank_owner"`
			PayAmount        string `json:"pay_amount"`
		} `json:"data"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Message != "success" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Data.PayUrl,
		ChannelOrderNo: channelResp.Data.PayTransactionId,
	}

	return
}
