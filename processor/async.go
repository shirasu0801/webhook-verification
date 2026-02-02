package processor

import (
	"context"
	"sync"
	"time"
	"webhook_verification/config"
	"webhook_verification/logger"

	"go.uber.org/zap"
)

// WebhookPayload Webhookのペイロードデータ
type WebhookPayload struct {
	Source      string
	Path        string
	Headers     map[string]string
	Body        []byte
	ReceivedAt  time.Time
	ClientIP    string
}

// Processor 非同期処理を管理する構造体
type Processor struct {
	workerPool chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewProcessor Processorのインスタンスを作成
func NewProcessor() *Processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Processor{
		workerPool: make(chan struct{}, config.AppConfig.MaxWorkers),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Process 非同期でWebhookペイロードを処理
func (p *Processor) Process(payload WebhookPayload) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		// ワーカープールに空きがあるまで待機
		p.workerPool <- struct{}{}
		defer func() { <-p.workerPool }()

		// 実際の処理を実行
		p.processPayload(payload)
	}()
}

// processPayload 実際のWebhook処理ロジック
// ここにDB保存、外部API呼び出し、AI解析などの処理を実装
func (p *Processor) processPayload(payload WebhookPayload) {
	logger.Logger.Info("Processing webhook payload",
		zap.String("source", payload.Source),
		zap.String("path", payload.Path),
		zap.String("client_ip", payload.ClientIP),
		zap.Time("received_at", payload.ReceivedAt),
		zap.Int("body_size", len(payload.Body)),
	)

	// ここに実際の処理を実装
	// 例: DBへの保存、外部APIへの転送、AI解析など

	// シミュレーション: 重い処理を模擬
	time.Sleep(100 * time.Millisecond)

	logger.Logger.Info("Webhook payload processed successfully",
		zap.String("source", payload.Source),
		zap.String("path", payload.Path),
	)
}

// Shutdown すべての処理が完了するまで待機
func (p *Processor) Shutdown() {
	p.cancel()
	p.wg.Wait()
	logger.Logger.Info("Processor shutdown completed")
}
