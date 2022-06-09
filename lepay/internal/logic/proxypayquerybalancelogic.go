package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/lepay/internal/svc"
	"github.com/copo888/channel_app/lepay/internal/types"

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

	logx.Infof("Enter ProxyPayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)
	return nil, errorx.New(responsex.PAY_TYPE_INVALID)
}
