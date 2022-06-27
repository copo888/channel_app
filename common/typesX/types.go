package typesX

type ChannelData struct {
	ID                      int64         `json:"id, optional"`
	Code                    string        `json:"code, optional"`
	Name                    string        `json:"name, optional"`
	IsProxy                 string        `json:"isProxy, optional"`
	IsNzPre                 string        `json:"isNzPre, optional"`
	ApiUrl                  string        `json:"apiUrl, optional"`
	CurrencyCode            string        `json:"currencyCode, optional"`
	ChannelWithdrawCharge   float64       `json:"channelWithdrawCharge, optional"`
	Balance                 float64       `json:"balance, optional"`
	Status                  string        `json:"status, optional"`
	Device                  string        `json:"device,optional"`
	MerId                   string        `json:"merId, optional"`
	MerKey                  string        `json:"merKey, optional"`
	PayUrl                  string        `json:"payUrl, optional"`
	PayQueryUrl             string        `json:"payQueryUrl, optional"`
	PayQueryBalanceUrl      string        `json:"payQueryBalanceUrl, optional"`
	ProxyPayUrl             string        `json:"proxyPayUrl, optional"`
	ProxyPayQueryUrl        string        `json:"proxyPayQueryUrl, optional"`
	ProxyPayQueryBalanceUrl string        `json:"proxyPayQueryBalanceUrl, optional"`
	WhiteList               string        `json:"whiteList, optional"`
	PayTypeMapList          []PayTypeMap  `json:"payTypeMapList, optional" gorm:"-"`
	PayTypeMap              string        `json:"payTypeMap, optional"`
	ChannelPort             string        `json:"channelPort, optional"`
	WithdrawBalance         float64       `json:"withdrawBalance, optional"`
	ProxypayBalance         float64       `json:"proxypayBalance, optional"`
	BankCodeMapList         []BankCodeMap `json:"bankCodeMapList, optional" gorm:"-"`
	Banks                   []Bank        `json:"banks, optional" gorm:"many2many:ch_channel_banks;foreignKey:Code;joinForeignKey:channel_code;references:bank_no;joinReferences:bank_no"`
}

type PayTypeMap struct {
	PayType string `json:"payType"`
	TypeNo  string `json:"typeNo"`
	MapCode string `json:"mapCode"`
}

type BankCodeMap struct {
	ID          int64  `json:"id, optional"`
	ChannelCode string `json:"channelCode, optional"`
	BankNo      string `json:"bankNo"`
	MapCode     string `json:"mapCode"`
}

type BankCodeMapX struct {
	BankCodeMap
	BankName string `json:"bankName"`
}

type Bank struct {
	ID           int64         `json:"id"`
	BankNo       string        `json:"bankNo"`
	BankName     string        `json:"bankName"`
	BankNameEn   string        `json:"bankNameEn"`
	Abbr         string        `json:"abbr"`
	BranchNo     string        `json:"branchNo"`
	BranchName   string        `json:"branchName"`
	City         string        `json:"city"`
	Province     string        `json:"province"`
	CurrencyCode string        `json:"currencyCode"`
	Status       string        `json:"status"`
	ChannelDatas []ChannelData `json:"channelDatas, optional" gorm:"many2many:ch_channel_banks:foreignKey:bankNo;joinForeignKey:bank_no;references:code;joinReferences:channel_code"`
}

type ChannelPayType struct {
	ID                int64   `json:"id, optional"`
	Code              string  `json:"code, optional"`
	ChannelCode       string  `json:"channelCode, optional"`
	PayTypeCode       string  `json:"payTypeCode, optional"`
	Fee               float64 `json:"fee, optional"`
	HandlingFee       float64 `json:"handlingFee, optional"`
	MaxInternalCharge float64 `json:"maxInternalCharge, optional"`
	DailyTxLimit      float64 `json:"dailyTxLimit, optional"`
	SingleMinCharge   float64 `json:"singleMinCharge, optional"`
	SingleMaxCharge   float64 `json:"singleMaxCharge, optional"`
	FixedAmount       string  `json:"fixedAmount, optional"`
	BillDate          int64   `json:"billDate, optional"`
	Status            string  `json:"status, optional"`
	IsProxy           string  `json:"isProxy, optional"`
	Device            string  `json:"device, optional"`
	MapCode           string  `json:"mapCode, optional"`
}
