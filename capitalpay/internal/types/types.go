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
	Remark               string `json:"remark, optional"`
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
	Ip         string `form:"ip, optional"`
	MerchantNo string `form:"merchant_no, optional"`
	Timestamp  string `form:"timestamp, optional"`
	SignType   string `form:"sign_type, optional"`
	Sign       string `form:"sign, optional"`
	Params     string `form:"params, optional"`
}

type PayCallBackRequest struct {
	MyIp       string `form:"myIp, optional"`
	MerchantNo string `form:"merchant_no, optional"`
	Timestamp  string `form:"timestamp, optional"`
	SignType   string `form:"sign_type, optional"`
	Sign       string `form:"sign, optional"`
	Params     string `form:"params, optional"`
}

type Param struct {
	Product       string  `json:"product, optional"`
	MerchantRef   string  `json:"merchant_ref, optional"`
	SystemRef     string  `json:"system_ref, optional"`
	PayAmount     string  `json:"pay_amount, optional"`
	Amount        string  `json:"amount, optional"`
	Fee           string  `json:"fee, optional"`
	Status        int64   `json:"status, optional"`
	SuccessTime   int64   `json:"success_time, optional"`
	ExtendParams  string  `json:"extendParams, optional"`
	ProductRef    string  `json:"productRef, optional"`
	BlockHash     string  `json:"block_hash, optional"`
	BlockNumber   string  `json:"block_number, optional"`
	From          string  `json:"from, optional"`
	To            string  `json:"to, optional"`
	BlockchainFee float64 `json:"blockchain_fee, optional"`
	FiatCurrency  string  `json:"fiat_currency, optional"`
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

type TelegramNotifyRequest struct {
	ChatID  int    `json:"chatId, optional"`
	Message string `json:"message"`
}
