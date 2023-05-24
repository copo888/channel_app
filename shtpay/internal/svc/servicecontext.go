package svc

import (
	"fmt"
	"github.com/copo888/channel_app/shtpay/internal/config"
	"github.com/go-redis/redis/v8"
	"github.com/neccoys/go-driver/mysqlx"
	"gorm.io/gorm"
	"strings"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *redis.Client
	MyDB        *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Redis
	redisCache := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    c.RedisCache.RedisMasterName,
		SentinelAddrs: strings.Split(c.RedisCache.RedisSentinelNode, ";"),
		DB:            c.RedisCache.RedisDB,
	})

	// DB
	db, err := mysqlx.New(c.Mysql.Host, fmt.Sprintf("%d", c.Mysql.Port), c.Mysql.UserName, c.Mysql.Password, c.Mysql.DBName).
		SetCharset("utf8mb4").
		SetLoc("UTC").
		Connect(mysqlx.Pool(50, 100, 180))

	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:      c,
		RedisClient: redisCache,
		MyDB:        db,
	}
}
