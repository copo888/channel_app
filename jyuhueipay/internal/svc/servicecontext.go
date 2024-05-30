package svc

import (
	"fmt"
	"github.com/copo888/channel_app/jyuhueipay/internal/config"
	"github.com/go-redis/redis/v8"
	"github.com/neccoys/go-driver/mysqlx"
	"gorm.io/gorm"
	"strings"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *redis.Client
	MyDB        *gorm.DB
	PrivateKey  string
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

	privateKey := "-----BEGIN PRIVATE KEY-----" +
		"MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAJPF+5zPGz7Tj3wF" +
		"+WGBgUi8t6G/f41HHfTau/1c1GU2XkzztNpSC6jFAV3tv7dH6t9R5BOI0zaNL2e6" +
		"AzxoAenxUCzpBM2ml/WGl0NTH+ug70whADP9aHYLLz3R2rs357Wy9I65zkfHj0rr" +
		"f6eatpPHVJFRZKjjG87FRkWsTllXAgMBAAECgYA2go65HlCvkTzZhB6k0C/PlAWs" +
		"/cXgaH5tZw53fHnAyBhrDG8HJcK1Q86oRl41DqteK4gViKBY/CWjfqNY8gjaIBkv" +
		"v4GRrXv39noRUqt4yyMjUV0DyX05GyHyIepv+PJsGTCYdbSvUsiSRof9hjP5cmiL" +
		"1xfSyCYM8kVbAp/NgQJBAOs0KdqPa5+qTrkmLdZ9yM/jKDD3wMpQK0+v241uzNQk" +
		"At4fSfKfx0HeMRWlN8EApmu79uVxXbS6d90uWlf518ECQQCg1tUqQKKJlbDblsNp" +
		"KVkJ6duFir2znnQ31qCmtPUOltzCVigWiZeaLvoj2jQPcmQwRlTEoUeaLAqw4HhL" +
		"4bcXAkAcRer6EN95E/nem85KyFgJVV8Rc9bC+xwZuiQEPlt07NdDaDN1bOuA6Xqx" +
		"BWlMVA2tSa0GK7h97yRVfCr0BF6BAkBeyPxqoBipeFv3rMuQR9u/KeBCvNa66eCX" +
		"n5ahZ/cbF/EVXezWC1QaeyDSzB7uE33TrT77eghWZSoPvtll7qkVAkArBPR9cri7" +
		"5jdZ+UK4tf9hx0fo/3QYkLjeNUWRcgBAgtVcuUFzp5LTLYda0NjR8pE7wla+FW+T" +
		"bELNYtRP0Esv" +
		"-----END PRIVATE KEY-----"

	//privateKey := "-----BEGIN PRIVATE KEY-----" +
	//	"MIICdwIBADANBgkqhkiG9w0BAQEFAASCAmEwggJdAgEAAoGBAIibKOkSsKlo4HJF" +
	//	"H5+C8Y+6KSlA/YPEmN0CqFPqDbOolEy8AbkMZDBaFy8JoxLKc9fOGZY/XhY8N7P4" +
	//	"rM3MLRxM+KzkwAZP30WaiqQspa5rK22XdF0HfjMPeLHITxZqeS0A1aoSmKU2eIl/" +
	//	"dpN/6cc0DFvuL/t+xR5v6E1HcrC7AgMBAAECgYBmWuPXZ2KpPOTXmgVszn9S8ui+" +
	//	"eWy624ayKriXT4r+r3SW3lPoJGm5dPdkDjN68+jCrTGsy0QjIvGVzuEjvjWZm1Kl" +
	//	"nFKlp7iBvpJJGr8+As6rXLVP0pP/QHbz7LzB0qesrj2HKgF5S70763+GfTDvk/CQ" +
	//	"1WQNdvkM2OJDEv7N6QJBAO6ibVV+WNiMiCFq4t8+Bq+x9kkkSEgTbIe93joRTDe7" +
	//	"nbqQtQ5pgIMQurN8CjSXoWF6lY9p8rxQ+zBwXNuhZHcCQQCSjAYlPgybdLPVU55u" +
	//	"KItFxmnqiBcVeBSuPqX41E5L3GEJuzFrl5s0IUdR1rXf8faV1reMiWNwnhXBlk/p" +
	//	"hDrdAkEAnpjM2VkTazha8Pq8tWnfv70i1hGLCHwAUWba3vTIFvJWLbwm2OE9S94+" +
	//	"dzMlBTcRRlvWMm5TqNyZVOQYks98mQJBAIvHPz1al8/XWohJf732shDVlcT8FXiG" +
	//	"1sL0Qn66kgvNokkT4amMK59ndo1azJNUSSzWZrCHgu+x+XJymrpTQ4kCQDZXWdvc" +
	//	"5OE17dl/yEXkqDQS8OvlQZ43G5WPsm6l/Ij+vaZfZM7p5dNNZslwvrbW/rExyXQ/" +
	//	"za+OF4Sj3qSRdPk=" +
	//	"-----END PRIVATE KEY-----"

	return &ServiceContext{
		Config:      c,
		RedisClient: redisCache,
		MyDB:        db,
		PrivateKey:  privateKey,
	}
}
