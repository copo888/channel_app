package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/jindingpay/internal/payutils"
	"github.com/copo888/channel_app/jindingpay/internal/service"
	"github.com/copo888/channel_app/jindingpay/internal/svc"
	"github.com/copo888/channel_app/jindingpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s,orderNo: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	if strings.EqualFold(req.PayType, "YL") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://9f47-211-75-36-190.ngrok-free.app/api/pay-call-back"
	timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	amount := utils.FloatMul(req.TransactionAmount, "100")
	transactionAmount := strconv.FormatFloat(amount, 'f', 0, 64)


	// 組請求參數
	//data := url.Values{}
	//data.Set("merchId", channel.MerId)
	//data.Set("money", req.TransactionAmount)
	//data.Set("userId", randomID)
	//data.Set("orderId", req.OrderNo)
	//data.Set("time", timestamp)
	//data.Set("notifyUrl", notifyUrl)
	//data.Set("payType", req.ChannelPayType)
	//data.Set("reType", "LINK")
	//data.Set("signType", "MD5")

	// 組請求參數 FOR JSON
	data := struct {
		Version string `json:"version"`
		Cid string `json:"cid"`
		TradeNo string `json:"tradeNo"`
		Amount string `json:"amount"`
		AcctName string `json:"acctName"`
		PayType string `json:"payType"`
		RequestTime string `json:"requestTime"`
		NotifyUrl string `json:"notifyUrl"`
		ReturnType string `json:"returnType"`
		Sign string `json:"sign"`
	}{
		Version: "1.6",
		Cid: channel.MerId,
		TradeNo: req.OrderNo,
		Amount: transactionAmount,
		PayType: req.ChannelPayType,
		RequestTime: timestamp,
		NotifyUrl: notifyUrl,
		ReturnType: "0",
		AcctName: req.UserId,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data,
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Retcode string `json:"retcode"`
		RetMsg string `json:"retmsg"`
		Status string `json:"status"`
		PayeeName string `json:"payeeName"`
		PayeeBankName string `json:"payeeBankName"`
		BranchName string `json:"branchName"`
		PayeeAcctNo string `json:"payeeAcctNo"`
		PostScript string `json:"postScript"`
		RockTradeNo string `json:"rockTradeNo"`
		TradeNo string `json:"tradeNo"`
		Amount string `json:"amount"`
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
		Content:   fmt.Sprintf("%+v", channelResp),
		TraceId:   l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.Retcode != "0" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RetMsg)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		respAmount := utils.FloatDiv(channelResp.Amount, "100")
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.PayeeName,
			CardNumber: channelResp.PayeeAcctNo,
			BankName:   channelResp.PayeeBankName,
			BankBranch: channelResp.BranchName,
			Amount:     respAmount,
			Link:       "",
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: "",
			IsCheckOutMer:  false, // 自組收銀台回傳 true
		}, nil
	}

	//resp = &types.PayOrderResponse{
	//	PayPageType:    "url",
	//	PayPageInfo:    channelResp.PayUrl,
	//	ChannelOrderNo: "",
	//}
	respAmount := utils.FloatDiv(channelResp.Amount, "100")

	// 返回json
	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
		CardName:   channelResp.PayeeName,
		CardNumber: channelResp.PayeeAcctNo,
		BankName:   channelResp.PayeeBankName,
		BankBranch: channelResp.BranchName,
		Amount:     respAmount,
		Link:       "",
		Remark:     "",
	})
	if err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	return &types.PayOrderResponse{
		PayPageType:    "json",
		PayPageInfo:    string(receiverInfoJson),
		ChannelOrderNo: "",
		IsCheckOutMer:  true, // 自組收銀台回傳 true
	}, nil
}
