package responsex

var (
	SUCCESS           = "0"     //"操作成功"
	FAIL              = "EX000" //"Fail"
	INVALID_PARAMETER = "EX001" //"参数不合法"
	// 系统讯息码
	NO_RECORD_DATA              = "001" // "无记录
	GENERAL_ERROR               = "002" // "系统忙碌中,请稍后再试
	GENERAL_EXCEPTION           = "003" // "系统錯誤,请稍后再试
	SERVICE_RESPONSE_ERROR      = "005" // "服务回傳失败
	SERVICE_RESPONSE_DATA_ERROR = "006" // "服务回傳資料錯誤
	IP_DENIED                   = "007" // "此IP非法登錄，請設定白名單
	FLAG_INVALID                = "008" // "渠道代付下發開關未開啟
	PARAMETER_TYPE_ERROE        = "009" // "JSON格式或参数类型错误
	WAIT_LOCK_EXCEPTION         = "010" // "在短暂间连续呼叫API，请检查程式
	ContentType_ERROR           = "011" // "内容类型错误，请使用 application/json
	DECODE_JSON_ERROR           = "012" // "解析BO返回JSON錯誤"

	// 參數錯誤訊息
	INVALID_TIMESTAMP                  = "101" //  "无效时间戳"
	INVALID_SIGN                       = "102" //  "无效验签"
	INVALID_CURRENCY_CODE              = "103" //  "无效货币编码"
	INVALID_ORDER_NO                   = "104" //  "无效订单编号"
	REPEAT_ORDER_NO                    = "105" //  "重复订单号"
	INVALID_START_DATE                 = "106" //  "无效开始日期时间"
	INVALID_END_DATE                   = "107" //  "无效结束日期时间"
	INVALID_DATE_RANGE                 = "108" //  "无效日期范围"
	INVALID_DATE_TYPE                  = "109" //  "无效日期筛选类型"
	INVALID_MERCHANT_CODE              = "110" //  "无效商户号"
	MERCHANT_IS_DISABLE                = "111" //  "商户已被禁用或结清"
	INVALID_AMOUNT                     = "112" //  "无效金额"
	INVALID_LANGUAGE_CODE              = "113" //  "无效语言编码"
	INVALID_BANK_ID                    = "114" //  "无效开户行号"
	INVALID_BANK_NAME                  = "115" //  "无效开户行名"
	INVALID_BANK_NO                    = "116" //  "无效银行卡号"
	INVALID_DEFRAY_NAME                = "117" //  "无效开户人姓名"
	INVALID_ACCESS_TYPE                = "118" //  "无效接入类型"
	INVALID_NOTIFY_URL                 = "119" //  "无效URL格式"
	SIGN_ERROR                         = "120" //  "签名出错"
	NO_AVAILABLE_CHANNEL_ERROR         = "121" //  "没有可用通道"
	CHANNEL_NOT_OPEN_OR_NOT_MEET_RULES = "122" //  "指定通道没有开启或不符合指定通道规则"
	INVALID_USER_ID                    = "123" // "userId不可为空"
	ISNULL_ORDERNAME                   = "124" // "汇款人不可为空"
	INVALID_MERCHANT_LEVEL             = "125" // "商户层级不可为空"
	INVALID_USER_IP                    = "126" // "userIp不可为空"
	PARAMS_JSON_IS_INVALID             = "127" // "不支援收银台模式"
	PARAMS_URL_IS_INVALID              = "128" // "不支援支付网址模式"
	// 渠道
	BANK_CODE_INVALID      = "208" // "银行代码错误"
	PAY_TYPE_INVALID       = "209" // "支付类型代码错误"
	CHANNEL_REPLY_ERROR    = "210" // "渠道返回错误"
	INVALID_STATUS_CODE    = "211" // "Http状态码错误"
	INTERNAL_SIGN_ERROR    = "301" // "系统验签错误"
	SYSTEM_ERROR           = "400" // "系统错误"
	NETWORK_ERROR          = "401" // "网路异常"
	ORDER_NUMBER_NOT_EXIST = "501" // "商户订单号不存在"
)
