package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
	"time"
)

func GetSign(source string) string {
	data := []byte(source)
	result := fmt.Sprintf("%x", md5.Sum(data))
	return result
}
func main() {
	//url := "https://wallet-api.coinez.net/wallet-api/echo"
	url := "https://wallet-api.coinez.net/wallet-api/testKey"
	merchantCode := "VAHG9"
	aesKey := "qHp8VxRtzQ7HpBfE"
	md5Key := "NmW236rjx8gTeHLs"

	dataInit := struct {
		AssetType  string `json:"assetType"`
		BigDecimal string `json:"bigDecimal"`
		Now        string `json:"now"`
		TimeStamp  string `json:"timeStamp"`
	}{
		AssetType:  "1",
		BigDecimal: "1234567890.1234567890123456789",
		Now:        "2020-10-06 01:16:10",
		TimeStamp:  "1601918170152",
	}
	dataBytes, err := json.Marshal(dataInit)
	if err != nil {
		logx.Error(err.Error())
	}
	params := EnPwdCode(string(dataBytes), aesKey)
	sign := GetSign(params + md5Key)
	data := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params"`    //参数密文
		Signature    string `json:"signature"` //参数签名(params + md5key)
	}{
		MerchantCode: merchantCode,
		Params:       params,
		Signature:    sign,
	}

	res, ChnErr := gozzle.Post(url).Timeout(20).JSON(data)
	if ChnErr != nil {
		logx.Error(ChnErr.Error())
	}
	logx.Info(res)
}

func MicroServiceEncrypt(key, publicKey string) (sing string, err error) {
	str := key + time.Now().Format("200601021504")
	src := []byte(str)
	if src, err = DesCBCEncrypt(src, []byte(publicKey)); err != nil {
		return
	}
	return base64.StdEncoding.EncodeToString(src), err
}

func MicroServiceVerification(sing, key, publicKey string) (isOk bool, err error) {
	var singByte []byte
	if singByte, err = base64.StdEncoding.DecodeString(sing); err != nil {
		return
	}
	if singByte, err = DesCBCDecrypt(singByte, []byte(publicKey)); err != nil {
		return
	}

	decryptStr := string(singByte)

	if strings.Index(decryptStr, key) > -1 {
		trimStr := strings.Replace(decryptStr, key, "", 1)
		if timeX, err := time.ParseInLocation("200601021504", trimStr, time.Local); err == nil {
			if time.Now().Sub(timeX).Minutes() <= 10 {
				isOk = true
			}
		}
	}
	return
}

func DesCBCEncrypt(origData, key []byte) ([]byte, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	origData = PKCS5Padding(origData, block.BlockSize())
	// origData = ZeroPadding(origData, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, key)
	crypted := make([]byte, len(origData))
	// 根據CryptBlocks方法的說明，如下方式初始化crypted也可以
	// crypted := origData
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func DesCBCDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, key)
	//origData := make([]byte, len(crypted))
	origData := crypted
	blockMode.CryptBlocks(origData, crypted)
	//origData = PKCS5UnPadding(origData)

	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//↓↓↓↓↓===============AES CBC 加密解密===================↓↓↓↓↓

var PwdKey = "3KuavQr177wJ5Kjx"

//PKCS7 填充模式
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	//Repeat()函数的功能是把切片[]byte{byte(padding)}复制padding个，然后合并成新的字节切片返回
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//填充的反向操作，删除填充字符串
func PKCS7UnPadding1(origData []byte) ([]byte, error) {
	//获取数据长度
	length := len(origData)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	} else {
		//获取填充字符串长度
		unpadding := int(origData[length-1])
		//截取切片，删除填充字节，并且返回明文
		return origData[:(length - unpadding)], nil
	}
}

//实现加密
func AesEcrypt(origData []byte, key []byte) ([]byte, error) {
	//创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//对数据进行填充，让数据长度满足需求
	origData = PKCS7Padding(origData, blockSize)
	//采用AES加密方法中CBC加密模式
	blocMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	//执行加密
	blocMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

//实现解密
func AesDeCrypt(cypted []byte, key []byte) (string, error) {
	//创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	//获取块大小
	blockSize := block.BlockSize()
	//创建加密客户端实例
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(cypted))
	//这个函数也可以用来解密
	blockMode.CryptBlocks(origData, cypted)
	//去除填充字符串
	origData, err = PKCS7UnPadding1(origData)
	if err != nil {
		return "", err
	}
	return string(origData), err
}

//加密base64
func EnPwdCode(pwdStr string, PwdKey string) string {
	pwd := []byte(pwdStr)
	result, err := AesEcrypt(pwd, []byte(PwdKey))
	if err != nil {
		return ""
	}
	resultByte := Base64Encode(result)
	return string(resultByte)
}

//解密
func DePwdCode(pwd string, PwdKey string) string {
	//temp, _ := hex.DecodeString(pwd)
	temp, _ := base64.StdEncoding.DecodeString(pwd)
	//执行AES解密
	res, _ := AesDeCrypt(temp, []byte(PwdKey))
	return res
}

func Base64Encode(input []byte) []byte {
	eb := make([]byte, base64.StdEncoding.EncodedLen(len(input)))
	base64.StdEncoding.Encode(eb, input)

	return eb
}
