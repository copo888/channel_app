package payutils

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func GetSign(source string) string {
	data := []byte(source)
	result := fmt.Sprintf("%x", md5.Sum(data))
	return result
}

/*
JoinStringsInASCII 按照規則，參數名ASCII碼從小到大排序後拼接
data 待拼接的數據
sep 連接符
onlyValues 是否只包含參數值，true則不包含參數名，否則參數名和參數值均有
includeEmpty 是否包含空值，true則包含空值，否則不包含，注意此參數不影響參數名的存在
exceptKeys 被排除的參數名，不參與排序及拼接
*/
func JoinStringsInASCII(data map[string]string, sep string, onlyValues, includeEmpty bool, key string, exceptKeys ...string) string {
	var list []string
	var keyList []string
	m := make(map[string]int)
	if len(exceptKeys) > 0 {
		for _, except := range exceptKeys {
			m[except] = 1
		}
	}
	for k := range data {
		if _, ok := m[k]; ok {
			continue
		}
		value := data[k]
		if !includeEmpty && value == "" {
			continue
		}
		if onlyValues {
			keyList = append(keyList, k)
		} else {
			list = append(list, fmt.Sprintf("%s=%s", k, value))
		}
	}
	if onlyValues {
		sort.Strings(keyList)
		keyList = append(keyList, key) //加key
		for _, v := range keyList {
			list = append(list, data[v])
		}
	} else {
		sort.Strings(list)
	}
	return strings.Join(list, sep)
}

// VerifySign 验簽
func VerifySign(reqSign string, data interface{}, screctKey string, ctx context.Context) bool {
	m := CovertToMap(data)
	source := JoinStringsInASCII(m, "&", false, false, screctKey)
	sign := GetSign(source)
	fmt.Sprintf("-------" + source)
	logx.WithContext(ctx).Info("verifySource: ", source)
	logx.WithContext(ctx).Info("verifySign: ", sign)
	logx.WithContext(ctx).Info("reqSign: ", reqSign)

	if reqSign == sign {
		return true
	}

	return false
}

// SortAndSignFromUrlValues map 排序後加簽
func SortAndSignFromUrlValues(values url.Values, screctKey string, ctx context.Context) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignFromMap(m, screctKey, ctx)
}

func SortAndSignFromUrlValues_SHA256(values url.Values, screctKey string) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignSHA256FromMap(m, screctKey)
}

// SortAndSignFromObj 物件 排序後加簽
func SortAndSignFromObj(data interface{}, screctKey string, ctx context.Context) string {
	m := CovertToMap(data)
	newSource := JoinStringsInASCII(m, "&", false, false, screctKey)
	newSign := GetSign(newSource)
	logx.WithContext(ctx).Info("加签参数: ", newSource)
	logx.WithContext(ctx).Info("签名字串: ", newSign)
	return newSign
}

// SortAndSignFromMap MAP 排序後加簽
func SortAndSignFromMap(newData map[string]string, screctKey string, ctx context.Context) string {
	newSource := JoinStringsInASCII(newData, "&", false, false, screctKey)
	newSign := GetSign_HMAC_SHA384(newSource, screctKey)
	newSign = strings.ToLower(newSign)
	logx.WithContext(ctx).Info("加签参数: ", newSource)
	logx.WithContext(ctx).Info("签名字串: ", newSign)
	return newSign
}

func SortAndSignSHA256FromMap(newData map[string]string, screctKey string) string {
	newSource := JoinStringsInASCII(newData, "&", false, true, screctKey)
	newSign := GetSign_SHA256(newSource)
	return newSign
}

func GetSign_SHA256(source string) string {
	data := []byte(source)
	source2 := fmt.Sprintf("%x", sha256.Sum256(data))
	logx.Infof("sha256 %s", source2)
	result := fmt.Sprintf("%x", md5.Sum([]byte(source2)))
	return strings.ToUpper(result)
}

func GetSign_HMAC_SHA384(source string, key string) string {
	key_for_sign := []byte(key)
	mac := hmac.New(sha512.New384, key_for_sign)
	mac.Write([]byte(source))
	return hex.EncodeToString((mac.Sum(nil)))
}

func CovertUrlValuesToMap(values url.Values) map[string]string {
	m := make(map[string]string)
	for k := range values {
		m[k] = values.Get(k)
	}
	return m
}

// CovertToMap 物件轉map 檢查請求參數是否有空值
func CovertToMap(req interface{}) map[string]string {
	m := make(map[string]string)

	val := reflect.ValueOf(req)
	for i := 0; i < val.Type().NumField(); i++ {
		jsonTag := val.Type().Field(i).Tag.Get("json") // [依据不同请求类型更改] from / json
		parts := strings.Split(jsonTag, ",")
		name := parts[0]
		if name != "sign" && name != "myIp" && name != "ip" { // 過濾不需加簽參數
			if val.Field(i).Type().Name() == "float64" {
				precise := GetDecimalPlaces(val.Field(i).Float())
				valTrans := strconv.FormatFloat(val.Field(i).Float(), 'f', precise, 64)
				m[name] = valTrans
			} else if val.Field(i).Type().Name() == "string" {
				m[name] = val.Field(i).String()
			} else if val.Field(i).Type().Name() == "int64" {

				m[name] = strconv.FormatInt(val.Field(i).Int(), 10)
			}
		}
	}

	return m
}

func GetDecimalPlaces(f float64) int {
	numstr := fmt.Sprint(f)
	tmp := strings.Split(numstr, ".")
	if len(tmp) <= 1 {
		return 0
	}
	return len(tmp[1])
}
