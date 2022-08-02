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

func (cb *ChannelBanks) GetChannelBankCode(db *gorm.DB, channelCode string, bankNo string) (bankCodeMap *typesX.BankCodeMapX, err error) {

	SelectX := "ccb.*, " +
		"bb.bank_name "
	err = db.Table("bk_banks bb").Select(SelectX).
		Joins("LEFT JOIN ch_channel_banks ccb ON bb.bank_no = ccb.bank_no").
		Where("ccb.channel_code = ? AND ccb.bank_no = ? ", channelCode, bankNo).
		Find(&bankCodeMap).Error
	return
}
