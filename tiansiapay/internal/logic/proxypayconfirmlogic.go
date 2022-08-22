package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"

	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayConfirmLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayConfirmLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayConfirmLogic {
	return ProxyPayConfirmLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayConfirmLogic) ProxyPayConfirm(req *types.ProxyPayConfirmRequest) (resp string, err error) {
	logx.WithContext(l.ctx).Infof("Enter ProxyPayConfirm. channelName: %s, ProxyPayCallBackRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return "fail", errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}
	//檢查白名單
	if isWhite := utils.IPChecker(req.Ip, channel.WhiteList); !isWhite {
		logx.WithContext(l.ctx).Errorf("IP: " + req.Ip)
		return "白名单错误", errorx.New(responsex.IP_DENIED, "IP: "+req.Ip)
	}


	crypted, _ := base64.StdEncoding.DecodeString(req.Params)
	logx.WithContext(l.ctx).Infof("base64解密后的：%s", crypted)
	callBackBytes := payutils.AesDecrypt(crypted, []byte(l.svcCtx.Config.AesKey))
	logx.WithContext(l.ctx).Infof("解密后的明文：%s", string(callBackBytes))

	callBack :=  struct {
		Ip              string  `form:"ip, optional"`
		UserName        string  `json:"userName, optional"`
		PayAmout        float64 `json:"payAmout, optional"`
		MerchantOrderId string  `json:"merchantOrderId, optional"`
		BankNum         string  `json:"bankNum, optional"`
		BankOwner       string  `json:"bankOwner, optional"`
		OrderType       int64   `json:"orderType, optional"`
	}{}

	if err = json.Unmarshal(callBackBytes, &callBack); err != nil {
		return "fail", err
	}

	var order typesX.Order


	if err = l.svcCtx.MyDB.Table("tx_orders").Where("order_no = ?", callBack.MerchantOrderId).Take(&order).Error; err != nil {
		return "取得订单错误", errorx.New(responsex.IP_DENIED, err.Error())
	}

	if callBack.PayAmout != order.OrderAmount {
		return "金额错误", errorx.New(responsex.IP_DENIED, err.Error())
	}

	result := struct {
		Code int64   `json:"code"`
		Msg string `json:"msg"`
	}{}

	result.Code = 200
	resultJson, _ := json.Marshal(result)
	return string(resultJson), nil
}
