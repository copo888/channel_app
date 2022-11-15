package svc

import (
	"fmt"
	"github.com/copo888/channel_app/ipay/internal/config"
	"github.com/go-redis/redis/v8"
	"github.com/neccoys/go-driver/mysqlx"
	"gorm.io/gorm"
	"strings"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *redis.Client
	MyDB        *gorm.DB
	PrivateKey  []byte
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

	// 读取私钥的位置，请自行更换
	PRIVATE_PEM_PASSPHASE_PATH := "-----BEGIN RSA PRIVATE KEY-----\n" +
		"MIIEowIBAAKCAQEAsaQh8bkLZoU/hBE0UuBLpRcy1vizZAQqkmtjLzjXZhIXtvBA\n" +
		"+QoL+m7oLPT+S0VIkQpKz5p1JbY/1gO/zrfBkj0ngpyyf/9x235pa6x5lFBFL1zD\n" +
		"UffPt7z30AXGvyBJGiQtPUjtvQUgL8p5xNAZVDTPGPorriErbpVP20Da3zxkSyTC\n" +
		"6yS+faPu5NHGTVGqiXKrRUyX1Rn01h29b5eW6QeB6MsvqSe/Y/kdy+HqLdIfn6eG\n" +
		"ZNfXJwZgeAzEQmFZjCKoRHQ66zSp4nTv2pFshTg07a5RXlp4/7XlB65314gZdyyD\n" +
		"jyOvN+COqlHFC+Rv+7/Ssdamqq3KKTjFzCI0CwIDAQABAoIBABBubC1dvm43OQ7Q\n" +
		"QJTB5n1Yzf0QeBdyQzXT9RKzIUlxtvvW8UuX4E/D3nn6F2OC/xlbaFwXn2pjlzgG\n" +
		"lMFcQe1y9qqgL+qjCDcTVFD/XSeY3S1qWS7Fy1Llic6WGjInnFtsqTqX+lWXmciR\n" +
		"4/2OeilN0TIwQcYTj17lNMPFFfm8Bt6G5dB9jLoqWLsjVrsfyYA28qLKxt/y5lff\n" +
		"9mgV95x3B3g+O+xPpZ9XB5J7A7pfAT40t1fQC5iZl6lmHC7FLF6f0ycrinlbJW4W\n" +
		"mdflxPOzepFdwF0rndj3oo0vZzxq9jqGD7XFZqtvDgwXUwFH/SoZobhaRSU5fyxo\n" +
		"TklzKcECgYEA2CRuoSbiJKMIso5hHV0xURmaAoapSMufk1gSAC39jRoK1or9Z29J\n" +
		"Qbznf7gETZ6XghrEJ9diYDXp3XEX8JxmkMGhwwHuB2frd9fX51lG7TBH7BqgIVsJ\n" +
		"LC3up8l0DrYjqWurSxZ39h8NuHI7j1N7P8JvmJb9T11csFbs75MPPpsCgYEA0mYm\n" +
		"tBQEjjXc6H11Oe/vgTe+3N4qoO7h6srkXmeteh9zbHlZue3fbDZJUqzMiq4GsDfc\n" +
		"oa0oSukfNBlg3pdPiNzoXoDTqqYZhsmpc6oaEMuWaDVFA4w6nyTTKyJGDGZ5BM0+\n" +
		"vYD8R6GTrqSlJ8JANMfvBTu0XlVl3IHTyfyJ/1ECgYBeVihi9dGmI/Jb3IDOjCpG\n" +
		"N2Jcz+F7AES5zqqsoWYU+9TXJvrK9muG4ag4ulxGdH20L3KF4R/y1hUorX/BaMHr\n" +
		"VFgCAQme+eBwAikdtH2ccIIzrrtNU6qBOdr8KJUbBqwx+ehdcYUSSyN16YXNXKZi\n" +
		"gb6rXttYlGssHAR13D2/GQKBgC41AwD/eHSm/aoNi4Y63JW7YW5uWFxZukHvZzIY\n" +
		"gO/WImpLSFpeFHhWf8npa051o8BltE3JkpTJF/JANJcOEgiTw3ClyFas/eQtO8rM\n" +
		"K8dOfuzJ7is2S9WRp9LMRygIBUH5tXK29jDhGmb7f834ilNNKYAzuYwSIznHRXUR\n" +
		"wljRAoGBAJJgUOT6/bmIvGvWOolljqHY5zRicJAtN+JrQdj0O4vFQpKtltudYVM4\n" +
		"pba+s45iQ/D4awEAZckHht0g4NPhI58a3ddwqh/RXT1RKTI7Pqe4KpvDvYjQ4pNi\n" +
		"l/lvzfpVaerz06nU5Lpl6sQZKB9hXkU25qKQcnhYpQFF9RK5vRul\n" +
		"-----END RSA PRIVATE KEY-----\n"

	return &ServiceContext{
		Config:      c,
		RedisClient: redisCache,
		MyDB:        db,
		PrivateKey:  []byte(PRIVATE_PEM_PASSPHASE_PATH),
	}
}
