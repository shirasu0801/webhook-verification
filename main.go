package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"webhook_verification/config"
	"webhook_verification/handlers"
	"webhook_verification/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 設定を読み込み
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ロガーを初期化
	if err := logger.InitLogger(config.AppConfig.LogLevel); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Logger.Info("Starting webhook verification service",
		zap.String("port", config.AppConfig.ServerPort),
		zap.String("log_level", config.AppConfig.LogLevel),
	)

	// Ginルーターを設定
	if config.AppConfig.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// ミドルウェアを設定
	router.Use(ginLogger())
	router.Use(gin.Recovery())

	// Webhookハンドラーを作成
	webhookHandler := handlers.NewWebhookHandler()

	// ルーティング設定
	api := router.Group("/api/v1")
	{
		api.POST("/webhook/github", webhookHandler.HandleWebhook)
		// 他のWebhookエンドポイントも追加可能
		// api.POST("/webhook/custom", webhookHandler.HandleWebhook)
	}

	// ヘルスチェックエンドポイント
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// HTTPサーバーを設定
	srv := &http.Server{
		Addr:         ":" + config.AppConfig.ServerPort,
		Handler:      router,
		ReadTimeout:  time.Duration(config.AppConfig.ServerReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.AppConfig.ServerWriteTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// グレースフルシャットダウンの設定
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("Failed to start server",
				zap.Error(err),
			)
		}
	}()

	logger.Logger.Info("Server started successfully",
		zap.String("address", srv.Addr),
	)

	// シグナルを待機
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("Shutting down server...")

	// グレースフルシャットダウン
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ハンドラーのシャットダウン
	webhookHandler.Shutdown()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Error("Server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Logger.Info("Server exited")
}

// ginLogger カスタムGinロガーミドルウェア
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Logger.Info("HTTP Request",
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.Int("size", c.Writer.Size()),
		)
	}
}
