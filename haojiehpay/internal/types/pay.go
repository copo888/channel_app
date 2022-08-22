package types

import "encoding/json"

type PayCallBackRequestX struct {
	MyIp           string `form:"myIp, optional"`
	PayOrderId     string `form:"payOrderId"`
	MchId          string `form:"mchId"`
	AppId          string `form:"appId"`
	ProductId      string `form:"productId"`
	MchOrderNo     string `form:"mchOrderNo"`
	Amount         string `form:"amount"`
	Status         string `form:"status"`
	ChannelOrderNo string `form:"channelOrderNo, optional"`
	ChannelAttach  string `form:"channelAttach, optional"`
	Param1         string `form:"param1, optional"`
	Param2         string `form:"param2, optional"`
	PaySuccTime    string `form:"paySuccTime"`
	BackType       string `form:"backType"`
	Sign           string `form:"sign"`
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
