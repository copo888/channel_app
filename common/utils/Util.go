package utils

import (
	"encoding/json"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

type RandomType int8
type UppLowType int8

const (
	ALL    RandomType = 0
	NUMBER RandomType = 1
	STRING RandomType = 2
)

const (
	MIX   UppLowType = 0
	UPPER UppLowType = 1
	LOWER UppLowType = 2
)

const (
	yyyy              string = "yyyy"
	yyyyMMdd          string = "yyyyMMdd"
	HHmmss            string = "HHmmss"
	yyyyMMddHHmm      string = "yyyyMMddHHmm"
	YYYYMMddHHmmss    string = "yyyyMMddHHmmss"
	YYYYMMddHHmmssSSS string = "yyyyMMddHHmmssSSS"
	YYYY_MM_dd        string = "yyyy-MM-dd"
	YYYYMMddHHmmss2   string = "yyyy-MM-dd HH:mm:ss"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RangeInt(min int, max int, n int) []int {
	arr := make([]int, n)
	var r int
	for r = 0; r <= n-1; r++ {
		arr[r] = rand.Intn(max) + min
	}
	return arr
}

//GetRandomString 生成随机字符串
func GetRandomString(length int, randomType RandomType, uppLowType UppLowType) string {
	var str string

	switch randomType {
	case NUMBER:
		str = "0123456789"
	case STRING:
		str = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	default:
		str = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}

	switch uppLowType {
	case UPPER:
		str = strings.ToUpper(str)
	case LOWER:
		str = strings.ToLower(str)
	}

	return string(result)
}

func GetRandomIp() string {
	IpArr := RangeInt(0, 255, 4)
	var ips []string
	for _, ip := range IpArr {
		ips = append(ips, strconv.Itoa(ip))
	}
	return strings.Join(ips, ".")
}

// IPChecker IP白名單確認
func IPChecker(myip string, whitelist string) bool {

	if myip == "localhost" || myip == "127.0.0.1" || myip == "0:0:0:0:0:0:0:1" || myip == "211.75.36.190" {
		return true
	}

	if whitelist == "" {
		return false
	}
	for _, ip := range strings.Split(whitelist, ",") {
		if !strings.Contains(ip, "/") {
			ip = ip + "/32"
		}
		_, ipnetA, _ := net.ParseCIDR(ip)
		if ipnetA == nil {
			continue
		}
		ipB := net.ParseIP(myip)

		if ipnetA.Contains(ipB) {
			return true
		}
	}
	return false
}

//FloatMul 浮點數乘法 (precision=3)
func FloatMul(s string, p string, precisions ...int32) float64 {

	f1, _ := decimal.NewFromString(s)
	f2, _ := decimal.NewFromString(p)

	var precision int32
	if len(precisions) > 0 {
		precision = precisions[0]
	} else {
		precision = 3
	}

	res, _ := f1.Mul(f2).Truncate(precision).Float64()

	return res
}

//FloatDiv 浮點數除法 (precision=3)
func FloatDiv(s string, p string, precisions ...int32) float64 {

	f1, _ := decimal.NewFromString(s)
	f2, _ := decimal.NewFromString(p)

	var precision int32
	if len(precisions) > 0 {
		precision = precisions[0]
	} else {
		precision = 3
	}
	res, _ := f1.Div(f2).Truncate(precision).Float64()

	return res
}

func GetDecimal(amount string, precisions ...int32) float64 {
	f1, _ := decimal.NewFromString(amount)
	var precision int32
	if len(precisions) > 0 {
		precision = precisions[0]
	} else {
		precision = 3
	}
	res, _ := f1.Truncate(precision).Float64()
	return res
}

func GetDecimal_Float(amount float64, precisions ...int32) float64 {
	f1 := decimal.NewFromFloat(amount)
	var precision int32
	if len(precisions) > 0 {
		precision = precisions[0]
	} else {
		precision = 3
	}
	res, _ := f1.Truncate(precision).Float64()
	return res
}

//FloatMulF 浮點數乘法 (precision=4)
func FloatMulF(s float64, p float64, precisions ...int32) float64 {

	f1 := decimal.NewFromFloat(s)
	f2 := decimal.NewFromFloat(p)

	var precision int32
	if len(precisions) > 0 {
		precision = precisions[0]
	} else {
		precision = 3
	}
	res, _ := f1.Mul(f2).Truncate(precision).Float64()
	return res
}

//FloatDivF 浮點數除法 (precision=4)
func FloatDivF(s float64, p float64, precisions ...int32) float64 {

	f1 := decimal.NewFromFloat(s)
	f2 := decimal.NewFromFloat(p)

	var precision int32
	if len(precisions) > 0 {
		precision = precisions[0]
	} else {
		precision = 3
	}
	res, _ := f1.Div(f2).Truncate(precision).Float64()
	return res
}

//取得時間戳
func GetCurrentMilliSec() int64 {
	unixNano := time.Now().UnixNano()
	return unixNano / 1000000
	//Number of millisecond elapsed since Unix epoch
}

func GetDateTimeSring(timePattern string) string {
	if strings.EqualFold(YYYYMMddHHmmss, timePattern) {
		return time.Now().Format("200601021504")
	} else if strings.EqualFold(YYYYMMddHHmmss2, timePattern) {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return ""
}

// ParseIntTime int時間隔式處理
func ParseIntTime(t int64) string {
	return time.Unix(t, 0).UTC().Format("2006-01-02 15:04:05")
}

// ParseTime 時間隔式處理
func ParseTime(t string) string {
	timeString, err := time.Parse(time.RFC3339, t)
	if err != nil {
	}
	str := strings.Split(timeString.String(), " +")
	res := str[0]
	return res
}

func CreateTransactionLog(db *gorm.DB, data *typesX.TransactionLogData) (err error) {

	jsonContent, err := json.Marshal(data.Content)
	if err != nil {
		logx.Errorf("產生交易日志錯誤:%s", err.Error())
	}

	txLog := typesX.TxLog{
		MerchantCode:    data.MerchantNo,
		MerchantOrderNo: data.MerchantOrderNo,
		OrderNo:         data.OrderNo,
		LogType:         data.LogType,
		LogSource:       data.LogSource,
		TraceId:         data.TraceId,
		Content:         string(jsonContent),
		CreatedAt:       time.Now().UTC().String(),
	}

	if err = db.Table("tx_log").Create(&txLog).Error; err != nil {
		return
	}

	return nil
}
