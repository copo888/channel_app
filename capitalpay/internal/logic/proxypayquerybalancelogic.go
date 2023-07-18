package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/capitalpay/internal/payutils"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"time"

	"github.com/copo888/channel_app/capitalpay/internal/svc"
	"github.com/copo888/channel_app/capitalpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayQueryBalanceLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayQueryBalanceLogic {
	return ProxyPayQueryBalanceLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayQueryBalanceLogic) ProxyPayQueryBalance() (resp *types.ProxyPayQueryInternalBalanceResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	timestamp := time.Now().Unix()

	t := strconv.FormatInt(timestamp, 10)
	data := url.Values{}
	data.Set("merchant_no", channel.MerId)
	data.Set("timestamp", t)
	data.Set("sign_type", "MD5")

	params := struct {
		Currency string `json:"currency"`
	}{
		Currency: "USDT",
	}

	paramsJs, err := json.Marshal(params)
	if err != nil {
		return nil,err
	}

	data.Set("params", string(paramsJs))

	//JSON 格式
	//data := struct {
	//	MerchId   string `json:"partner"`
	//}{
	//	MerchId: channel.MerId,
	//}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	signString := channel.MerId + string(paramsJs) + "MD5" + t + channel.MerKey
	sign := payutils.GetSign(signString)
	logx.WithContext(l.ctx).Info("加签参数: ", signString)
	logx.WithContext(l.ctx).Info("签名字串: ", sign)
	data.Set("sign", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%+v", channel.ProxyPayQueryBalanceUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() > 201 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	balanceQueryResp := struct {
		Code int `json:"code"`
		Timestamp int64 `json:"timestamp"`
		Message string `json:"message"`
		Params string `json:"params, optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if balanceQueryResp.Code != 200 {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.Code, balanceQueryResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.Message)
	}

	Params := struct {
		CurrentBalance string `json:"current_balance"`
		AvailableBalance string `json:"available_balance"`
		PendingSettlement string `json:"pending_settlement"`
		PendingPayout string `json:"pending_payout"`
	}{}

	err4 := json.Unmarshal([]byte(balanceQueryResp.Params), &Params)
	if err4  != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err4.Error())
	}
	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    Params.AvailableBalance,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
