package main

import (
	"flag"
	"fmt"
	"github.com/copo888/channel_app/txpay/rpc/internal/config"
	"github.com/copo888/channel_app/txpay/rpc/internal/server"
	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"
	"github.com/joho/godotenv"
	"github.com/neccoys/go-zero-extension/consul"
	"google.golang.org/grpc/health/grpc_health_v1"
	"log"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	configFile = flag.String("f", "etc/txpay.yaml", "the config file")
	envFile    = flag.String("env", "etc/.env", "the env file")
)

func main() {
	flag.Parse()

	if err := godotenv.Load(*envFile); err != nil {
		log.Fatal("Error loading .env file")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)
	srv := server.NewTxPayServer(ctx)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		txpay.RegisterTxPayServer(grpcServer, srv)

		grpc_health_v1.RegisterHealthServer(grpcServer, srv)

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	// 注册Consul服务
	if err := consul.RegisterService(c.ListenOn, c.Consul); err != nil {
		log.Panicln("Consul err:", err)
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
