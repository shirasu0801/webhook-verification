package security

import (
	"fmt"
	"net"
	"webhook_verification/config"
	"webhook_verification/logger"

	"go.uber.org/zap"
)

// IPFilter IPアドレス制限の枠組み
// 現在は実装の枠組みのみ。将来の拡張用。
type IPFilter struct {
	allowedIPs []string
	allowedCIDRs []*net.IPNet
}

// NewIPFilter IPFilterのインスタンスを作成
func NewIPFilter() *IPFilter {
	filter := &IPFilter{
		allowedIPs:  config.AppConfig.AllowedIPs,
		allowedCIDRs: []*net.IPNet{},
	}

	// CIDR形式のIPアドレスをパース
	for _, ipStr := range config.AppConfig.AllowedIPs {
		if _, cidr, err := net.ParseCIDR(ipStr); err == nil {
			filter.allowedCIDRs = append(filter.allowedCIDRs, cidr)
		}
	}

	return filter
}

// IsAllowed IPアドレスが許可されているかチェック
// 現在は設定がない場合は全て許可（将来の拡張用）
func (f *IPFilter) IsAllowed(ip string) (bool, error) {
	// 設定がない場合は全て許可
	if len(f.allowedIPs) == 0 {
		return true, nil
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false, fmt.Errorf("invalid IP address: %s", ip)
	}

	// 直接IPアドレスのマッチング
	for _, allowedIP := range f.allowedIPs {
		if allowedIP == ip {
			return true, nil
		}
	}

	// CIDR形式のマッチング
	for _, cidr := range f.allowedCIDRs {
		if cidr.Contains(parsedIP) {
			return true, nil
		}
	}

	logger.Logger.Warn("IP address not allowed",
		zap.String("ip", ip),
		zap.Strings("allowed_ips", f.allowedIPs),
	)

	return false, fmt.Errorf("IP address %s is not allowed", ip)
}

// GetClientIP リクエストからクライアントIPを取得
// X-Forwarded-For や X-Real-IP ヘッダーも考慮
func GetClientIP(remoteAddr string, forwardedFor string, realIP string) string {
	// X-Real-IP を優先
	if realIP != "" {
		return realIP
	}

	// X-Forwarded-For の最初のIPを取得
	if forwardedFor != "" {
		// カンマ区切りの最初のIPを取得
		for i, char := range forwardedFor {
			if char == ',' {
				return forwardedFor[:i]
			}
		}
		return forwardedFor
	}

	// remoteAddrからIPを抽出（"host:port"形式の場合）
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}

	return remoteAddr
}
