package payutils

import (
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	keyAlgorithm    = "RSA"
	maxEncryptBlock = 117 // 1024-bit key, minus padding
	base64LineBreak = 64
)

func GetSign(source string) string {
	data := []byte(source)
	result := fmt.Sprintf("%x", md5.Sum(data))
	return result
}

/*
JoinStringsInASCII 按照规则，参数名ASCII码从小到大排序后拼接
data 待拼接的数据
sep 连接符
onlyValues 是否只包含参数值，true则不包含参数名，否则参数名和参数值均有
includeEmpty 是否包含空值，true则包含空值，否则不包含，注意此参数不影响参数名的存在
exceptKeys 被排除的参数名，不参与排序及拼接
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

// VerifySign 验签
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

// SortAndSignFromUrlValues map 排序后加签
func SortAndSignFromUrlValues(values url.Values, privateKey string, ctx context.Context) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignFromMap(m, privateKey, ctx)
}

func SortAndSignFromUrlValues_SHA256(values url.Values, screctKey string) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignSHA256FromMap(m, screctKey)
}

// SortAndSignFromObj 物件 排序后加签
func SortAndSignFromObj(data interface{}, screctKey string, ctx context.Context) string {
	m := CovertToMap(data)
	newSource := JoinStringsInASCII(m, "&", false, false, "")
	_, _ = encryptByPrivateKey([]byte(newSource), screctKey)

	newSign := GetSign3([]byte(newSource), []byte(screctKey))
	//if err != nil{
	//	logx.WithContext(ctx).Errorf("%s",err.Error())
	//}
	logx.WithContext(ctx).Info("加签参数: ", newSource)
	logx.WithContext(ctx).Info("签名字串: ", newSign)
	return newSign
}

// SortAndSignFromMap MAP 排序后加签
func SortAndSignFromMap(newData map[string]string, privateKey string, ctx context.Context) string {
	newSource := JoinStringsInASCII(newData, "&", false, false, "")
	newSign := GetSign3([]byte(newSource), []byte(privateKey))
	//if err != nil {
	//	logx.WithContext(ctx).Errorf("加签错误:%s", err.Error())
	//}

	logx.WithContext(ctx).Info("加签参数: ", newSource)
	logx.WithContext(ctx).Info("签名字串:", newSign)
	return newSign
}

func SortAndSignSHA256FromMap(newData map[string]string, screctKey string) string {
	newSource := JoinStringsInASCII(newData, "&", false, true, screctKey)
	newSign := GetSign_SHA256(newSource)
	logx.Info("加签参数: ", newSource)
	logx.Info("签名字串: ", newSign)
	return newSign
}

func GetSign3(dataJson, privateKey []byte) string {
	hashed := sha256.Sum256(dataJson)
	block, _ := pem.Decode(privateKey)
	privateKey2, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
	sign, _ := rsa.SignPKCS1v15(rand.Reader, privateKey2, crypto.SHA256, hashed[:])
	signature := base64.StdEncoding.EncodeToString(sign)
	return signature
}

func encryptByPrivateKey(data []byte, privateKeyStr string) ([]byte, error) {
	// 解码 Base64 格式的私钥
	block, _ := pem.Decode([]byte(privateKeyStr))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	// 解析 PEM 编码的私钥
	//privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	//if err != nil {
	//	return nil, err
	//}

	// 转换为 RSA 私钥类型
	//rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	//if !ok {
	//	return nil, errors.New("unexpected private key type")
	//}

	// 对数据进行分段加密
	inputLen := len(data)
	var encryptedData []byte

	for offSet := 0; offSet < inputLen; offSet += maxEncryptBlock {
		end := offSet + maxEncryptBlock
		if end > inputLen {
			end = inputLen
		}

		// 对每个数据块进行加密
		//encryptedBlock, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPrivateKey, data[offSet:end])
		//if err != nil {
		//	return nil, err
		//}

		// 追加到加密结果中
		//encryptedData = append(encryptedData, encryptedBlock...)
	}

	return encryptedData, nil
}

func GetSign_SHA256(source string) string {
	data := []byte(source)
	source2 := fmt.Sprintf("%x", sha256.Sum256(data))
	logx.Infof("sha256 %s", source2)
	result := fmt.Sprintf("%x", md5.Sum([]byte(source2)))
	return strings.ToUpper(result)
}

func CovertUrlValuesToMap(values url.Values) map[string]string {
	m := make(map[string]string)
	for k := range values {
		m[k] = values.Get(k)
	}
	return m
}

// CovertToMap 物件轉map 檢查請求參數是否有空值
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
