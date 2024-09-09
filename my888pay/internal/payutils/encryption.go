package payutils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
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

//===============================================================================================

// OriginalText 用于表示要加密的原始数据结构
type OriginalText struct {
	Account             string `json:"account"`
	Username            string `json:"username"`
	RemittanceBank      string `json:"remittance_bank"`
	RemittanceAccount   string `json:"remittance_account"`
	Amount              int    `json:"amount"`
	PlatformOrderNumber string `json:"platform_order_number"`
	ApiKey              string `json:"api_key"`
}

// 生成16字节的IV
func generateIv() ([]byte, error) {
	iv := make([]byte, aes.BlockSize) // 16字节
	_, err := io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}
	return iv, nil
}

// Base64解码密钥
func DecodeBase64Key(encodedKey string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// AES加密
func Encrypt(text string, key []byte) (string, error) {
	// 创建AES块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 生成随机的IV
	iv, err := generateIv()
	if err != nil {
		return "", err
	}

	// 填充明文数据
	plaintext := pad([]byte(text), aes.BlockSize)

	// 创建加密器
	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	// 将IV和密文组合起来
	combined := append(iv, ciphertext...)

	// 使用Base64编码返回
	return base64.StdEncoding.EncodeToString(combined), nil
}

// AES解密
func Decrypt(encryptedText string, key []byte) (string, error) {
	// Base64解码
	combined, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	// 分离IV和密文
	iv := combined[:aes.BlockSize]
	ciphertext := combined[aes.BlockSize:]

	// 创建AES块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 创建解密器
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除填充
	plaintext = unpad(plaintext)

	// 返回解密后的明文
	return string(plaintext), nil
}

// PKCS7填充
func pad(buf []byte, blockSize int) []byte {
	padding := blockSize - len(buf)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	return append(buf, padText...)
}

// PKCS7去除填充
func unpad(buf []byte) []byte {
	length := len(buf)
	padding := int(buf[length-1])
	return buf[:length-padding]
}
