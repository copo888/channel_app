package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/yibipay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"

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
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	aesKey := "qHp8VxRtzQ7HpBfE"
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	//JSON 格式
	dataInit := struct {
		CurrencyList []string `json:"currencyList"`
		MerchantCode string   `json:"merchantCode"`
		MerchantId   string   `json:"merchantId"`
		Timestamp    string   `json:"timestamp"`
	}{
		CurrencyList: []string{"1"},
		MerchantCode: channel.MerId,
		Timestamp:    timeStamp,
	}

	dataBytes, err := json.Marshal(dataInit)
	params := utils.EnPwdCode(string(dataBytes), aesKey)
	sign := payutils.SortAndSignSHA256FromObj(dataInit, channel.MerKey)
	logx.WithContext(l.ctx).Infof("加签原串:%s，Encryption: %s，Signature: %s", string(dataBytes)+channel.MerKey, params, sign)
	data := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params"`    //参数密文
		Signature    string `json:"signature"` //参数签名(params + md5key)
	}{
		MerchantCode: channel.MerId,
		Params:       params,
		Signature:    sign,
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%+v", channel.ProxyPayQueryBalanceUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).Timeout(20).Trace(span).JSON(data)

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
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params,optional"`
		Sign         string `json:"signature"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	paramsDecode := utils.DePwdCode(balanceQueryResp.Params, aesKey)
	logx.WithContext(l.ctx).Infof("paramsDecode: %s", paramsDecode)
	balanceQueryResp2 := struct {
		Code        string        `json:"code"`
		BalanceList []balanceData `json:"data,optional"`
		Message     string        `json:"message,optional"`
		Timestamp   string        `json:"timestamp,optional"`
	}{}
	if err = json.Unmarshal([]byte(paramsDecode), &balanceQueryResp2); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}

	if balanceQueryResp2.Code != "200" {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp2.Code, balanceQueryResp2.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp2.Message)
	}

	var balanceStr string
	for _, balance := range balanceQueryResp2.BalanceList {
		if balance.Currency == "1" {
			balanceStr = balance.Balance
		}
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    balanceStr,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}
	return
}

type balanceData struct {
	Balance  string `json:"balance"`
	Currency string `json:"currency"`
}
