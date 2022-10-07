package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	service.ServiceConf
	Server string
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
	ApiKey struct {
		PublicKey  string
		PayKey     string
		ProxyKey   string
		ChannelKey string
	}
	ProjectName string
	Merchant    struct {
		Host string
		Port int
	}
}
