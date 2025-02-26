package logic

import (
	"context"
	"encoding/base64"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/infapay/internal/svc"
	"github.com/copo888/channel_app/infapay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayQueryBalanceLogic {
	return ProxyPayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayQueryBalanceLogic) ProxyPayQueryBalance() (resp *types.ProxyPayQueryInternalBalanceResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 請求渠道s
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s", channel.ProxyPayQueryBalanceUrl)
	span := trace.SpanFromContext(l.ctx)

	credentials := []byte("copo:" + channel.MerKey)
	logx.WithContext(l.ctx).Infof("copo:" + channel.MerKey)
	token := base64.StdEncoding.EncodeToString(credentials)

	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).
		Header("Authorization", "Basic "+token).
		Header("Content-Type", "application/json").
		Header("Merchant-No", channel.MerId).
		Timeout(20).Trace(span).JSON(nil)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}
	//else if ChannelResp.Status() != 200 {
	//	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	//	return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	//}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	balanceQueryResp := struct {
		ReturnCode        string `json:"returnCode"`
		Message           string `json:"message"`
		MerchantAccountNo string `json:"merchantAccountNo"`
		ChannelOrderNo    string `json:"orderNo"`
		RequestTime       string `json:"requestTime"`
		AvailableBalance  string `json:"availableBalance"`
		AccountBalance    string `json:"accountBalance"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if balanceQueryResp.ReturnCode != "IPS00000" {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.ReturnCode, balanceQueryResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.Message)
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    balanceQueryResp.AvailableBalance,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
