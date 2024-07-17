package blockchain

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"sort"
	"sync"
	"time"

	"block-scanner/config"
	"block-scanner/internal/database"
	"block-scanner/internal/models"
	"block-scanner/internal/queue"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
)

// Scanner 结构体定义了区块扫描器
type Scanner struct {
	client            *ethclient.Client
	contractAddresses map[common.Address]bool
	startBlock        uint64
	concurrentWorkers int
	db                *gorm.DB
}

// NewScanner 创建一个新的扫描器实例
func NewScanner(cfg *config.Config) (*Scanner, error) {
	client, err := ethclient.Dial(cfg.Ethereum.RPCURL)
	if err != nil {
		return nil, err
	}

	contractAddresses := make(map[common.Address]bool)
	for _, addr := range cfg.Ethereum.ContractAddresses {
		contractAddresses[common.HexToAddress(addr)] = true
	}

	return &Scanner{
		client:            client,
		contractAddresses: contractAddresses,
		startBlock:        cfg.Scanner.StartBlock,
		concurrentWorkers: cfg.Scanner.ConcurrentWorkers,
		db:                database.GetDB(),
	}, nil
}

// Start 开始扫描区块
func (s *Scanner) Start() error {
	for {
		// 检查是否有未完成的区块
		incompleteBlocks, err := models.GetIncompleteBlocks(s.db)
		if err != nil {
			log.Printf("Error getting incomplete blocks: %v", err)
			time.Sleep(time.Second * 10)
			continue
		}

		// 处理未完成的区块
		if len(incompleteBlocks) > 0 {
			for _, block := range incompleteBlocks {
				err := s.processBlock(block.BlockNumber)
				if err != nil {
					log.Printf("Error processing incomplete block %d: %v", block.BlockNumber, err)
					continue
				}
				block.ScanStatus = 1
				block.Update(s.db)
			}
		}

		// 获取最后扫描的区块
		lastScannedBlock, err := models.GetLastScannedBlock(s.db)
		if err != nil {
			log.Printf("Error getting last scanned block: %v", err)
			time.Sleep(time.Second * 10)
			continue
		}

		// 更新起始区块
		if lastScannedBlock != nil && lastScannedBlock.BlockNumber > s.startBlock {
			s.startBlock = lastScannedBlock.BlockNumber + 1
		}

		// 获取最新区块
		latestBlock, err := s.client.BlockNumber(context.Background())
		if err != nil {
			log.Printf("Error getting latest block: %v", err)
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
				log.Printf("Error processing block %d: %v", blockNumber, err)
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
	// 获取区块信息
	block, err := s.client.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
	if err != nil {
		return err
	}

	// 创建区块记录
	blockModel := &models.Block{
		BlockNumber: blockNumber,
		BlockHash:   block.Hash().Hex(),
		BlockTime:   time.Unix(int64(block.Time()), 0),
		ScannedAt:   time.Now(),
		ScanStatus:  0,
	}
	err = blockModel.Create(s.db)
	if err != nil {
		return err
	}

	transactions := make([]*models.Transaction, 0)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// 使用缓冲通道作为信号量，限制并发数量
	semaphore := make(chan struct{}, s.concurrentWorkers)

	// 并发处理区块中的交易
	for _, tx := range block.Transactions() {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(tx *types.Transaction) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			// 检查交易是否与指定的合约地址相关
			if s.isRelevantTransaction(tx) {
				// 转换交易为模型
				transaction, err := s.convertToModelTransaction(tx, block)
				if err != nil {
					log.Printf("Error converting transaction: %v", err)
					return
				}

				// 将交易添加到列表中
				mutex.Lock()
				transactions = append(transactions, transaction)
				mutex.Unlock()
			}
		}(tx)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 按时间戳升序排列交易
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].TxTime.Before(transactions[j].TxTime)
	})

	// 批量保存交易到数据库
	err = models.CreateTransactions(s.db, transactions)
	if err != nil {
		return err
	}

	// 发布交易到 RabbitMQ
	txHashs := make([]*string, len(transactions))
	for k, v := range transactions {
		txHashs[k] = &v.TxHash
	}
	// err = queue.PublishTransactions(transactions)
	err = queue.PublishTransactions(txHashs)
	if err != nil {
		return err
	}

	// 更新区块扫描状态
	blockModel.ScanStatus = 1
	err = blockModel.Update(s.db)
	if err != nil {
		return err
	}

	log.Printf("Processed block %d with %d relevant transactions", blockNumber, len(transactions))
	return nil
}

// isRelevantTransaction 检查交易是否与指定的合约地址相关
func (s *Scanner) isRelevantTransaction(tx *types.Transaction) bool {
	if tx.To() == nil {
		return false
	}
	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)

	if err != nil {
		log.Printf("Error getting transaction sender: %v", err)
		return false
	}

	return s.contractAddresses[*tx.To()] || s.contractAddresses[from]
}

// convertToModelTransaction 将以太坊交易转换为模型交易
func (s *Scanner) convertToModelTransaction(tx *types.Transaction, block *types.Block) (*models.Transaction, error) {
	receipt, err := s.client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, err
	}

	logs, err := json.Marshal(receipt.Logs)
	if err != nil {
		return nil, err
	}

	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return nil, err
	}

	var contractAddress string
	if s.contractAddresses[*tx.To()] {
		contractAddress = tx.To().Hex()
	} else if s.contractAddresses[from] {
		contractAddress = from.Hex()
	}

	return &models.Transaction{
		BlockNumber:     block.NumberU64(),
		TxHash:          tx.Hash().Hex(),
		FromAddress:     from.Hex(),
		ToAddress:       tx.To().Hex(),
		ContractAddress: contractAddress,
		TxTime:          time.Unix(int64(block.Time()), 0),
		GasUsed:         receipt.GasUsed,
		GasPrice:        tx.GasPrice().Uint64(),
		Status:          uint8(receipt.Status),
		Logs:            string(logs),
	}, nil
}
