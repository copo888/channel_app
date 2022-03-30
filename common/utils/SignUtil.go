package utils

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"strings"
	"time"
)

func MicroServiceEncrypt(key, publicKey string) (sing string, err error) {
	var block cipher.Block
	str := key + time.Now().Format("200601021504")
	src := []byte(str)
	if block, err = des.NewCipher([]byte(publicKey)); err != nil {
		return "", err
	}
	src = padding(src, block.BlockSize())
	blockmode := cipher.NewCBCEncrypter(block, []byte(publicKey))
	blockmode.CryptBlocks(src, src)
	sing = string(src)
	return
}

func MicroServiceVerification(sing, key, publicKey string) bool {
	decryptSing := string(decryptDES([]byte(sing), publicKey))
	isOk := false

	if strings.Index(decryptSing, key) > -1 {
		trimStr := strings.Replace(decryptSing, key, "", 1)
		if timeX, err := time.Parse("200601021504", trimStr); err == nil {
			if time.Now().Sub(timeX).Minutes() <= 5 {
				isOk = true
			}
		}
	}
	return isOk
}

func decryptDES(sing []byte, publicKey string) []byte {
	block, _ := des.NewCipher([]byte(publicKey))
	blockMode := cipher.NewCBCDecrypter(block, []byte(publicKey))
	blockMode.CryptBlocks(sing, sing)
	sing = unpadding(sing)
	return sing
}

func padding(src []byte, blocksize int) []byte {
	n := len(src)
	padnum := blocksize - n%blocksize
	pad := bytes.Repeat([]byte{byte(padnum)}, padnum)
	dst := append(src, pad...)
	return dst
}

func unpadding(src []byte) []byte {
	n := len(src)
	unpadnum := int(src[n-1])
	dst := src[:n-unpadnum]
	return dst
}
