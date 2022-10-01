package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/uzpay7777/internal/payutils"
	"github.com/copo888/channel_app/uzpay7777/internal/svc"
	"github.com/copo888/channel_app/uzpay7777/internal/types"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)
	uid := channel.MerId

	// Copo777（支转卡）:55639 (cate=3) 300～10000 (AK)
	// Copo888（卡转卡）:55640 (cate=remit) 500～10000 (YK) 100~10000
	// Copo999 话费微信 :55641 (cate=1) 30 50 100 (A5)


	// 組請求參數
	//data := url.Values{}
	//data.Set("uid", uid)
	//data.Set("amount", req.TransactionAmount)
	//data.Set("from_bankflag", from_bankflag)
	//data.Set("orderid", req.OrderNo)
	//data.Set("notify", notifyUrl)
	//data.Set("cate", req.ChannelPayType)
	//data.Set("userid", randomID)
	//data.Set("userip", ip)

	// 組請求參數 FOR JSON
	data := struct {
		Uid   string `json:"uid"`
		Amount     string `json:"amount"`
		Orderid   string `json:"orderid"`
		FromBankFlag      string `json:"from_bankflag"`
		FromComment  string `json:"from_comment"`
		NotifyUrl string `json:"notify"`
		Cate   string `json:"cate"`
		Userid string `json:"userid"`
		Userip string `json:"userip"`
		Sign      string `json:"sign"`
		Extend string `json:"extend"`
	}{
		Uid:   uid,
		Amount:     req.TransactionAmount,
		Orderid:   req.OrderNo,
		NotifyUrl: notifyUrl,
		Cate:   req.ChannelPayType,
		Userid: randomID,
		Userip: ip,
		FromBankFlag: "BANK",
		FromComment: req.MerchantOrderNo,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign
	data.Extend = uid

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
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Success    bool `json:"success, optional"`
		Msg     string `json:"msg, optional"`
		Info struct {
			Action string `json:"action, optional"`
			Qrurl string `json:"qrurl, optional"`
			Card struct {
				Bankflag       string `json:"bankflag, optional"`
				Cardnumber       string `json:"cardnumber, optional"`
				Cardname      string `json:"cardname, optional"`
				Location  string `json:"location, optional"`
				Comment string `json:"comment, optional"`
			} `json:"card, optional"`
		} `json:"info, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Success != true {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		amount, err2 := strconv.ParseFloat(req.TransactionAmount, 64)
		if err2 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
		}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.Info.Card.Cardname,
			CardNumber: channelResp.Info.Card.Cardnumber,
			BankName:   channelResp.Info.Card.Bankflag,
			Amount:     amount,
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
	}else {
		 resp = &types.PayOrderResponse{
			PayPageType:    "url",
			PayPageInfo:    channelResp.Info.Qrurl,
			ChannelOrderNo: "",
		}
		return resp, nil
	}


}
