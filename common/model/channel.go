package model

import (
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/zeromicro/go-zero/core/logx"
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

func (c *Channel) GetChannel(channelCode string) (ch typesX.ChannelData, err error) {
	err = c.MyDB.Table(c.Table).
		Where("code = ?", channelCode).
		Take(&ch).Error
	return
}

func (c *Channel) GetChannelByProjectName(projectName string) (ch typesX.ChannelData, err error) {
	if err = c.MyDB.Table(c.Table).
		Where("project_name = ?", projectName).
		Take(&ch).Error; err != nil {
		logx.Errorf("Channel not found. ProjectName:", projectName)
		return ch, errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}
	return
}
