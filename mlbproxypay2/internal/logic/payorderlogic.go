package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/mlbproxypay2/internal/payutils"
	"github.com/copo888/channel_app/mlbproxypay2/internal/svc"
	"github.com/copo888/channel_app/mlbproxypay2/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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

	logx.Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	// 檢查 userId
	if req.PayType == "YK" && len(req.UserId) == 0 {
		logx.Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	data := url.Values{}
	data.Set("merchId", channel.MerId)
	data.Set("money", req.TransactionAmount)
	data.Set("userId", randomID)
	data.Set("orderId", req.OrderNo)
	data.Set("time", timestamp)
	data.Set("notifyUrl", notifyUrl)
	data.Set("payType", req.ChannelPayType)
	data.Set("reType", "LINK")
	data.Set("signType", "MD5")

	// 組請求參數 FOR JSON
	//data := struct {
	//	MerchId   string `json:"merchId"`
	//	Money     string `json:"money"`
	//	OrderId   string `json:"orderId"`
	//	Time      string `json:"time"`
	//	NotifyUrl string `json:"notifyUrl"`
	//	PayType   string `json:"payType"`
	//	sign      string
	//}{
	//	MerchId:   channel.MerId,
	//	Money:     req.TransactionAmount,
	//	OrderId:   req.OrderNo,
	//	Time:      timestamp,
	//	NotifyUrl: notifyUrl,
	//	PayType:   req.ChannelPayType,
	//}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.JumpType, "YK") {
	//	if req.UserId == "" {
	//		return nil, errorx.New(responsex.INVALID_USER_ID, err.Error())
	//	}
	//	data.Set("playerName", req.UserId)
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.Infof("支付下单请求地址:%s,支付請求參數:%#v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(10).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(10).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(10).Trace(span).Form(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code    string `json:"code"`
		Msg     string `json:"msg, optional"`
		Sign    string `json:"sign"`
		Money   string `json:"money"`
		OrderId string `json:"orderId"`
		PayUrl  string `json:"payUrl"`
		PayInfo struct {
			Name       string `json:"name"`
			Card       string `json:"card"`
			Bank       string `json:"bank"`
			Subbranch  string `json:"subbranch"`
			ExpiringAt string `json:"expiring_at"`
		} `json:"payInfo"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Code != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

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

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.PayUrl,
		ChannelOrderNo: "",
	}

	return
}
