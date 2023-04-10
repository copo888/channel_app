// Code generated by goctl. DO NOT EDIT.
package types

type PayOrderRequest struct {
	OrderNo           string `json:"orderNo"`
	PayType           string `json:"payType, optional"`
	ChannelPayType    string `json:"channelPayType, optional"`
	TransactionAmount string `json:"transactionAmount"`
	BankCode          string `json:"bankCode, optional"`
	PageUrl           string `json:"pageUrl, optional"`
	OrderName         string `json:"orderName, optional"`
	MerchantId        string `json:"merchantId, optional"`
	Currency          string `json:"currency, optional"`
	SourceIp          string `json:"sourceIp, optional"`
	UserId            string `json:"userId, optional"`
	JumpType          string `json:"jumpType, optional"`
	PlayerId          string `json:"playerId, optional"`
	Address           string `json:"address, optionial"`
	City              string `json:"city, optionial"`
	ZipCode           string `json:"zipCode, optionial"`
	Country           string `json:"country, optionial"`
	Phone             string `json:"phone, optionial"`
	Email             string `json:"email, optionial"`
	PageFailedUrl     string `json:"pageFailedUrl, optional"`
}

type PayOrderResponse struct {
	PayPageInfo    string `json:"payPageInfo, optional"`
	PayPageType    string `json:"payPageType, optional"`
	ChannelOrderNo string `json:"channelOrderNo, optional"`
	OrderAmount    string `json:"orderAmount, optional"`
	RealAmount     string `json:"realAmount, optional"`
	Status         string `json:"status, optional"`
	IsCheckOutMer  bool   `json:"isCheckOutMer, optional"`
}

type PayOrderQueryRequest struct {
	OrderNo        string `json:"orderNo"`
	ChannelOrderNo string `json:"channelOrderNo, optional"`
}

type PayOrderQueryResponse struct {
	OrderStatus      string  `json:"orderStatus"`
	OrderAmount      float64 `json:"orderAmount"`
	ChannelOrderTime string  `json:"channelOrderTime"`
	ChannelCharge    float64 `json:"channelCharge"`
	CallBackStatus   string  `json:"callBackStatus"`
	OrderUpdateTime  string  `json:"orderUpdateTime"`
	ChannelOrderNo   string  `json:"channelOrderNo"`
}

type Empty struct {
}

type OrderResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,optional"`
}

type PayQueryInternalBalanceResponse struct {
	ChannelNametring   string `json:"channelNametring"`
	ChannelCodingtring string `json:"channelCodingtring"`
	WithdrawBalance    string `json:"withdrawBalance"`
	ProxyPayBalance    string `json:"proxyPayBalance"`
	UpdateTimetring    string `json:"updateTimetring"`
	ErrorCodetring     string `json:"errorCodetring, optional"`
	ErrorMsgtring      string `json:"errorMsgtring, optional"`
}

type ProxyPayOrderRequest struct {
	OrderNo              string `json:"orderNo"`
	TransactionType      string `json:"transactionType"`
	TransactionAmount    string `json:"transactionAmount"`
	ReceiptAccountNumber string `json:"receiptAccountNumber"`
	ReceiptAccountName   string `json:"receiptAccountName"`
	ReceiptCardProvince  string `json:"receiptCardProvince"`
	ReceiptCardCity      string `json:"receiptCardCity"`
	ReceiptCardArea      string `json:"receiptCardArea"`
	ReceiptCardBranch    string `json:"receiptCardBranch"`
	ReceiptCardBankCode  string `json:"receiptCardBankCode"`
	ReceiptCardBankName  string `json:"receiptCardBankName"`
	PlayerId             string `json:"playerId, optional"`
}

type ProxyPayOrderResponse struct {
	ChannelOrderNo string `json:"channelOrderNo"`
	OrderStatus    string `json:"orderStatus"`
}

type ProxyPayOrderQueryRequest struct {
	OrderNo        string `json:"orderNo"` //渠道
	ChannelOrderNo string `json:"channelOrderNo"`
}

type ProxyPayOrderQueryResponse struct {
	Status           int     `json:"status"`
	ChannelOrderNo   string  `json:"channelOrderNo"`
	OrderStatus      string  `json:"orderStatus"`
	CallBackStatus   string  `json:"callBackStatus"`
	ChannelReplyDate string  `json:"channelReplyDate"`
	ChannelCharge    float64 `json:"channelCharge"`
}

type ProxyPayQueryInternalBalanceResponse struct {
	ChannelNametring   string `json:"channelNametring"`
	ChannelCodingtring string `json:"channelCodingtring"`
	WithdrawBalance    string `json:"withdrawBalance"`
	ProxyPayBalance    string `json:"proxyPayBalance"`
	UpdateTimetring    string `json:"updateTimetring"`
	ErrorCodetring     string `json:"errorCodetring"`
	ErrorMsgtring      string `json:"errorMsgtring"`
}

type ProxyPayCallBackRequest struct {
	Ip          string  `form:"ip, optional"`
	Status      string  `json:"status, optional"`
	TradeNo     string  `json:"tradeNo, optional"`
	OrderNo     string  `json:"orderNo, optional"`
	Amount      float64 `json:"amount, optional"`
	Name        string  `json:"name, optional"`
	BankName    string  `json:"bankName, optional"`
	BankAccount string  `json:"bankAccount, optional"`
	BankBranch  string  `json:"bankBranch, optional"`
	Memo        string  `json:"memo, optional"`
	Mobile      string  `json:"mobile, optional"`
	Fee         float64 `json:"fee, optional"`
	Extra       string  `json:"extra, optional"`
	Sign        string  `json:"sign, optional"`
}

type PayCallBackRequest struct {
	MyIp      string  `json:"myIp, optional"`
	Status    string  `json:"status, optional"`
	TradeNo   string  `json:"tradeNo, optional"`
	OrderNo   string  `json:"orderNo, optional"`
	UserNo    int64   `json:"userNo, optional"`
	UserName  string  `json:"userName, optional"`
	ChannelNo int64   `json:"channelNo, optional"`
	Amount    float64 `json:"amount, optional"`
	Discount  float64 `json:"discount, optional"`
	Lucky     float64 `json:"lucky, optional"`
	Paid      float64 `json:"paid, optional"`
	Extra     string  `json:"extra, optional"`
	Sign      string  `json:"sign, optional"`
}

type ReceiverInfoVO struct {
	CardName   string  `json:"cardName"`
	CardNumber string  `json:"cardNumber"`
	BankName   string  `json:"bankName"`
	BankBranch string  `json:"bankBranch"`
	Amount     float64 `json:"amount"`
	Link       string  `json:"link"`
	Remark     string  `json:"remark"`
}