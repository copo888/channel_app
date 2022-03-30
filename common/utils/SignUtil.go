package utils

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"strings"
	"time"
)

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
		if timeX, err := time.Parse("200601021504", trimStr); err == nil {
			if time.Now().Sub(timeX).Minutes() <= 5 {
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
