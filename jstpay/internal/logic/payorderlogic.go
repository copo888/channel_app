package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/jstpay/internal/payutils"
	"github.com/copo888/channel_app/jstpay/internal/svc"
	"github.com/copo888/channel_app/jstpay/internal/types"
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

	/** UserId 必填時使用 **/
	//if req.PayType == "YK" && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	timestamp := time.Now().Format("20060102150405")
	ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數 FOR JSON
	data := struct {
		CID        string `json:"CID"`
		BID        string `json:"BID"`
		BillSerial string `json:"BillSerial"`
		PayValue   string `json:"PayValue"`
		DesTime    string `json:"DesTime"`
		PayType    string `json:"PayType"`
		UID        string `json:"UID"`
		IPAddress  string `json:"IPAddress"`
		ReturnUrl  string `json:"ReturnUrl"`
		NoticeUrl  string `json:"NoticeUrl"`
		SignCode   string `json:"SignCode"`
	}{
		CID:        channel.MerId,
		BID:        "1",
		BillSerial: req.OrderNo,
		PayValue:   req.TransactionAmount,
		DesTime:    timestamp,
		PayType:    req.ChannelPayType,
		UID:        randomID,
		IPAddress:  ip,
		ReturnUrl:  req.PageUrl,
		NoticeUrl:  notifyUrl,
	}
	// 加簽
	m := payutils.CovertToMap(data)
	delete(m, "IPAddress")
	delete(m, "NoticeUrl")
	delete(m, "ReturnUrl")
	delete(m, "SignCode")
	source := payutils.JoinStringsInASCII(m, "&", false, true, channel.MerKey)
	sign := payutils.GetSign(source)
	logx.Info("加签参数: ", source)
	logx.Info("签名字串: ", sign)
	data.SignCode = sign

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
	// 若有證書問題 請使用
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
		Success bool   `json:"Success"`
		Message string `json:"Message, optional"`
		Result  struct {
			RedirectURL string `json:"RedirectURL"`
			BillSerial  string `json:"BillSerial"`
		} `json:"Result"`
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
	if !channelResp.Success {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Result.RedirectURL,
		ChannelOrderNo: channelResp.Result.BillSerial,
	}

	return
}
