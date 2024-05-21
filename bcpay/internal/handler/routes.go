// Code generated by goctl. DO NOT EDIT.
package handler

import (
	"net/http"

	"github.com/copo888/channel_app/bcpay/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/pay",
				Handler: PayOrderHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/pay-query",
				Handler: PayOrderQueryHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/pay-call-back",
				Handler: PayCallBackHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/pay-query-balance-internal",
				Handler: PayQueryBalanceHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay",
				Handler: ProxyPayOrderHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay-query",
				Handler: ProxyPayOrderQueryHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay-call-back",
				Handler: ProxyPayCallBackHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/proxy-pay-query-balance-internal",
				Handler: ProxyPayQueryBalanceHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/test-conn",
				Handler: HealthTestConnHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api"),
	)
}
