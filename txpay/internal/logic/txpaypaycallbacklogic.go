package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayPayCallBackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxPayPayCallBackLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxPayPayCallBackLogic {
	return TxPayPayCallBackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxPayPayCallBackLogic) TxPayPayCallBack(req *types.PayCallBackRequest) error {
	// todo: add your logic here and delete this line

	return nil
}
