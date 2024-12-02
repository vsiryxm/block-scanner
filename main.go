package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"block-scanner/configs"
	"block-scanner/internal/blockchain"
	"block-scanner/internal/queue"

	"block-scanner/internal/pkg/logger"

	"go.uber.org/zap"
)

func main() {

	// 加载配置
	err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	cfg := configs.GetConfig()

	// 初始化 logger
	accessLogger, err := logger.NewJSONLogger(
		logger.WithDisableConsole(),
		logger.WithField("domain", cfg.Server.Name),
		logger.WithTimeLayout("2006-01-02 15:04:05"),
		logger.WithFileP("./logs/"+cfg.Server.Name+"-access.log"),
	)
	if err != nil {
		panic(err)
	}

	// 初始化 RabbitMQ 连接
	err = queue.InitRabbitMQ(cfg)
	if err != nil {
		accessLogger.Fatal("error initializing RabbitMQ",
			zap.Error(err))
	}
	defer queue.CloseRabbitMQ()

	// 创建并启动扫描器
	scanner, err := blockchain.NewScanner(cfg, accessLogger)
	if err != nil {
		accessLogger.Fatal("error creating scanner",
			zap.Error(err))
	}

	// 在后台启动扫描器
	go func() {
		if err := scanner.Start(); err != nil {
			accessLogger.Fatal("error starting scanner",
				zap.Error(err))
		}
	}()

	// 设置一个简单的 HTTP 服务器，用于健康检查
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Scanner is running")
	})

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: http.DefaultServeMux,
	}

	// 在后台启动 HTTP 服务器
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			accessLogger.Fatal("error starting server",
				zap.Error(err))
		}
	}()

	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	accessLogger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		accessLogger.Fatal("server forced to shutdown", zap.Error(err))
	}

	accessLogger.Info("server exiting")
}
