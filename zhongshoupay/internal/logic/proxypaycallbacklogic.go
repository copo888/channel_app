package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/zhongshoupay/internal/payutils"
	"github.com/copo888/channel_app/zhongshoupay/internal/svc"
	"github.com/copo888/channel_app/zhongshoupay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"

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

func (l *ProxyPayCallBackLogic) ProxyPayCallBack(req *types.ProxyPayCallBackRequest) error {
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}
	//檢查白名單
	if isWhite := utils.IPChecker(req.Ip, channel.WhiteList); !isWhite {
		logx.Errorf("IP: " + req.Ip)
		return errorx.New(responsex.IP_DENIED, "IP: "+req.Ip)
	}
	// 檢查驗簽
	if isSameSign := payutils.VerifySign(req.Sign, req, channel.MerKey); !isSameSign {
		return errorx.New(responsex.INVALID_SIGN)
	}

	var orderAmount float64
	if orderAmount, err = strconv.ParseFloat(req.Amount, 64); err != nil {
		return errorx.New(responsex.INVALID_SIGN)
	}
	var status = "1"
	if req.Status == "1" {
		status = "20"
	} else if strings.Index("2,3,5", req.Status) > -1 {
		status = "30"
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     req.OutTradeNo,
		ChannelOrderNo:      "",
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		ChannelResultNote:   req.StatusStr,
		Amount:              orderAmount,
		ChannelCharge:       0,
		UpdatedBy:           "",
	}

	// call boadmin callback api
	span := trace.SpanFromContext(l.ctx)
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.PayKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		return errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	url := "http://127.0.0.1:8080/dior/merchant-api/proxy-pay-call-back" // TODO: URL要抽出来
	res, errx := gozzle.Post(url).Timeout(10).Trace(span).Header("authenticationPaykey", payKey).JSON(proxyPayCallBackBO)
	if errx != nil {
		return errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if res.Status() != 200 {
		return errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("status:%d", res.Status()))
	}

	return nil
}
