// Code generated by goctl. DO NOT EDIT.
package handler

import (
	"net/http"

	"github.com/copo888/channel_app/txpay/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/pay",
				Handler: TxPayOrderHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/pay-query",
				Handler: TxPayOrderQueryHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/pay-query-balance-internal",
				Handler: TxPayQueryBalanceHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/pay-call-back",
				Handler: TxPayPayCallBackHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay",
				Handler: TxProxyPayOrderHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay-query",
				Handler: TxProxyPayOrderQueryHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay-query-balance-internal",
				Handler: TxProxyPayQueryBalanceHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay-call-back",
				Handler: ProxyPayCallBackHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api"),
	)
}