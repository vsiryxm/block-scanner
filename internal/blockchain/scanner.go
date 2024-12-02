package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"block-scanner/configs"
	"block-scanner/internal/pkg/utils"
	"block-scanner/internal/queue"

	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// Scanner 结构体定义了区块扫描器
type Scanner struct {
	client            *ethclient.Client
	log               *zap.Logger
	contracts         map[string]bool
	startBlock        uint64
	concurrentWorkers int
	markFile          string
}

type Transaction struct {
	TxHash string    `json:"tx_hash"`
	TxTime time.Time `json:"tx_time"`
}

type Message struct {
	ChainId uint64 `json:"chain_id"`
	TxHash  string `json:"tx_hash"`
}

// NewScanner 创建一个新的扫描器实例
func NewScanner(cfg *configs.Config, log *zap.Logger) (*Scanner, error) {
	client, err := ethclient.Dial(cfg.Chain.Provider)
	if err != nil {
		return nil, err
	}

	contracts := make(map[string]bool)
	for _, addr := range cfg.Chain.Contracts {
		contracts[strings.ToLower(addr)] = true
	}

	return &Scanner{
		client:            client,
		log:               log,
		contracts:         contracts,
		startBlock:        cfg.Scanner.StartBlock,
		concurrentWorkers: cfg.Scanner.ConcurrentWorkers,
		markFile:          cfg.Scanner.MarkFile,
	}, nil
}

// Start 开始扫描区块
func (s *Scanner) Start() error {
	for {
		// 获取最后扫描的区块
		lastScannedBlock, err := s.GetLastScannedBlockInfo()
		if err != nil {
			s.log.Fatal("error getting last scanned blocks",
				zap.Error(err))
		}

		// 处理未完成的区块
		if lastScannedBlock.BlockNumber > 0 && lastScannedBlock.ScanStatus == 0 {
			err := s.processBlock(lastScannedBlock.BlockNumber)
			if err != nil {
				s.log.Fatal("error processing incomplete block",
					zap.Uint64("blockNumber", lastScannedBlock.BlockNumber),
					zap.Error(err))
				continue
			}
			lastScannedBlock.ScanStatus = 1
			lastScannedBlock.ScannedAt = time.Now().Format("2006-01-02 15:04:05")
			s.SaveLatestScannedBlockInfo(&lastScannedBlock)
		}

		// 更新起始区块
		if lastScannedBlock.BlockNumber > 0 && lastScannedBlock.BlockNumber > s.startBlock {
			s.startBlock = lastScannedBlock.BlockNumber + 1
		}

		// 获取最新区块
		latestBlock, err := s.client.BlockNumber(context.Background())
		if err != nil {
			s.log.Fatal("error getting latest block",
				zap.Error(err))
			time.Sleep(time.Second * 10)
			continue
		}

		if s.startBlock == 0 {
			s.startBlock = 1
		}

		// 扫描新区块
		for blockNumber := s.startBlock; blockNumber <= latestBlock; blockNumber++ {
			err := s.processBlock(blockNumber)
			if err != nil {
				s.log.Fatal("error processing block",
					zap.Uint64("blockNumber", blockNumber),
					zap.Error(err))
				time.Sleep(time.Second * 5)
				continue
			}
			s.startBlock = blockNumber + 1
		}

		time.Sleep(time.Second * 10)
	}
}

// processBlock 处理单个区块
func (s *Scanner) processBlock(blockNumber uint64) error {

	// 创建区块记录
	lastScannedBlockInfo := &LastScannedBlockInfo{
		BlockNumber: blockNumber,
		ScannedAt:   time.Now().Format("2006-01-02 15:04:05"),
		ScanStatus:  0,
	}

	// 使用低级 RPC 调用获取区块数据
	var result map[string]interface{}
	err := s.client.Client().Call(&result, "eth_getBlockByNumber", fmt.Sprintf("0x%x", blockNumber), true)
	if err != nil {
		return fmt.Errorf("error fetching block %d: %v", blockNumber, err)
	}

	// 解析区块数据
	blockTime := utils.HexToUint64(result["timestamp"].(string))

	// 处理交易
	transactions := make([]*Transaction, 0)
	txs := result["transactions"].([]interface{})

	var wg sync.WaitGroup
	var mutex sync.Mutex
	semaphore := make(chan struct{}, s.concurrentWorkers)

	for _, txData := range txs {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(txData interface{}) {
			defer wg.Done()
			defer func() { <-semaphore }()

			tx := txData.(map[string]interface{})
			if s.isRelevantTransaction(tx) {
				transaction, err := s.convertToModelTransaction(tx, blockTime)
				if err != nil {
					s.log.Fatal("error converting transaction",
						zap.Error(err))
					return
				}

				mutex.Lock()
				transactions = append(transactions, transaction)
				mutex.Unlock()
			}
		}(txData)
	}

	wg.Wait()

	// 按时间戳升序排列交易
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].TxTime.Before(transactions[j].TxTime)
	})

	// 发布交易到 RabbitMQ
	messages := make([][]byte, len(transactions))
	for k, v := range transactions {
		message, _ := json.Marshal(Message{
			ChainId: configs.GetConfig().Chain.ChainID,
			TxHash:  v.TxHash,
		})
		messages[k] = message
	}
	err = queue.PublishTransactions(messages)
	if err != nil {
		return err
	}

	// 更新区块扫描状态
	lastScannedBlockInfo.ScanStatus = 1
	err = s.SaveLatestScannedBlockInfo(lastScannedBlockInfo)
	if err != nil {
		return err
	}

	s.log.Info(fmt.Sprintf("Processed block %d with %d relevant transactions\n", blockNumber, len(transactions)))
	return nil
}

// isRelevantTransaction 检查交易是否与指定的合约地址相关
func (s *Scanner) isRelevantTransaction(tx map[string]interface{}) bool {
	from := tx["from"].(string)
	to, toExists := tx["to"].(string)
	// fmt.Printf("from: %s, to: %s \n", from, to)
	if !toExists {
		// 这可能是一个合约创建交易
		return s.contracts[strings.ToLower(from)]
	}

	return s.contracts[strings.ToLower(to)] || s.contracts[strings.ToLower(from)]
}

// convertToModelTransaction 解析交易数据
func (s *Scanner) convertToModelTransaction(tx map[string]interface{}, blockTime uint64) (*Transaction, error) {
	txHash, ok := tx["hash"].(string)
	if !ok {
		return nil, fmt.Errorf("transaction hash not found or not a string")
	}

	return &Transaction{
		TxHash: txHash,
		TxTime: time.Unix(int64(blockTime), 0),
	}, nil
}
