package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"

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
	dataInit := struct {
		MerchId   string `json:"merchantId"`
		Timestamp string `json:"timestamp"`
		Nonce     string `json:"nonce"`
	}{
		MerchId:   channel.MerId,
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		Nonce:     utils.GetRandomString(10, utils.ALL, utils.MIX),
	}
	dataBytes, err := json.Marshal(dataInit)

	encryptContent := utils.EnPwdCode(string(dataBytes), channel.MerKey)
	// 組請求參數 FOR JSON
	reqObj := struct {
		Id   string `json:"id"`
		Data string `json:"data"`
	}{
		Id:   channel.MerId,
		Data: encryptContent,
	}
	//// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%#v", channel.ProxyPayQueryBalanceUrl, reqObj)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).Timeout(10).Trace(span).JSON(reqObj)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	response := utils.DePwdCode(string(res.Body()), channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	balanceQueryResp := struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Balance string `json:"availableAmount"`
	}{}

	if err3 := json.Unmarshal([]byte(response), &balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if balanceQueryResp.Code != 0 {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.Code, balanceQueryResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.Msg)
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    balanceQueryResp.Balance,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
