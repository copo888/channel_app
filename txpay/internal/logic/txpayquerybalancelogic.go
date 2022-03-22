package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxPayQueryBalanceLogic {
	return TxPayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxPayQueryBalanceLogic) TxPayQueryBalance() (resp *types.TxPayQueryInternalBalanceResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
