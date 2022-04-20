package model

import (
	"github.com/copo888/channel_app/common/types"
	"gorm.io/gorm"
)

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

func (c *Channel) GetChannel(channelCode string) (ch types.ChannelData, err error) {
	err = c.MyDB.Table(c.Table).
		Where("code = ?", channelCode).
		Take(&ch).Error
	return
}

func (c *Channel) GetChannelByProjectName(projectName string) (ch types.ChannelData, err error) {
	err = c.MyDB.Table(c.Table).
		Where("project_name = ?", projectName).
		Take(&ch).Error
	return
}
