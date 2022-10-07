package logic

import (
	"bytes"
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/jindepay/internal/payutils"
	"github.com/copo888/channel_app/jindepay/internal/svc"
	"github.com/copo888/channel_app/jindepay/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io/ioutil"
	"net/http"
	"net/url"
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
	if req.PayType == "YK" && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"

	// 組請求參數
	data := url.Values{}
	data.Set("merchant_id", channel.MerId)
	data.Set("pay_type", req.ChannelPayType)
	data.Set("out_trade_no", req.OrderNo)
	data.Set("notify_url", notifyUrl)
	data.Set("money", req.TransactionAmount)
	data.Set("pay_name", req.UserId)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)
	data.Set("sign_type", "md5")
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

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
	//span := trace.SpanFromContext(l.ctx)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) (err error) {
			return errorx.New("redirects")
		},
	}
	requset, err := http.NewRequest("POST", channel.PayUrl, bytes.NewBufferString(data.Encode()))
	res, ChnErr := client.Do(requset)

	payUrl := ""
	if ChnErr != nil && strings.Index(ChnErr.Error(), "redirects") <= 0 {
		aaa := ChnErr.Error()
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, aaa)
	}
	pageUrl, _ := res.Location()

	logx.WithContext(l.ctx).Infof("Status: %s  payUrl: %s", res.Status, payUrl)
	if pageUrl == nil {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		bodyStr := string(body)
		idx := strings.Index(bodyStr, "orange")
		msg := bodyStr[idx+7 : idx+67]
		logx.WithContext(l.ctx).Infof("Status: %s  msg: %s", res.Status, bodyStr)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, msg)
	}
	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    pageUrl.String(),
		ChannelOrderNo: "",
	}

	return
}
