package blockchain

import (
	"context"
	"log"
	"math/big"
	"sort"
	"sync"
	"time"

	"block-scanner/config"
	"block-scanner/internal/queue"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Scanner 结构体定义了区块扫描器
type Scanner struct {
	client            *ethclient.Client
	contractAddresses map[common.Address]bool
	startBlock        uint64
	concurrentWorkers int
	markFile          string
}

type Transaction struct {
	TxHash string    `json:"tx_hash"`
	TxTime time.Time `json:"tx_time"`
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
		markFile:          cfg.Scanner.MarkFile,
	}, nil
}

// Start 开始扫描区块
func (s *Scanner) Start() error {
	for {
		// 获取最后扫描的区块
		lastScannedBlock, err := s.GetLastScannedBlockInfo()
		if err != nil {
			log.Printf("Error getting last scanned blocks: %v", err)
		}

		// 处理未完成的区块
		if lastScannedBlock.BlockNumber > 0 && lastScannedBlock.ScanStatus == 0 {
			err := s.processBlock(lastScannedBlock.BlockNumber)
			if err != nil {
				log.Printf("Error processing incomplete block %d: %v", lastScannedBlock.BlockNumber, err)
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
	latestBlock := &LastScannedBlockInfo{
		BlockNumber: blockNumber,
		ScannedAt:   time.Now().Format("2006-01-02 15:04:05"),
		ScanStatus:  0,
	}

	transactions := make([]*Transaction, 0)
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
				transaction := &Transaction{
					TxHash: tx.Hash().Hex(),
					TxTime: time.Unix(int64(block.Time()), 0),
				}
				if err != nil {
					log.Printf("Error converting transaction: %v", err)
					return
				}

				// 将交易哈希添加到列表中
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

	// 发布交易到 RabbitMQ
	txHashs := make([]*string, len(transactions))
	for k, v := range transactions {
		txHashs[k] = &v.TxHash
	}
	err = queue.PublishTransactions(txHashs)
	if err != nil {
		return err
	}

	// 更新区块扫描状态
	latestBlock.ScanStatus = 1
	err = s.SaveLatestScannedBlockInfo(latestBlock)
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
