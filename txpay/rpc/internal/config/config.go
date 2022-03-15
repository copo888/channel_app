package config

import (
	"github.com/neccoys/go-zero-extension/consul"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Consul consul.Conf
}
