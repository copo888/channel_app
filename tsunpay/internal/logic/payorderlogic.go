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
	"github.com/copo888/channel_app/tsunpay/internal/payutils"
	"github.com/copo888/channel_app/tsunpay/internal/service"
	"github.com/copo888/channel_app/tsunpay/internal/svc"
	"github.com/copo888/channel_app/tsunpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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

	/** UserId 必填時使用 **/
	if len(req.SourceIp) == 0 {
		logx.WithContext(l.ctx).Errorf("SourceIp不可为空 SourceIp:%s", req.SourceIp)
		return nil, errorx.New(responsex.INVALID_USER_IP)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	//ip := utils.GetRandomIp()

	// 組請求參數 FOR JSON
	data := struct {
		ChannelCode string `json:"channel_code"`
		Username    string `json:"username"`
		Amount      string `json:"amount"`
		OrderNumber string `json:"order_number"`
		NotifyUrl   string `json:"notify_url"`
		ReturnUrl   string `json:"return_url, optional"`
		ClientIp    string `json:"client_ip, optional"`
		RealName    string `json:"real_name, optional"`
		Sign        string `json:"sign"`
	}{
		ChannelCode: req.ChannelPayType,
		Username:    channel.MerId,
		Amount:      req.TransactionAmount,
		OrderNumber: req.OrderNo,
		NotifyUrl:   notifyUrl,
		ReturnUrl:   req.PageUrl,
		//RealName:       req.UserId,
		ClientIp: req.SourceIp, // 渠道返回JSON 必填参数
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         data,
		TraceId:         l.traceID}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}
	//payUrl := channel.PayUrl +"?channel_code="+req.ChannelPayType+"&username="+channel.MerId+"&amount="+req.TransactionAmount+
	//	"&order_number=" + req.OrderNo + "&notify_url=" + notifyUrl + "&sign=" +sign
	//
	//if len(req.PageUrl) > 0 {
	//	payUrl += "&return_url=" + req.PageUrl
	//}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		HttpStatusCode int64  `json:"http_status_code"`
		ErrorCode      int64  `json:"error_code"`
		Message        string `json:"message"`
		Data           struct {
			SystemOrderNumber string `json:"system_order_number, optional"`
			OrderNumber       string `json:"order_number, optional"`
			Amount            string `json:"amount, optional"`
			CasherUrl         string `json:"casher_url, optional"`
			ReceiverAccount   string `json:"receiver_account, optional"`
			ReceiverBankName  string `json:"receiver_bank_name, optional"`
			ReceiverName      string `json:"receiver_name, optional"`
		} `json:"data, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.RESPONSE_FROM_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         fmt.Sprintf("%+v", channelResp),
		TraceId:         l.traceID}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.HttpStatusCode != 201 {

		// 寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          fmt.Sprintf("%s", channelResp.Message),
			TraceId:          l.traceID,
			ChannelErrorCode: fmt.Sprintf("%d", channelResp.HttpStatusCode),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	// 若需回傳JSON 請自行更改
	//if strings.EqualFold(req.JumpType, "json") {
	//	amount, err2 := strconv.ParseFloat(channelResp.Data.Amount, 64)
	//	if err2 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
	//	}
	//	// 返回json
	//	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
	//		CardName:   channelResp.Data.ReceiverName,
	//		CardNumber: channelResp.Data.ReceiverAccount,
	//		BankName:   channelResp.Data.ReceiverBankName,
	//		BankBranch: channelResp.Data.ReceiverBankName,
	//		Amount:     amount,
	//		Link:       "",
	//		Remark:     "",
	//	})
	//	if err3 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	//	}
	//	return &types.PayOrderResponse{
	//		PayPageType:    "json",
	//		PayPageInfo:    string(receiverInfoJson),
	//		ChannelOrderNo: "",
	//		IsCheckOutMer:  false, // 自組收銀台回傳 true
	//	}, nil
	//}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Data.CasherUrl,
		ChannelOrderNo: channelResp.Data.SystemOrderNumber,
	}

	return
}
