package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/testpay/internal/svc"
	"github.com/copo888/channel_app/testpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	traceID string
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {
	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %#v", l.svcCtx.Config.ProjectName, req)
	//resp := &types.ProxyPayOrderResponse{
	//	ChannelOrderNo: utils.GetRandomString(10, 0, 0),
	//	OrderStatus:    "20",
	//}

	return nil, errorx.New(responsex.TIMEOUT_CHANNEL, "Post \"https://globalgopay.com/api/payment\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)")
	//return resp, nil
}
