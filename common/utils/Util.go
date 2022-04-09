package utils

import (
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

//生成随机字符串
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

	if myip == "localhost" || myip == "127.0.0.1" || myip == "0:0:0:0:0:0:0:1" {
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
		ipB := net.ParseIP(myip)

		if ipnetA.Contains(ipB) {
			return true
		}
	}
	return false
}
