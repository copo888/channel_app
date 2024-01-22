package payutils

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func main() {
	//content := "1232435465765345243152524312143526375868431214352637586821435263758687856745243121435263758685634524312313254637689767856752431214352637586845643123143524312143526375868526375524312143526375868869867565345524312143526375868524312143526375868524312143526375868"

	//publicKey := "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCBgveeFzs3TKSdfMH9Z5uZ5aAThUaSaCLjabXIyAzpbmz1SzbCc6YNXAEYwDvXkztTzbks6jQbt61ib1Uuy4z123wEYk3p4IyFMKfEquPAauj7yTybME0J23rmpXDgXLsX5vO2LB1P9pcv1HiJG/403wYiebnLOfB1w/20qtRnyQIDAQAB"

	//privateKey := "MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAIGC954XOzdMpJ18wf1nm5nloBOFRpJoIuNptcjIDOlubPVLNsJzpg1cARjAO9eTO1PNuSzqNBu3rWJvVS7LjPXbfARiTengjIUwp8Sq48Bq6PvJPJswTQnbeualcOBcuxfm87YsHU/2ly/UeIkb/jTfBiJ5ucs58HXD/bSq1GfJAgMBAAECgYAL8KEXiBjDfmNmyYuw6w5jX9IkOpNJCCS/Ro2l1xupoa6V5rtDrhnO/X50Y7SgqUg876h0xZrMO2DWxGDcEZQLLKaeNNLB1XehgF+J+Tffwbo1eoMpH4DnTp8fVU5gWSei55GMN/xzp1qph+06b74wQ2hJq6ig49xVU6VYxErDAQJBANJPQ6dfbzbF/D9WmXiFRKf87gXDnVTyQ/tmVTCns5VV0M+E6rDs/E/9pmt7emXKvDbzv+3V4hQ1EJEg5da1RbkCQQCdpfrNBm7KQPGDBqreOoMmk/UoimD5hVznAsZQ3ko0Hi2eUehQBxLVMYXpKw5U5JP1wVPjf7mAsOwFacxNPDqRAkADGMGxRDl5//5P3HGUEbpKEvJaSWAWsR6JJB+bAM0nJMVXWOivxD2O2/hIWuAZgZu1327zDJQwoftld6uKts6ZAkAtEvHcgQRYS61B2zwrgetRsmgcCUSk0x625jIxmPz6Xc6JP73+c6dM0XYKLsdQOnKbh4UmvLQbOXqiKZfCVYAhAkBNsuCHrdgjN6mQNI6Xeo6+v8mjeQJ0XnPpMpbbq5E6bTpQGeNlirs6bonn09gWdv3ItYlreaLs08CeBSXby/6U"

}

func GetSign(source string) string {
	data := []byte(source)
	result := fmt.Sprintf("%x", md5.Sum(data))
	return result
}

func GetSign_RSA(data []byte, publicKey string) string {
	//hashed := sha256.Sum256(data)
	////转换为pem格式的公钥
	publicKeyPEM := "-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"

	logx.Infof(publicKeyPEM)
	// 解码 PEM 格式的公钥
	block, _ := pem.Decode([]byte(publicKeyPEM))
	// 解析公钥
	publicKey2, _ := x509.ParsePKIXPublicKey(block.Bytes)
	// 将公钥转换为 RSA 公钥类型
	rsaPublicKey, _ := publicKey2.(*rsa.PublicKey)

	// 分块加密
	blockSize := rsaPublicKey.Size() - 11 // 11是填充的大小
	encrypted := make([]byte, 0)

	for i := 0; i < len(data); i += blockSize {
		endIndex := i + blockSize
		if endIndex > len(data) {
			endIndex = len(data)
		}
		// 加密数据
		ciphertext, _ := rsa.EncryptPKCS1v15(rand.Reader, rsaPublicKey, data[i:endIndex])
		encrypted = append(encrypted, ciphertext...)
	}
	// 将加密后的数据进行 Base64 编码
	encryptedData := base64.StdEncoding.EncodeToString(encrypted)
	return encryptedData
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
	return strings.Join(list, sep) + key
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
func SortAndSignFromUrlValues(values url.Values, screctKey string, ctx context.Context) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignFromMap(m, screctKey, ctx)
}

func SortAndSignFromUrlValues_SHA256(values url.Values, screctKey string) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignSHA256FromMap(m, screctKey)
}

// SortAndSignFromObj 物件 排序后加签
func SortAndSignFromObj(data interface{}, screctKey string, ctx context.Context) string {
	m := CovertToMap(data)
	newSource := JoinStringsInASCII(m, "&", false, false, screctKey)
	newSign := GetSign(newSource)
	logx.WithContext(ctx).Info("加签参数: ", newSource)
	logx.WithContext(ctx).Info("签名字串: ", newSign)
	return newSign
}

// SortAndSignFromMap MAP 排序后加签
func SortAndSignFromMap(newData map[string]string, screctKey string, ctx context.Context) string {
	newSource := JoinStringsInASCII(newData, "&", false, false, screctKey)
	newSign := GetSign(newSource)
	logx.WithContext(ctx).Info("加签参数: ", newSource)
	logx.WithContext(ctx).Info("签名字串: ", newSign)
	return newSign
}

func SortAndSignSHA256FromMap(newData map[string]string, screctKey string) string {
	newSource := JoinStringsInASCII(newData, "&", false, true, screctKey)
	newSign := GetSign_SHA256(newSource)
	logx.Info("加签参数: ", newSource)
	logx.Info("签名字串: ", newSign)
	return newSign
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
		if name != "myIp" && name != "ip" { // 過濾不需加簽參數 name != "sign"
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
