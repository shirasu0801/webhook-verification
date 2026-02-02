package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"webhook_verification/config"
	"webhook_verification/logger"
)

// VerifySignature HMAC署名を検証する
// signature: リクエストヘッダーから取得した署名（例: "sha256=xxxxx"）
// payload: リクエストボディの生データ
// secret: 共有シークレット
func VerifySignature(signature, payload string) error {
	if config.AppConfig.WebhookSecret == "" {
		logger.Logger.Warn("Webhook secret is not configured, skipping signature verification")
		return nil
	}

	if signature == "" {
		return fmt.Errorf("signature header is missing")
	}

	// GitHub形式の署名を処理（"sha256=xxxxx"）
	var signatureHash string
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signatureHash = signature[7:]
	} else {
		signatureHash = signature
	}

	// HMAC-SHA256で署名を計算
	mac := hmac.New(sha256.New, []byte(config.AppConfig.WebhookSecret))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// タイミング攻撃を防ぐためにhmac.Equalを使用
	if !hmac.Equal([]byte(signatureHash), []byte(expectedSignature)) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}
