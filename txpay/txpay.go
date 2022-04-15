package main

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/logx"
	"log"

	"github.com/copo888/channel_app/txpay/internal/config"
	"github.com/copo888/channel_app/txpay/internal/handler"
	"github.com/copo888/channel_app/txpay/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var (
	configFile = flag.String("f", "txpay/etc/txpay.yaml", "the config file")
	envFile    = flag.String("env", "txpay/etc/.env", "the env file")
)

func main() {
	flag.Parse()

	if err := godotenv.Load(*envFile); err != nil {
		logx.Error(err.Error())
		log.Fatal("Error loading .env file")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
