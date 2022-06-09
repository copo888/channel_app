package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/lepay/internal/svc"
	"github.com/copo888/channel_app/lepay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {

	logx.Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)
	return nil, errorx.New(responsex.PAY_TYPE_INVALID)
}
