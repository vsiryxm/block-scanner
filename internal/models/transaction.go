package models

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BlockNumber     uint64    `gorm:"index" json:"block_number"`
	TxHash          string    `gorm:"type:varchar(66);uniqueIndex" json:"tx_hash"`
	FromAddress     string    `gorm:"type:varchar(42);index" json:"from_address"`
	ToAddress       string    `gorm:"type:varchar(42);index" json:"to_address"`
	ContractAddress string    `gorm:"type:varchar(42);index" json:"contract_address"`
	TxTime          time.Time `json:"tx_time"`
	GasUsed         uint64    `json:"gas_used"`
	GasPrice        uint64    `json:"gas_price"`
	Status          uint8     `json:"status"`
	Logs            string    `gorm:"type:text" json:"logs"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (t *Transaction) Create(db *gorm.DB) error {
	return db.Create(t).Error
}

func (t *Transaction) Update(db *gorm.DB) error {
	return db.Save(t).Error
}

func GetTransactionByHash(db *gorm.DB, txHash string) (*Transaction, error) {
	var tx Transaction
	err := db.Where("tx_hash = ?", txHash).First(&tx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func CreateTransactions(db *gorm.DB, transactions []*Transaction) error {
	return db.CreateInBatches(transactions, 100).Error
}
