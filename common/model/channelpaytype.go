package model

import (
	"github.com/copo888/channel_app/common/typesX"
	"gorm.io/gorm"
)

type ChannelPayType struct {
	MyDB  *gorm.DB
	Table string
}

func NewChannelPayType(mydb *gorm.DB, t ...string) *ChannelPayType {
	table := "ch_channel_pay_types"
	if len(t) > 0 {
		table = t[0]
	}
	return &ChannelPayType{
		MyDB:  mydb,
		Table: table,
	}
}

func (cpt *ChannelPayType) GetChannelPayType(db *gorm.DB, channelPayTypeCode string) (bankCodeMap *typesX.ChannelPayType, err error) {
	err = db.Table(cpt.Table).Where("code = ?", channelPayTypeCode).Find(&bankCodeMap).Error
	return

}
