package types

type PayCallBackRequestX struct {
	MyIp           string `form:"myIp, optional"`
	PayOrderId     string `form:"payOrderId, optional"`
	MchId          string `form:"mchId, optional"`
	AppId          string `form:"appId, optional"`
	ProductId      string `form:"productId, optional"`
	MchOrderNo     string `form:"mchOrderNo, optional"`
	Income         string `form:"income, optional"`
	Amount         string `form:"amount, optional"`
	Status         string `form:"status, optional"`
	ChannelOrderNo string `form:"channelOrderNo, optional"`
	ChannelAttach  string `form:"channelAttach, optional"`
	Param1         string `form:"param1, optional"`
	Param2         string `form:"param2, optional"`
	PaySuccTime    string `form:"paySuccTime, optional"`
	BackType       string `form:"backType, optional"`
	Sign           string `form:"sign, optional"`
}

type ProxyPayCallBackRequestX struct {
	Ip              string `form:"ip, optional"`
	AgentpayOrderId string `form:"agentpayOrderId, optional"`
	MchOrderNo      string `form:"mchOrderNo, optional"`
	Status          string `form:"status, optional"`
	Amount          string `form:"amount, optional"`
	Fee             string `form:"fee, optional"`
	TransMsg        string `form:"transMsg, optional"`
	Sign            string `form:"sign, optional"`
}
