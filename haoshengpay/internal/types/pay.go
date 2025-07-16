package types

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
	Ip              string `form:"ip, optional"`
	TransOrderId    string `form:"transOrderId"`
	MchId           string `form:"mchId"`
	MchTransOrderNo string `form:"mchTransOrderNo"`
	Amount          string `form:"amount"`
	Status          string `form:"status"`
	ChannelOrderNo  string `form:"channelOrderNo, optional"`
	Param1          string `form:"param1, optional"`
	Param2          string `form:"param2, optional"`
	TransSuccTime   string `form:"transSuccTime"`
	BackType        string `form:"backType"`
	Sign            string `form:"sign"`
}
