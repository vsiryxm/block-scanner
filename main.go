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

	"block-scanner/config"
	"block-scanner/internal/blockchain"
	"block-scanner/internal/queue"
)

func main() {
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 初始化 RabbitMQ 连接
	err = queue.InitRabbitMQ(cfg)
	if err != nil {
		log.Fatalf("Error initializing RabbitMQ: %v", err)
	}
	defer queue.CloseRabbitMQ()

	// 创建并启动扫描器
	scanner, err := blockchain.NewScanner(cfg)
	if err != nil {
		log.Fatalf("Error creating scanner: %v", err)
	}

	// 启动扫描器
	go func() {
		if err := scanner.Start(); err != nil {
			log.Fatalf("Error starting scanner: %v", err)
		}
	}()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Scanner is running")
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
