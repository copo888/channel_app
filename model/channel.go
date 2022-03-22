package model

import "gorm.io/gorm"

type Channel struct {
	MyDB  *gorm.DB
	Table string
}

func NewChannel(mydb *gorm.DB, t ...string) *Channel {
	table := "ch_channels"
	if len(t) > 0 {
		table = t[0]
	}
	return &Channel{
		MyDB:  mydb,
		Table: table,
	}
}

func (c *Channel) GetChannel(channelCode string) (ch ChannelData, err error) {
	err = c.MyDB.Table(c.Table).
		Where("code = ?", channelCode).
		Take(&ch).Error
	return
}

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
