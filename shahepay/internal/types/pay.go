package types

type ProxyPayCallBackRequestX struct {
	Ip        string `form:"ip, optional"`
	signature string `form:"status, optional"`
	Data      string `form:"data, optional"`
}

type ProxyPayCallBackDataX struct {
	Order      PayCallBackOrderX
	NotifyType string `json:"notify_type"`
}

type ProxyPayCallBackOrderX struct {
	ID              string `json:"id"`
	TotalAmount     string `json:"total_amount"`
	MerchantOrderId string `json:"merchant_order_id"`
	Status          string `json:"status"`
}

type PayCallBackRequestX struct {
	MyIp      string `form:"myIp, optional"`
	signature string `form:"status, optional"`
	Data      string `form:"data, optional"`
}

type PayCallBackDataX struct {
	Order      PayCallBackOrderX
	NotifyType string `json:"notify_type"`
}

type PayCallBackOrderX struct {
	ID              string `json:"id"`
	TotalAmount     string `json:"total_amount"`
	MerchantOrderId string `json:"merchant_order_id"`
	Status          string `json:"status"`
}
