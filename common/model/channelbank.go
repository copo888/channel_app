package model

import (
	"github.com/copo888/channel_app/common/typesX"
	"gorm.io/gorm"
)

type ChannelBanks struct {
	MyDB  *gorm.DB
	Table string
}

func NewChannelBank(mydb *gorm.DB, t ...string) *ChannelBanks {
	table := "ch_channel_banks"
	if len(t) > 0 {
		table = t[0]
	}
	return &ChannelBanks{
		MyDB:  mydb,
		Table: table,
	}
}

func (cb *ChannelBanks) GetChannelBankCode(db *gorm.DB, channelCode string, bankNo string) (bankCodeMap *typesX.BankCodeMap, err error) {
	err = db.Table(cb.Table).Where("channel_code = ? AND bank_no = ? ", channelCode, bankNo).Find(&bankCodeMap).Error
	return
}
