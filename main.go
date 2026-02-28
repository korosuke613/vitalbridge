package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/korosuke613/vitalbridge/config"
	"github.com/korosuke613/vitalbridge/handlers"
	"github.com/korosuke613/vitalbridge/middleware"
	"github.com/korosuke613/vitalbridge/store"
)

const Version = "0.1.0"

func main() {
	configPath := flag.String("config", "config/config.yaml", "設定ファイルのパス")
	showVersion := flag.Bool("version", false, "バージョンを表示")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Health Ingest Service v%s\n", Version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	log.Printf("Health Ingest Service v%s を開始します...", Version)

	// メトリクスストアを初期化
	ms := store.NewMetricsStore()

	// ルーティング設定
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ingest", middleware.BearerAuth(cfg.Auth.APIKey, handlers.NewIngestHandler(ms)))
	mux.HandleFunc("/api/health", handlers.NewHealthHandler())
	mux.HandleFunc("/metrics", handlers.NewMetricsHandler(ms))
	mux.HandleFunc("/api/status", middleware.BearerAuth(cfg.Auth.APIKey, handlers.NewStatusHandler(ms)))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// TTLクリーンアップgoroutine
	ttl := time.Duration(cfg.Metrics.TTLHours) * time.Hour
	cleanupInterval := time.Duration(cfg.Metrics.CleanupIntervalMinutes) * time.Minute
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			ms.CleanExpired(ttl)
			log.Println("[Cleanup] 期限切れメトリクスのクリーンアップを実行しました")
		}
	}()

	// サーバー起動（非同期）
	go func() {
		log.Printf("[Server] %s で待ち受け開始", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Server] サーバー起動エラー: %v", err)
		}
	}()

	// シグナルハンドリング
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Health Ingest Service が開始されました。Ctrl+C で停止できます。")

	sig := <-sigChan
	log.Printf("シグナル %v を受信しました。サービスを停止します...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("サーバーのシャットダウンに失敗しました: %v", err)
	}

	log.Println("Health Ingest Service を停止しました")
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Printf("タイムゾーンの設定に失敗しました: %v", err)
	} else {
		time.Local = loc
	}
}
