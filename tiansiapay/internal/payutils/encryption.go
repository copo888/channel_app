package payutils

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
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
	return strings.Join(list, sep) + "&" + key
}

// VerifySign 验签
func VerifySign(reqSign string, data interface{}, screctKey string) bool {
	m := CovertToMap(data)
	source := JoinStringsInASCII(m, "&", false, false, screctKey)
	sign := GetSign(source)
	fmt.Sprintf("-------" + source)
	logx.Info("verifySource: ", source)
	logx.Info("verifySign: ", sign)
	logx.Info("reqSign: ", reqSign)

	if reqSign == sign {
		return true
	}

	return false
}

// SortAndSignFromUrlValues map 排序后加签
func SortAndSignFromUrlValues(values url.Values, screctKey string) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignFromMap(m, screctKey)
}

func SortAndSignFromUrlValues_SHA256(values url.Values, screctKey string) string {
	m := CovertUrlValuesToMap(values)
	return SortAndSignSHA256FromMap(m, screctKey)
}

// SortAndSignFromObj 物件 排序后加签
func SortAndSignFromObj(data interface{}, screctKey string) string {
	m := CovertToMap(data)
	newSource := JoinStringsInASCII(m, "&", false, false, screctKey)
	newSign := GetSign(newSource)
	logx.Info("加签参数: ", newSource)
	logx.Info("签名字串: ", newSign)
	return newSign
}

// SortAndSignFromMap MAP 排序后加签
func SortAndSignFromMap(newData map[string]string, screctKey string) string {
	newSource := JoinStringsInASCII(newData, "&", false, false, screctKey)
	newSign := GetSign(newSource)
	logx.Info("加签参数: ", newSource)
	logx.Info("签名字串: ", newSign)
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



type liFangEncryptionInterface interface {
	LiFangEncrypt() (string, error) // 立方停車加密
	LiFangDecrypt() ([]byte, error) // 立方停車解密
}

type LiFangEncryptionStruct struct {
	Key               string // 立方密鑰
	NeedEncryptString string // 需要加密的字符串
	NeedDecryptString string // 需要解密的字符串
}

// NewLiFangEncryption 創建立方加解密對象
func NewLiFangEncryption(lfs *LiFangEncryptionStruct) liFangEncryptionInterface {
	return &LiFangEncryptionStruct{
		Key:               lfs.Key,
		NeedEncryptString: lfs.NeedEncryptString,
		NeedDecryptString: lfs.NeedDecryptString,
	}
}

func (lfs *LiFangEncryptionStruct) LiFangEncrypt() (encodeStr string, err error) {
	decodeKey, err := base64.StdEncoding.DecodeString(lfs.Key) //這一行是說，將Key密鑰進行base64編碼，這一行與加密 AES/ECB/PKCS5Padding 沒有關係
	aseByte, err := aesEncrypt([]byte(lfs.NeedEncryptString), decodeKey)//這一行開始就是 AES/ECB/PKCS5Padding 的標準加密了
	encodeStr = strings.ToUpper(hex.EncodeToString(aseByte)) //把加密後的字符串變爲大寫
	return
}

func (lfs *LiFangEncryptionStruct) LiFangDecrypt() (lastByte []byte, err error) {
	hexStrByte, err := hex.DecodeString(lfs.NeedDecryptString) //這一行的意思，把需要解密的字符串從16進制字符轉爲2進制byte數組
	decodeKey, err := base64.StdEncoding.DecodeString(lfs.Key) //這行還是將Key密鑰進行base64編碼
	lastByte, err = aesDecrypt(hexStrByte, decodeKey) // 這裏開始就是 AES/ECB/PKCS5Padding 的標準解密了
	return
}

func aesEncrypt(src, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key) // 生成加密用的block對象
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize() // 根據傳入的密鑰，返回block的大小，也就是俗稱的數據塊位數，如128位，192位，256位
	src = pKCS5Padding(src, bs)// 這裏是PKCS5Padding填充方式，繼續向下看
	if len(src)%bs != 0 { // 如果加密字符串的byte長度不能整除數據塊位數，則表示當前加密的塊大小不適用
		return nil, errorx.New("Need a multiple of the blocksize")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs]) // 開始用已經產生的key來加密
		src = src[bs:]
		dst = dst[bs:]
	}
	return out, nil
}

func aesDecrypt(src, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return nil, errors.New("crypto/cipher: input not full blocks")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = pKCS5UnPadding(out)
	return out, nil
}

func pKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}