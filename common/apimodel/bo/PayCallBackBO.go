package bo

type PayCallBackBO struct {
	CallbackTime   string  `json:"callbackTime"`
	ChannelOrderNo string  `json:"channelOrderNo"`
	OrderAmount    float64 `json:"orderAmount"`
	OrderStatus    string  `json:"orderStatus"`
	PayOrderNo     string  `json:"payOrderNo"`
}
