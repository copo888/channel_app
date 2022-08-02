package svc

import (
	"fmt"
	"github.com/copo888/channel_app/shahepay/internal/config"
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
	PRIVATE_PEM_PASSPHASE_PATH := "-----BEGIN RSA PRIVATE KEY-----\n"+
		"MIIEowIBAAKCAQEAnu9nJDY5YX0F9Xge/7XaZoVNCatKJPQnZdqxpcePNtMhxHCQ\n" +
		"Lieol/b3JOmF6TU4JwPx7rtQ3YA94NZYof4nhUEcnSU7p/ROSQUHX51PYyUs95O/\n" +
		"wA5ha1XR/n2IxzSpaVoaYcPu1Yi8vDJJscPgYUisO7ahtdwWvnoVJe70sNHRlbkP\n" +
		"vAECglanvNYoin1Hgmz7tNwwtXtn/KDHV79AY88YgpOS4U5c8I0vWKg587pRZTWL\n" +
		"mAZm3BB31HwLcYIIOd4WjR9utemvjdvHU5RmGLez2rotLF2nHVpw1G7HMPwtvQAQ\n" +
		"5mnpY2v3v+4jP3vFnDdFAOmM4Ie2US4q+bkQ1wIDAQABAoIBAAj1+Hu7LusHMIHR\n" +
		"fvXt291x4JEN/kUtGteMR/3PzYxKxRmdOxPPGptOykpjfDBU1tCkUUyjdQC4DUUS\n" +
		"8LZZbQL/U8ysX7utc4h8ZxkF9obhfrKKuwHqDaYOlaNikoagunh9IwWmFV4msnVt\n" +
		"5GfIYms5vLQ1LNLjEMk2euDDozog048fBdfSx5joSlns4HJYUgdY2K4XLj9dsg74\n" +
		"y1pnvhs4KXRIFdhc+YtKPBugA+ZDuOoOsD/pXhdztdybl495+KBPlzywck4GgPD1\n" +
		"Z5ODASxQQhp7vyAEoqt5mqA87GQI/B2kyto+ShwncuyrK5L+Fu/h3y0/pQPf7zzk\n" +
		"LBY37WECgYEAzfxkIiLq8Cwgez3F5dSpXV870Y92nqUY70Jb9oP8QGHTtTuu/BlW\n" +
		"kTw3+rJNP70GX8zL5UiqeEdJRjkP8CMKzGPeQbc3Z6HN36wDb5WMeIsxCYhbg/Ef\n" +
		"fiP0bEaqshgvCnIIdiHu1BhDvhFmFpg6BlgccWlcyO6soQauO0Y8rMcCgYEAxYZy\n" +
		"TndNZkKLBv/YL11wZgnGHu030gsfg9X9/HYnbLcRlLbRCt44X6yWpn2/QzfoK6ts\n" +
		"eA6Jc1bsGYs/ehSlpWdFNv1rY2ZcURAIY1F+bGM7z50i5amOwfFpeZVZV+NtcqW7\n" +
		"6uzc4tMwmmR2KFruMmqALuX9iiB8ktfyZJ+Zy3ECgYA6OmSb+b7sEa1E6Vtt8sXF\n" +
		"rUwdmy5u/kCkMeAJOZovIPhVvP9kKE1+VMWGSqznnamVnzDsKbR2t8AQ58SHn4BH\n" +
		"8ts7PG2fD/BAkEGQY3gIA2DjTvZ/v8OlRsiravaJzahOjZmyuzjmH83WhtiS/ok4\n" +
		"jePMc3pVGpMWGetauiogtQKBgGKtO8V/TCdd7t5cSb+/yjrvfw5MK6q+68uMyAmr\n" +
		"bR6ehiXo/p2TTk5dhhU+lKIb99x5EwMXIAuCzQglzFxMnEP5R1alW1SY+l10yPv0\n" +
		"5ld3a5XYRmq9PhgdZjfbKHsDntW7fhlqox6dqpY2weB/LKf7FHZZZ0Pt3s3tG6ax\n" +
		"JL/BAoGBAK6rIHc7olv6JrV7tMxmqgRJjy0f/Nsh0geBn28QoHfh52KALNUGHbHJ\n" +
		"tR82UCx+uoOHk06y0J74Hz890TZZZaJ4j3EEK7f9P5E2+NSCY8+labl4/Eu33ZRd\n" +
		"IAbxfUq31g8nYfCr6+txe5bvscYL4XJRb9DZrtkfk190i7s+CZ+c\n" +
		"-----END RSA PRIVATE KEY-----"

	return &ServiceContext{
		Config:      c,
		RedisClient: redisCache,
		MyDB:        db,
		PrivateKey:  []byte(PRIVATE_PEM_PASSPHASE_PATH),
	}
}
