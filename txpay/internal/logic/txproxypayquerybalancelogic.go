package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxProxyPayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxProxyPayQueryBalanceLogic {
	return TxProxyPayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxProxyPayQueryBalanceLogic) TxProxyPayQueryBalance() (resp *types.TxProxyPayQueryInternalBalanceResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
