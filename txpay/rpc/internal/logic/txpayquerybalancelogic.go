package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayQueryBalanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTxPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TxPayQueryBalanceLogic {
	return &TxPayQueryBalanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TxPayQueryBalanceLogic) TxPayQueryBalance(in *txpay.TxPayQueryBalanceRequest) (*txpay.TxPayQueryInternalBalanceResponse, error) {
	// todo: add your logic here and delete this line

	return &txpay.TxPayQueryInternalBalanceResponse{}, nil
}
