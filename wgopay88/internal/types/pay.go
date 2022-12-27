package types

import "encoding/json"

type PayCallBackRequestX struct {
	MyIp        string      `json:"myIp, optional"`
	PaymentId   string      `json:"payment_id, optional"`
	PaymentClId string      `json:"payment_cl_id, optional"`
	PlatformId  string      `json:"platform_id, optional"`
	Amount      json.Number `json:"amount, optional"`
	RealAmount  json.Number `json:"real_amount, optional"`
	Fee         json.Number `json:"fee, optional"`
	Status      json.Number `json:"status, optional"`
	CreateTime  json.Number `json:"create_time, optional"`
	UpdateTime  json.Number `json:"update_time, optional"`
	Sign        string      `json:"sign, optional"`
}

type ProxyPayCallBackRequestX struct {
	Ip         string      `json:"ip, optional"`
	PayoutId   string      `json:"payout_id, optional"`
	PayoutClId string      `json:"payout_cl_id, optional"`
	PlatformId string      `json:"platform_id, optional"`
	Amount     json.Number `json:"amount, optional"`
	Fee        json.Number `json:"fee, optional"`
	Status     json.Number `json:"status, optional"`
	CreateTime json.Number `json:"create_time, optional"`
	UpdateTime json.Number `json:"update_time, optional"`
	Sign       string      `json:"sign, optional"`
}
