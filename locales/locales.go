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
	message.SetString(tag, "001", "无记录")
	message.SetString(tag, "002", "系统忙碌中,请稍后再试")
	message.SetString(tag, "003", "系统錯誤,请稍后再试")
	message.SetString(tag, "005", "服务回傳失败")
	message.SetString(tag, "006", "服务回傳資料錯誤")
	message.SetString(tag, "007", "此IP非法登錄，請設定白名單")
	message.SetString(tag, "008", "渠道代付下發開關未開啟")
	message.SetString(tag, "009", "JSON格式或参数类型错误")
	message.SetString(tag, "010", "在短暂间连续呼叫API，请检查程式")
	message.SetString(tag, "011", "内容类型错误，请使用 application/json")
	message.SetString(tag, "101", "无效时间戳")
	message.SetString(tag, "102", "无效验签")
	message.SetString(tag, "103", "无效货币编码")
	message.SetString(tag, "104", "无效订单编号")
	message.SetString(tag, "105", "重复订单号")
	message.SetString(tag, "106", "无效开始日期时间")
	message.SetString(tag, "107", "无效结束日期时间")
	message.SetString(tag, "108", "无效日期范围")
	message.SetString(tag, "109", "无效日期筛选类型")
	message.SetString(tag, "110", "无效商户号")
	message.SetString(tag, "111", "商户已被禁用或结清")
	message.SetString(tag, "112", "无效金额")
	message.SetString(tag, "113", "无效语言编码")
	message.SetString(tag, "114", "无效开户行号")
	message.SetString(tag, "115", "无效开户行名")
	message.SetString(tag, "116", "无效银行卡号")
	message.SetString(tag, "117", "无效开户人姓名")
	message.SetString(tag, "118", "无效接入类型")
	message.SetString(tag, "119", "无效URL格式")
	message.SetString(tag, "120", "签名出错")
	message.SetString(tag, "121", "没有可用通道")
	message.SetString(tag, "122", "指定通道没有开启或不符合指定通道规则")
	message.SetString(tag, "123", "userId不可为空")
	message.SetString(tag, "124", "汇款人不可为空")
	message.SetString(tag, "125", "商户层级不可为空")
	message.SetString(tag, "126", "userIp不可为空")
	message.SetString(tag, "127", "不支援收银台模式")
	message.SetString(tag, "128", "不支援支付网址模式")
	message.SetString(tag, "208", "银行代码错误")
	message.SetString(tag, "209", "支付类型代码错误")
	message.SetString(tag, "210", "渠道返回错误")
	message.SetString(tag, "211", "Http状态码错误")
	message.SetString(tag, "301", "系统验签错误")
	message.SetString(tag, "400", "系统错误")
	message.SetString(tag, "401", "网路异常")
	message.SetString(tag, "501", "商户订单号不存在")
	message.SetString(tag, "EX001", "Fail")
}
