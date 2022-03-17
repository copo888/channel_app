package config

import (
	"github.com/neccoys/go-zero-extension/consul"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Consul consul.Conf
	Mysql  struct {
		Host       string
		Port       int
		DBName     string
		UserName   string
		Password   string
		DebugLevel string
	}
	RedisCache struct {
		RedisSentinelNode string
		RedisMasterName   string
		RedisDB           int
	}
}
