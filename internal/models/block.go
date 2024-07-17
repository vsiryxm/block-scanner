package models

import (
	"time"

	"gorm.io/gorm"
)

type Block struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BlockNumber uint64    `gorm:"uniqueIndex" json:"block_number"`
	BlockHash   string    `gorm:"type:varchar(66)" json:"block_hash"`
	BlockTime   time.Time `json:"block_time"`
	ScannedAt   time.Time `json:"scanned_at"`
	ScanStatus  uint8     `gorm:"default:0" json:"scan_status"`
}

func (b *Block) TableName() string {
	return "tb_block"
}

func (b *Block) Create(db *gorm.DB) error {
	return db.Create(b).Error
}

func (b *Block) Update(db *gorm.DB) error {
	return db.Save(b).Error
}

func GetLastScannedBlock(db *gorm.DB) (*Block, error) {
	var block Block
	err := db.Order("block_number desc").First(&block).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &block, nil
}

func GetIncompleteBlocks(db *gorm.DB) ([]Block, error) {
	var blocks []Block
	err := db.Where("scan_status = ?", 0).Order("block_number asc").Find(&blocks).Error
	return blocks, err
}
