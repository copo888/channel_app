package locales

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// init
func init() {
	initEn(language.Make("en"))
}

// initEn will init en support.
func initEn(tag language.Tag) {
	message.SetString(tag, "0", "Success")
	message.SetString(tag, "210", "渠道返回错误")
	message.SetString(tag, "501", "商户订单号不存在")
	message.SetString(tag, "EX001", "Fail")
}
