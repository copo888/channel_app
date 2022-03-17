package svc

import (
	"fmt"
	"github.com/copo888/channel_app/txpay/rpc/internal/config"
	"github.com/gioco-play/go-driver/logrusz"
	"github.com/gioco-play/go-driver/mysqlz"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"log"
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

	log.Println(c.Mysql)
	// DB
	db, err := mysqlz.New(c.Mysql.Host, fmt.Sprintf("%d", c.Mysql.Port), c.Mysql.UserName, c.Mysql.Password, c.Mysql.DBName).
		SetCharset("utf8mb4").
		SetLoc("UTC").
		SetLogger(logrusz.New().SetLevel(c.Mysql.DebugLevel).Writer()).
		Connect(mysqlz.Pool(50, 100, 180))

	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:      c,
		RedisClient: redisCache,
		MyDB:        db,
	}
}
