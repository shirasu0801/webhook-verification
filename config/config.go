package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort        string
	ServerReadTimeout int // seconds
	ServerWriteTimeout int // seconds
	
	// Webhook設定
	WebhookSecret string // HMAC署名検証用のシークレット
	
	// IP制限設定（将来の拡張用）
	AllowedIPs []string
	
	// ログ設定
	LogLevel string // debug, info, warn, error
	
	// 非同期処理設定
	MaxWorkers int // 最大並行処理数
}

var AppConfig *Config

func LoadConfig() error {
	// .envファイルを読み込む（存在しない場合は無視）
	_ = godotenv.Load()
	
	AppConfig = &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		ServerReadTimeout: getEnvAsInt("SERVER_READ_TIMEOUT", 10),
		ServerWriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 10),
		WebhookSecret:     getEnv("WEBHOOK_SECRET", ""),
		AllowedIPs:        getEnvAsSlice("ALLOWED_IPS", []string{}),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		MaxWorkers:        getEnvAsInt("MAX_WORKERS", 10),
	}
	
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	// カンマ区切りで分割
	result := []string{}
	for _, v := range splitString(valueStr, ",") {
		if trimmed := trimSpace(v); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func splitString(s, sep string) []string {
	result := []string{}
	current := ""
	for _, char := range s {
		if string(char) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
