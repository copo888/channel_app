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
	"github.com/copo888/channel_app/wanjioupay/internal/payutils"
	"github.com/copo888/channel_app/wanjioupay/internal/service"
	"github.com/copo888/channel_app/wanjioupay/internal/svc"
	"github.com/copo888/channel_app/wanjioupay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	var payUrl string
	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://afb4-211-75-36-190.ngrok-free.app/api/pay-call-back"
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	//ip := utils.GetRandomIp() TET
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	data := url.Values{}
	data.Set("pay_memberid", channel.MerId)
	data.Set("pay_orderid", req.OrderNo)
	data.Set("pay_applydate", timestamp)
	data.Set("pay_bankcode", req.ChannelPayType)
	data.Set("pay_notifyurl", notifyUrl)
	data.Set("pay_callbackurl", req.PageUrl)
	data.Set("pay_amount", req.TransactionAmount)

	//data.Set("payType", req.ChannelPayType)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("pay_md5sign", sign)
	data.Set("pay_productname", "deposit")
	//data.Set("pay_type", "JSON")

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:  req.MerchantId,
		ChannelCode: channel.Code,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:  req.MerchantId,
			ChannelCode: channel.Code,
			//MerchantOrderNo: req.OrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          string(res.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(res.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code   string `json:"status"` //1 为下单成功 0未下单失败
		Msg    string `json:"msg, optional"`
		H5Url  string `json:"h5_url, optional"`
		SdkUrl string `json:"sdk_url, optional"`
	}{}

	// <script>location.href='https://dk2.fanhuiwangluo.top/api/ysf?orderNo=FZYN20231008162757062MAQQEu';</script>
	respBody := string(res.Body())

	if strings.Index(respBody, "location.href") <= -1 {
		err = res.DecodeJSON(&channelResp)

		if err != nil || channelResp.Code != "1" {
			// 寫入交易日志
			if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
				MerchantNo:  req.MerchantId,
				ChannelCode: channel.Code,
				//MerchantOrderNo: req.OrderNo,
				OrderNo:          req.OrderNo,
				LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
				LogSource:        constants.API_ZF,
				Content:          fmt.Sprintf("%s", channelResp.Msg),
				TraceId:          l.traceID,
				ChannelErrorCode: channelResp.Code,
			}); err != nil {
				logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
			}
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
		}

	} else {

		payUrl = respBody[strings.Index(respBody, "'")+1 : strings.LastIndex(respBody, "'")]
		if len(payUrl) == 0 {
			// 寫入交易日志
			if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
				MerchantNo:  req.MerchantId,
				ChannelCode: channel.Code,
				//MerchantOrderNo: req.OrderNo,
				OrderNo:          req.OrderNo,
				LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
				LogSource:        constants.API_ZF,
				Content:          respBody,
				TraceId:          l.traceID,
				ChannelErrorCode: respBody,
			}); err != nil {
				logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
			}
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
		}
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:  req.MerchantId,
		ChannelCode: channel.Code,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   respBody}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    payUrl,
		ChannelOrderNo: "",
	}

	return
}
