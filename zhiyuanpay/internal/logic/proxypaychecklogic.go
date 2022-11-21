package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/zhiyuanpay/internal/payutils"

	"github.com/copo888/channel_app/zhiyuanpay/internal/svc"
	"github.com/copo888/channel_app/zhiyuanpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayCheckLogic {
	return ProxyPayCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayCheckLogic) ProxyPayCheck(req *types.ProxyPayCheckRequest) (resp string, err error) {
	logx.WithContext(l.ctx).Infof("Enter ProxyPayCheck. channelName: %s, ProxyPayCallBackRequest: %+v", l.svcCtx.Config.ProjectName, req)

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
	// 檢查驗簽
	if isSameSign := payutils.VerifySign(req.Sign, *req, channel.MerKey); !isSameSign {
		return "fail", errorx.New(responsex.INVALID_SIGN)
	}

	var order typesX.Order
	if err = l.svcCtx.MyDB.Table("tx_orders").Where("order_no = ?", req.OrderId).Take(&order).Error; err != nil {
		logx.WithContext(l.ctx).Errorf("订单号不存在 %s", req.OrderId)
		return "fail", errorx.New(responsex.ORDER_NUMBER_NOT_EXIST, err.Error())
	}


	return "success", err
}
