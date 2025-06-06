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
	"github.com/copo888/channel_app/jinwangpay/internal/payutils"
	"github.com/copo888/channel_app/jinwangpay/internal/service"
	"github.com/copo888/channel_app/jinwangpay/internal/svc"
	"github.com/copo888/channel_app/jinwangpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
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
	if (strings.EqualFold(req.PayType, "YK") || strings.EqualFold(req.PayType, "YL")) && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	amountInt := int(amount)

	// 組請求參數
	data := url.Values{}
	//data.Set("mchId", channel.MerId)
	//data.Set("appId", "4e09054267a540be91d0b1b04ae116ec")
	//data.Set("productId", req.ChannelPayType)
	//data.Set("mchOrderNo", req.OrderNo)
	//data.Set("param1", req.UserId) //汇款 人户名(实名 制使用)
	//data.Set("currency", "cny")
	//data.Set("amount", fmt.Sprintf("%d", amountInt)) // 單位:分
	//data.Set("notifyUrl", notifyUrl)
	//data.Set("subject", "COPO")
	//data.Set("body", "COPO")

	dataJS := struct {
		MerchId    string `json:"mchId"`
		AppId      string `json:"appId"`
		ProductId  string `json:"productId"`
		MchOrderNo string `json:"mchOrderNo"`
		Param1     string `json:"param1"`
		Currency   string `json:"currency"`
		Amount     string `json:"amount"`
		NotifyUrl  string `json:"notifyUrl"`
		ReturnUrl  string `json:"returnUrl"`
		Subject    string `json:"subject"`
		Body       string `json:"body"`
		Sign       string `json:"sign"`
	}{
		MerchId:    channel.MerId,
		AppId:      "7f2812fcfb594d25bbca1d2ae05c8132",
		ProductId:  req.ChannelPayType,
		MchOrderNo: req.OrderNo,
		Param1:     req.UserId,
		Currency:   "CNY",
		Amount:     fmt.Sprintf("%d", amountInt),
		NotifyUrl:  notifyUrl,
		ReturnUrl:  req.PageUrl,
		Subject:    "COPO",
		Body:       "COPO",
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(dataJS, channel.MerKey, l.ctx)
	dataJS.Sign = sign
	dataJSStr, err := json.Marshal(dataJS)
	if err != nil {
		fmt.Println("error:", err)
	}
	data.Set("params", string(dataJSStr))

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		OrderNo:         req.OrderNo,
		ChannelCode:     channel.Code,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         data}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          string(res.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(res.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		RetCode    string `json:"retCode"`
		RetMsg     string `json:"retMsg, optional"`
		Sign       string `json:"sign, optional"`
		PayOrderId string `json:"payOrderId, optional"`
		PayParams  struct {
			PayAmount   float64 `json:"amount, optional"`
			Account     string  `json:"bankNo, optional"`
			PayPayUrl   string  `json:"payUrl, optional"`
			Bank        string  `json:"bankName"`
			RecepitName string  `json:"bankOwner, optional"`
			BankDetail  string  `json:"bankFrom, optional"`
		} `json:"payParams, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		OrderNo:         req.OrderNo,
		ChannelCode:     channel.Code,
		LogType:         constants.RESPONSE_FROM_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         fmt.Sprintf("%+v", channelResp),
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.RetCode != "SUCCESS" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RetMsg)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		//payAmount, err2 := strconv.ParseFloat(channelResp.PayParams.PayAmount, 64)
		//if err2 != nil {
		//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
		//}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.PayParams.RecepitName,
			CardNumber: channelResp.PayParams.Account,
			BankName:   channelResp.PayParams.Bank,
			BankBranch: channelResp.PayParams.BankDetail,
			Amount:     channelResp.PayParams.PayAmount,
			Link:       channelResp.PayParams.PayPayUrl,
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

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.PayParams.PayPayUrl,
		ChannelOrderNo: "",
	}

	return
}
