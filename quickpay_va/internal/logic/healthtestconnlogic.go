package logic

import (
	"context"

	"github.com/copo888/channel_app/quickpay_va/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type HealthTestConnLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthTestConnLogic(ctx context.Context, svcCtx *svc.ServiceContext) HealthTestConnLogic {
	return HealthTestConnLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HealthTestConnLogic) HealthTestConn() (resp string, err error) {

	resp = "success"

	return resp, nil
}
