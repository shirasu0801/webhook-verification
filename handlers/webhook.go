package handlers

import (
	"bytes"
	"io"
	"net/http"
	"time"
	"webhook_verification/logger"
	"webhook_verification/processor"
	"webhook_verification/security"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WebhookHandler Webhookハンドラー構造体
type WebhookHandler struct {
	processor *processor.Processor
	ipFilter  *security.IPFilter
}

// NewWebhookHandler WebhookHandlerのインスタンスを作成
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		processor: processor.NewProcessor(),
		ipFilter:  security.NewIPFilter(),
	}
}

// HandleWebhook Webhook受信ハンドラー
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	startTime := time.Now()

	// リクエストボディを読み取り（署名検証のため）
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Logger.Error("Failed to read request body",
			zap.Error(err),
			zap.String("path", c.Request.URL.Path),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// ボディを再度設定（Ginが後で使用できるように）
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// クライアントIPを取得
	clientIP := security.GetClientIP(
		c.ClientIP(),
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)

	// IP制限チェック（将来の拡張用）
	allowed, err := h.ipFilter.IsAllowed(clientIP)
	if !allowed {
		logger.Logger.Warn("IP address not allowed",
			zap.String("ip", clientIP),
			zap.String("path", c.Request.URL.Path),
			zap.Error(err),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "IP address not allowed"})
		return
	}

	// 署名検証
	signature := c.GetHeader("X-Hub-Signature-256") // GitHub形式
	if signature == "" {
		signature = c.GetHeader("X-Signature") // カスタム形式
	}

	if err := security.VerifySignature(signature, string(bodyBytes)); err != nil {
		logger.Logger.Warn("Signature verification failed",
			zap.Error(err),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", clientIP),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// 受信ログを記録
	logger.Logger.Info("Webhook received",
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", c.GetHeader("User-Agent")),
		zap.Int("body_size", len(bodyBytes)),
		zap.Duration("latency", time.Since(startTime)),
	)

	// 非同期処理に送信
	payload := processor.WebhookPayload{
		Source:     "github", // パスから判定可能
		Path:       c.Request.URL.Path,
		Headers:    extractHeaders(c.Request),
		Body:       bodyBytes,
		ReceivedAt: time.Now(),
		ClientIP:   clientIP,
	}

	h.processor.Process(payload)

	// 即座に202 Acceptedを返す（非同期処理のため）
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Webhook received and processing",
		"timestamp": time.Now().Unix(),
	})
}

// extractHeaders リクエストヘッダーを抽出
func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// Shutdown ハンドラーのシャットダウン処理
func (h *WebhookHandler) Shutdown() {
	h.processor.Shutdown()
}
