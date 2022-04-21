package bo

// channel app 回傳給bo 代付回調的參數
type ProxyPayCallBackBO struct {
	ProxyPayOrderNo     string  `json:"proxyPayOrderNo"`     // 平台订单号
	ChannelOrderNo      string  `json:"channelOrderNo"`      //渠道商回复单号
	ChannelResultAt     string  `json:"channelResultAt"`     //渠道商回复日期  //(YYYYMMDDhhmmss)
	ChannelResultStatus string  `json:"channelResultStatus"` //渠道商回复处理状态  //(Dior渠道商范例：状态 0待处理，1处理中，2成功，3失败) */
	ChannelResultNote   string  `json:"channelResultNote"`   //渠道商回复处理备注
	Amount              float64 `json:"amount"`              //代付金额
	ChannelCharge       float64 `json:"channelCharge"`       //渠道商成本
	UpdatedBy           string  `json:"updatedBy"`           //更新人员
}
