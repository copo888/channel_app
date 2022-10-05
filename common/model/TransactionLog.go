package model

import (
	"github.com/copo888/channel_app/common/typesX"
	"gorm.io/gorm"
	"time"
)

type TxLog struct {
	MyDB  *gorm.DB
	Table string
}

func NewTxLog(mydb *gorm.DB, t ...string) *Channel {
	table := "tx_log"
	if len(t) > 0 {
		table = t[0]
	}
	return &Channel{
		MyDB:  mydb,
		Table: table,
	}
}

//交易日志新增Func
func (c *TxLog) CreateTransactionLog(db *gorm.DB, data *typesX.TransactionLogData) (err error) {

	txLog := typesX.TxLog{
		MerchantCode:    data.MerchantNo,
		MerchantOrderNo: data.MerchantOrderNo,
		OrderNo:         data.OrderNo,
		LogType:         data.LogType,
		LogSource:       data.LogSource,
		Content:         data.Content,
		CreatedAt:       time.Now().UTC().String(),
	}

	if err = db.Table("tx_log").Create(&txLog).Error; err != nil {
		return
	}

	return nil
}
