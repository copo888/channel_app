package constants

const (
	//交易日志类型
	ERROR_MSG                 = "1" //1:錯誤訊息
	MERCHANT_REQUEST          = "2" //2:商户请求
	ERROR_REPLIED_TO_MERCHANT = "3" //3:返回商户错误
	DATA_REQUEST_CHANNEL      = "4" //4.打给渠道资料
	RESPONSE_FROM_CHANNEL     = "5" //5.渠道返回资料
	CALLBACK_FROM_CHANNEL     = "6" //6.渠道回调资料
	CALLBACK_TO_MERCHANT      = "7" //7.回调给商户

	//日誌來源(1:內充平台、2:支付API、3:代付API、4:代付平台、5:下發API)
	PLATEFORM_NC = "1"
	API_ZF       = "2"
	API_DF       = "3"
	PLATEFORM_DF = "4"
	API_XF       = "5"
	PLATEFORM_XF = "6"
)
