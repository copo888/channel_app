package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/lelifupay2/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/lelifupay2/internal/svc"
	"github.com/copo888/channel_app/lelifupay2/internal/types"

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
	timestamp := time.Now().Format("20060102150405")

	data := url.Values{}
	data.Set("txnType", "00")
	data.Set("txnSubType", "90")
	data.Set("secpVer", "icp3-1.1")
	data.Set("secpMode", "perm")
	data.Set("macKeyId", channel.MerId)
	data.Set("merId", channel.MerId)
	data.Set("accCat", "00")
	data.Set("timeStamp", timestamp)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("mac", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%#v", channel.ProxyPayQueryUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	balanceQueryResp := struct {
		RespCode string `json:"respCode"`
		RespMsg string `json:"respMsg"`
		SecpVer string `json:"secpVer"`
		SecpMode string `json:"secpMode"`
		MacKeyId string `json:"macKeyId"`
		MerId string `json:"merId"`
		ExtInfo string `json:"extInfo"`
		AccCat string `json:"accCat"`
		Balance string `json:"balance"`
		BalanceT1 string `json:"balanceT1"`
		FrozenAmount string `json:"frozenAmount"`
		CurrencyCode string `json:"currencyCode"`
		TimeStamp string `json:"timeStamp"`
		Mac string `json:"mac"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if balanceQueryResp.RespCode != "0000" {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.RespCode, balanceQueryResp.RespMsg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.RespMsg)
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    fmt.Sprintf("%f", utils.FloatDiv(balanceQueryResp.Balance, "100")),
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
