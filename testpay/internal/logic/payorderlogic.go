package logic

import (
	"context"
	"github.com/copo888/channel_app/testpay/internal/svc"
	"github.com/copo888/channel_app/testpay/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %#v", l.svcCtx.Config.ProjectName, req)

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    "https://docs.goldenf.vip/",
		ChannelOrderNo: "",
	}

	return
}
