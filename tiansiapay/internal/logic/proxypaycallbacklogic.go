package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayCallBackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayCallBackLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayCallBackLogic {
	return ProxyPayCallBackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayCallBackLogic) ProxyPayCallBack(req *types.ProxyPayCallBackRequest) (resp string, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayCallBack. channelName: %s, ProxyPayCallBackRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return "fail", errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}
	//檢查白名單
	if isWhite := utils.IPChecker(req.Ip, channel.WhiteList); !isWhite {
		logx.WithContext(l.ctx).Errorf("IP: " + req.Ip)
		return "fail", errorx.New(responsex.IP_DENIED, "IP: "+req.Ip)
	}

	crypted, _ := base64.StdEncoding.DecodeString(req.Params)
	logx.WithContext(l.ctx).Infof("base64解密后的：%s", crypted)
	callBackBytes := payutils.AesDecrypt(crypted, []byte(l.svcCtx.Config.AesKey))
	logx.WithContext(l.ctx).Infof("解密后的明文：%s", string(callBackBytes))

	callBack := struct {
		OrderNo         string  `json:"orderNo, optional"`
		OrderStatus     int64   `json:"orderStatus, optional"`
		OrderAmount     float64 `json:"orderAmount, optional"`
		PaidAmount      float64 `json:"paidAmount, optional"`
		PlayerName      string  `json:"playerName, optional"`
		MerchantOrderId string  `json:"merchantOrderId, optional"`
		DepositName     string  `json:"depositName, optional"`
	}{}

	if err = json.Unmarshal(callBackBytes, &callBack); err != nil {
		return "fail", err
	}

	var status = "0" //渠道回調狀態(0:處理中1:成功2:失敗)
	if callBack.OrderStatus == 1 {
		status = "1"
	} else if callBack.OrderStatus == 2 {
		status = "2"
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     callBack.MerchantOrderId,
		ChannelOrderNo:      callBack.OrderNo,
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		ChannelResultNote:   "",
		Amount:              callBack.PaidAmount,
		ChannelCharge:       0,
		UpdatedBy:           "",
	}

	// call boadmin callback api
	span := trace.SpanFromContext(l.ctx)
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.PayKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//BoProxyRespVO := &vo.BoadminProxyRespVO{}
	url := fmt.Sprintf("%s:%d/dior/merchant-api/proxy-call-back", l.svcCtx.Config.Merchant.Host, l.svcCtx.Config.Merchant.Port)

	res, errx := gozzle.Post(url).Timeout(20).Trace(span).Header("authenticationPaykey", payKey).JSON(proxyPayCallBackBO)
	logx.Info("回调后资讯: ", res)
	if errx != nil {
		logx.WithContext(l.ctx).Error(errx.Error())
		return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if res.Status() != 200 {
		return "fail", errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("status:%d", res.Status()))
	}

	result := struct {
		Code int64   `json:"code"`
		Msg string `json:"msg"`
	}{}

	result.Code = 200
	resultJson, _ := json.Marshal(result)
	return string(resultJson), nil
}
