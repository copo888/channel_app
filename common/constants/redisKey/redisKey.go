package redisKey

const (
	CACHE_ORDER_DATA                    = "cache:order:data:"                // 返回給 user 輸入畫面的功能
	CACHE_ORDER_DATA_REDIRECT           = "cache:orderRedirect:data:"        // 返回渠道导向的功能
	CACHE_PAY_ORDER_CHANNEL_BANK        = "cache:payChannelBank:data:"       // 自組收銀台暫存訊息
	CACHE_PAY_ORDER_CHANNEL_REDIRECT    = "cache:payChannelRedirect:data:"   // 渠道导向参数
	CACHE_PAY_ORDER_CHANNEL_REDIRECT_VA = "cache:payChannelRedirectVA:data:" // 渠道导向参数
)
