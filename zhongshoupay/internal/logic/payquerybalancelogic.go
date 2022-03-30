package logic

import (
	"context"

	"github.com/copo888/channel_app/zhongshoupay/internal/svc"
	"github.com/copo888/channel_app/zhongshoupay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayQueryBalanceLogic {
	return PayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayQueryBalanceLogic) PayQueryBalance() (resp *types.PayQueryInternalBalanceResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
