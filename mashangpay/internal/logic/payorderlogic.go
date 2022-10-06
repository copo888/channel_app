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
	"github.com/copo888/channel_app/mashangpay/internal/payutils"
	"github.com/copo888/channel_app/mashangpay/internal/svc"
	"github.com/copo888/channel_app/mashangpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
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
	if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	amountFlo, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	//notifyUrl := "http://25d1-211-75-36-190.ngrok.io/api/pay-call-back"
	// 組請求參數
	data := struct {
		MerchId          string `json:"merchantCode"`
		OrderId          string `json:"merchantOrderId"`
		PayType          string `json:"paymentTypeCode"`
		Amount           string `json:"amount"` //单位﹔元，精确到分， 如﹔100.25
		NotifyUrl        string `json:"successUrl"`
		MerchantMemberId string `json:"merchantMemberId"`
		MerchantMemberIp string `json:"merchantMemberIp"`
		PayerName        string `json:"payerName"`
		Sign             string `json:"sign"`
	}{
		MerchId:          channel.MerId,
		OrderId:          req.OrderNo,
		PayType:          req.ChannelPayType,
		Amount:           fmt.Sprintf("%.2f", amountFlo), //两位小数
		NotifyUrl:        notifyUrl,
		MerchantMemberId: channel.MerId,
		MerchantMemberIp: utils.GetRandomIp(),
	}
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign
	data.PayerName = req.UserId

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

	// 組請求參數 FOR JSON
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
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

	Resp := struct {
		Result bool `json:"result"`
	}{}

	if err = json.Unmarshal(res.Body(), &Resp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	if Resp.Result != true {
		// 渠道回覆處理 [請依照渠道返回格式 自定義]
		channelResp := struct {
			Result   bool `json:"result"`
			ErrorMsg struct {
				Code     int    `json:"code"`
				ErrorMsg string `json:"errorMsg"`
				Descript string `json:"descript"`
			} `json:"errorMsg"`
		}{}

		if err = json.Unmarshal(res.Body(), &channelResp); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg.Descript)
	} else {
		// 渠道回覆處理 [請依照渠道返回格式 自定義]
		channelResp := struct {
			Result   bool   `json:"result"`
			ErrorMsg string `json:"errorMsg, optional"`
			Data     struct {
				GamerOrderId string `json:"gamerOrderId"`
				HttpUrl      string `json:"httpUrl"`
				HttpsUrl     string `json:"httpsUrl"`
			} `json:"data"`
		}{}
		if err = json.Unmarshal(res.Body(), &channelResp); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}

		resp = &types.PayOrderResponse{
			PayPageType:    "url",
			PayPageInfo:    channelResp.Data.HttpsUrl,
			ChannelOrderNo: "",
		}
	}

	// 返回body 轉 struct
	//if err = res.DecodeJSON(&channelResp); err != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	//}

	// 渠道狀態碼判斷
	//if channelResp.Result != true {
	//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg)
	//}

	// 若需回傳JSON 請自行更改
	//if strings.EqualFold(req.JumpType, "json") {
	//	amount, err2 := strconv.ParseFloat(channelResp.Money, 64)
	//	if err2 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
	//	}
	//	// 返回json
	//	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
	//		CardName:   channelResp.PayInfo.Name,
	//		CardNumber: channelResp.PayInfo.Card,
	//		BankName:   channelResp.PayInfo.Bank,
	//		BankBranch: channelResp.PayInfo.Subbranch,
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

	return
}
