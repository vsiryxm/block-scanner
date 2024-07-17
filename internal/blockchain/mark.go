package blockchain

import (
	"encoding/json"
	"io"
	"os"
)

type LastScannedBlockInfo struct {
	BlockNumber uint64 `json:"block_number"` // 最近扫描区块
	ScanStatus  uint8  `json:"scan_status"`  // 是否完成 1-已完成，0-未完成
	ScannedAt   string `json:"scanned_at"`   // 扫描时间
}

func (s *Scanner) GetLastScannedBlockInfo() (LastScannedBlockInfo, error) {
	var blockInfo LastScannedBlockInfo

	file, err := os.Open(s.markFile)
	if err != nil {
		return blockInfo, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return blockInfo, err
	}

	err = json.Unmarshal(bytes, &blockInfo)
	if err != nil {
		return blockInfo, err
	}

	return blockInfo, nil
}

func (s *Scanner) SaveLatestScannedBlockInfo(blockInfo *LastScannedBlockInfo) error {

	bytes, err := json.MarshalIndent(blockInfo, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(s.markFile, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
