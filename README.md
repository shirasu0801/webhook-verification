# Webhook Verification Service

外部サービス（GitHub等）から送信されるHTTP POSTリクエスト（Webhook）をセキュアに受信し、非同期で処理するGo製マイクロサービスです。

---

## 目次

1. [プロジェクト概要](#プロジェクト概要)
2. [要件定義](#要件定義)
3. [事前準備とインストール](#事前準備とインストール)
4. [設定](#設定)
5. [実行方法](#実行方法)
6. [フォルダ構造と各ファイルの内容](#フォルダ構造と各ファイルの内容)
7. [各ファイルのコード解説](#各ファイルのコード解説)
8. [改善ポイント](#改善ポイント)
9. [デプロイや運用のネクストアクション](#デプロイや運用のネクストアクション)

---

## プロジェクト概要

### 目的

GitHub Webhookなどの外部サービスからのHTTPリクエストを安全に受信し、署名検証を行った上で非同期処理を実行するサービスです。

### 主な機能

| 機能 | 説明 |
|------|------|
| Webhook受信 | POSTエンドポイントでJSONデータを受信 |
| 署名検証 | HMAC-SHA256による署名検証（GitHub形式対応） |
| IP制限 | 許可されたIPアドレスからのアクセスのみ許可 |
| 非同期処理 | Goroutine + ワーカープールによるバックグラウンド処理 |
| 構造化ログ | zapによるJSON形式のログ出力 |
| グレースフルシャットダウン | SIGINT/SIGTERM対応の安全なサーバー停止 |

### 技術スタック

- **言語**: Go 1.21+
- **Webフレームワーク**: Gin
- **ログ管理**: zap (uber-go/zap)
- **設定管理**: godotenv

---

## 要件定義

### 機能要件

1. **Webhook受信機能**
   - `POST /api/v1/webhook/github` エンドポイントでJSONペイロードを受信
   - 受信後、即座に `202 Accepted` を返却

2. **セキュリティ機能**
   - HMAC-SHA256署名検証（`X-Hub-Signature-256` ヘッダー対応）
   - IPアドレス制限（CIDR形式対応）
   - タイミング攻撃対策（`hmac.Equal`使用）

3. **非同期処理機能**
   - ワーカープールによる並行処理数制御
   - グレースフルシャットダウン対応

4. **監視機能**
   - ヘルスチェックエンドポイント（`GET /health`）
   - 構造化ログ出力（JSON形式）

### 非機能要件

| 項目 | 要件 |
|------|------|
| 可用性 | グレースフルシャットダウン対応 |
| パフォーマンス | 最大10並行処理（設定変更可能） |
| セキュリティ | 署名検証必須、IP制限対応 |
| 保守性 | 構造化ログ、環境変数による設定 |

---

## 事前準備とインストール

### 前提条件

- Go 1.21以上がインストールされていること
- Git（オプション）
- Docker（Docker実行の場合）

### Goのインストール確認

```bash
go version
# 出力例: go version go1.21.0 windows/amd64
```

### プロジェクトのセットアップ

```bash
# リポジトリをクローン（または既存ディレクトリを使用）
cd /path/to/webhook-verification

# 依存関係のインストール
go mod download
```

### GitHub Webhook設定（GitHub連携の場合）

1. GitHubリポジトリの **Settings** → **Webhooks** → **Add webhook**
2. 以下を設定:
   - **Payload URL**: `http://your-server:8081/api/v1/webhook/github`
   - **Content type**: `application/json`
   - **Secret**: 安全なランダム文字列（後述の生成方法参照）
   - **Events**: 必要なイベントを選択

### シークレットキーの生成

```bash
# Linux/Mac/Git Bash
openssl rand -hex 32

# PowerShell
[System.Guid]::NewGuid().ToString().Replace("-","") + [System.Guid]::NewGuid().ToString().Replace("-","")
```

---

## 設定

### 環境変数一覧

`.env.example` をコピーして `.env` を作成し、必要な値を設定します。

```bash
cp .env.example .env
```

| 変数名 | 説明 | デフォルト | 必須 |
|--------|------|-----------|------|
| `SERVER_PORT` | サーバーポート | `8080` | - |
| `SERVER_READ_TIMEOUT` | 読み取りタイムアウト（秒） | `10` | - |
| `SERVER_WRITE_TIMEOUT` | 書き込みタイムアウト（秒） | `10` | - |
| `WEBHOOK_SECRET` | 署名検証用シークレット | - | **必須** |
| `ALLOWED_IPS` | 許可IPアドレス（カンマ区切り、CIDR可） | 空（全許可） | - |
| `LOG_LEVEL` | ログレベル（debug/info/warn/error） | `info` | - |
| `ENV` | 実行環境（development/production） | `production` | - |
| `MAX_WORKERS` | 最大並行処理数 | `10` | - |

### 設定例

```env
# サーバー設定
SERVER_PORT=8081
SERVER_READ_TIMEOUT=10
SERVER_WRITE_TIMEOUT=10

# Webhook設定（GitHubで設定した値と同じものを指定）
WEBHOOK_SECRET=your_generated_secret_key_here

# IP制限設定（カンマ区切り、CIDR形式も可）
# 例: ALLOWED_IPS=192.168.1.0/24,10.0.0.1
ALLOWED_IPS=

# ログ設定
LOG_LEVEL=info
ENV=production

# 非同期処理設定
MAX_WORKERS=10
```

### セキュリティに関する注意

- `WEBHOOK_SECRET` はGitHubで設定した値と**完全に一致**させる必要があります
- `.env` ファイルはGitにコミットしないでください（`.gitignore` に含まれています）
- シークレットが漏洩した場合、攻撃者が偽のWebhookを送信できるようになります

---

## 実行方法

### 方法1: go run で直接実行（開発時推奨）

```bash
go run main.go
```

### 方法2: ビルドして実行（本番推奨）

```bash
# ビルド
go build -o webhook-server.exe .

# 実行
./webhook-server.exe
```

### 方法3: Docker Composeで実行

```bash
docker-compose up -d
```

### 方法4: Docker単体で実行

```bash
docker build -t webhook-verification .
docker run -p 8081:8081 -e WEBHOOK_SECRET=your_secret_here webhook-verification
```

### 起動確認

起動成功時のログ:
```json
{"level":"info","timestamp":"...","message":"Starting webhook verification service","port":"8081","log_level":"info"}
{"level":"info","timestamp":"...","message":"Server started successfully","address":":8081"}
```

### 動作確認

```bash
# ヘルスチェック
curl http://localhost:8081/health

# レスポンス例
{"status":"healthy","timestamp":1234567890}
```

### 停止方法

`Ctrl + C` でグレースフルシャットダウンが実行されます。

---

## フォルダ構造と各ファイルの内容

```
webhook-verification/
├── main.go                 # アプリケーションエントリーポイント
├── go.mod                  # Goモジュール定義
├── go.sum                  # 依存関係チェックサム
├── .env                    # 環境変数設定（Git管理外）
├── .env.example            # 環境変数テンプレート
├── .gitignore              # Git除外設定
├── .dockerignore           # Docker除外設定
├── Dockerfile              # Dockerイメージ定義
├── docker-compose.yml      # Docker Compose設定
├── README.md               # このファイル
├── webhook-server.exe      # ビルド済み実行ファイル
│
├── config/
│   └── config.go           # 設定管理（環境変数読み込み）
│
├── handlers/
│   └── webhook.go          # HTTPリクエストハンドラー
│
├── security/
│   ├── signature.go        # HMAC署名検証
│   └── ipfilter.go         # IPアドレスフィルタリング
│
├── processor/
│   └── async.go            # 非同期処理（ワーカープール）
│
└── logger/
    └── logger.go           # 構造化ログ設定
```

---

## 各ファイルのコード解説

### main.go

アプリケーションのエントリーポイント。以下の処理を行います:

```go
func main() {
    // 1. 設定読み込み
    config.LoadConfig()

    // 2. ロガー初期化
    logger.InitLogger(config.AppConfig.LogLevel)

    // 3. Ginルーター設定
    router := gin.New()
    router.Use(ginLogger())      // カスタムロガー
    router.Use(gin.Recovery())   // パニックリカバリー

    // 4. ルーティング設定
    api.POST("/webhook/github", webhookHandler.HandleWebhook)
    router.GET("/health", healthCheck)

    // 5. サーバー起動（goroutine）
    go srv.ListenAndServe()

    // 6. シグナル待機（SIGINT/SIGTERM）
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // 7. グレースフルシャットダウン
    srv.Shutdown(ctx)
}
```

**ポイント**:
- `gin.Recovery()` でパニック時も500エラーを返却
- goroutineでサーバーを起動し、メインスレッドでシグナルを監視
- 30秒のタイムアウト付きグレースフルシャットダウン

---

### config/config.go

環境変数から設定を読み込む。

```go
type Config struct {
    ServerPort         string
    ServerReadTimeout  int
    ServerWriteTimeout int
    WebhookSecret      string
    AllowedIPs         []string
    LogLevel           string
    MaxWorkers         int
}
```

**ポイント**:
- `godotenv.Load()` で `.env` ファイルを読み込み
- デフォルト値を設定し、環境変数がない場合も動作
- カンマ区切りの文字列を配列に変換（`ALLOWED_IPS`用）

---

### handlers/webhook.go

Webhookリクエストを処理するハンドラー。

```go
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
    // 1. リクエストボディ読み取り
    bodyBytes, _ := io.ReadAll(c.Request.Body)

    // 2. クライアントIP取得（プロキシ対応）
    clientIP := security.GetClientIP(...)

    // 3. IP制限チェック
    h.ipFilter.IsAllowed(clientIP)

    // 4. 署名検証
    security.VerifySignature(signature, string(bodyBytes))

    // 5. 非同期処理に送信
    h.processor.Process(payload)

    // 6. 即座に202 Acceptedを返却
    c.JSON(http.StatusAccepted, gin.H{...})
}
```

**ポイント**:
- ボディを先に読み取り、署名検証とGin両方で使用
- IP制限 → 署名検証 → 非同期処理の順で実行
- 処理完了を待たずに202を返却（非同期パターン）

---

### security/signature.go

HMAC-SHA256署名検証。

```go
func VerifySignature(signature, payload string) error {
    // 1. シークレット未設定なら検証スキップ（警告ログ）
    if config.AppConfig.WebhookSecret == "" {
        return nil
    }

    // 2. GitHub形式の署名をパース（"sha256=xxxxx"）
    signatureHash = signature[7:]

    // 3. HMAC-SHA256で期待値を計算
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(payload))
    expectedSignature := hex.EncodeToString(mac.Sum(nil))

    // 4. タイミング攻撃対策でhmac.Equalを使用
    if !hmac.Equal([]byte(signatureHash), []byte(expectedSignature)) {
        return fmt.Errorf("signature verification failed")
    }
    return nil
}
```

**ポイント**:
- `hmac.Equal` でタイミング攻撃を防止（単純な`==`比較は危険）
- GitHub形式（`sha256=...`）とプレーン形式の両方に対応

---

### security/ipfilter.go

IPアドレスによるアクセス制限。

```go
func (f *IPFilter) IsAllowed(ip string) (bool, error) {
    // 設定がない場合は全て許可
    if len(f.allowedIPs) == 0 {
        return true, nil
    }

    // 直接IPマッチング
    for _, allowedIP := range f.allowedIPs {
        if allowedIP == ip { return true, nil }
    }

    // CIDR形式マッチング
    for _, cidr := range f.allowedCIDRs {
        if cidr.Contains(parsedIP) { return true, nil }
    }

    return false, fmt.Errorf("IP address %s is not allowed", ip)
}
```

**ポイント**:
- CIDR形式（`192.168.1.0/24`）に対応
- `X-Forwarded-For`、`X-Real-IP` ヘッダーも考慮

---

### processor/async.go

ワーカープールによる非同期処理。

```go
type Processor struct {
    workerPool chan struct{}  // セマフォとして使用
    wg         sync.WaitGroup // シャットダウン時の待機用
    ctx        context.Context
    cancel     context.CancelFunc
}

func (p *Processor) Process(payload WebhookPayload) {
    p.wg.Add(1)
    go func() {
        defer p.wg.Done()

        // ワーカープールに空きがあるまで待機（セマフォ）
        p.workerPool <- struct{}{}
        defer func() { <-p.workerPool }()

        p.processPayload(payload)
    }()
}
```

**ポイント**:
- チャネルをセマフォとして使用し、並行数を制限
- `sync.WaitGroup` でシャットダウン時に全処理の完了を待機
- `processPayload` に実際のビジネスロジックを実装

---

### logger/logger.go

zapによる構造化ログ設定。

```go
func InitLogger(level string) error {
    config := zap.NewProductionConfig()
    config.Level = zap.NewAtomicLevelAt(zapLevel)
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

    // 開発環境ではコンソール出力
    if os.Getenv("ENV") == "development" {
        config.Encoding = "console"
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    }

    Logger, _ = config.Build()
}
```

**ポイント**:
- 本番環境: JSON形式（ログ集約ツール向け）
- 開発環境: カラー付きコンソール出力
- ISO8601形式のタイムスタンプ

---

## 改善ポイント

### セキュリティ

| 項目 | 現状 | 改善案 |
|------|------|--------|
| シークレット管理 | 環境変数 | HashiCorp Vault、AWS Secrets Manager等 |
| レート制限 | 未実装 | `golang.org/x/time/rate` で実装 |
| リプレイ攻撃対策 | 未実装 | タイムスタンプ検証を追加 |
| HTTPS | 未対応 | TLS証明書設定またはリバースプロキシ |

### パフォーマンス

| 項目 | 現状 | 改善案 |
|------|------|--------|
| 処理キュー | インメモリ | Redis/RabbitMQ等のメッセージキュー |
| 処理の永続化 | なし | 失敗時のリトライ機構 |
| コネクションプール | なし | DB接続プール実装 |

### 監視・運用

| 項目 | 現状 | 改善案 |
|------|------|--------|
| メトリクス | なし | Prometheus + Grafana |
| トレーシング | なし | OpenTelemetry/Jaeger |
| アラート | なし | PagerDuty/Slack通知 |

### コード品質

| 項目 | 現状 | 改善案 |
|------|------|--------|
| テスト | なし | ユニットテスト、E2Eテスト追加 |
| CI/CD | なし | GitHub Actions設定 |
| ドキュメント | README | GoDoc、OpenAPI仕様書 |

---

## デプロイや運用のネクストアクション

### Phase 1: 本番稼働準備

- [ ] **HTTPS対応**: Let's Encrypt証明書またはリバースプロキシ（nginx）設定
- [ ] **ユニットテスト追加**: 署名検証、IPフィルター等のテスト
- [ ] **CI/CDパイプライン構築**: GitHub Actions でビルド・テスト自動化
- [ ] **ログ集約**: CloudWatch Logs、Datadog等への転送設定

### Phase 2: 信頼性向上

- [ ] **メッセージキュー導入**: Redis/RabbitMQによる処理の永続化
- [ ] **リトライ機構**: 失敗した処理の再実行
- [ ] **ヘルスチェック強化**: 依存サービス（DB等）の接続確認
- [ ] **レート制限実装**: DoS攻撃対策

### Phase 3: 監視・スケーリング

- [ ] **Prometheusメトリクス**: リクエスト数、レイテンシ、エラー率
- [ ] **Grafanaダッシュボード**: 可視化
- [ ] **アラート設定**: エラー率閾値超過時の通知
- [ ] **Kubernetes対応**: Helm Chart作成、水平スケーリング

### Phase 4: 機能拡張

- [ ] **複数Webhookソース対応**: Stripe、Slack、GitLab等
- [ ] **Webhook管理UI**: 登録・確認用の管理画面
- [ ] **イベント履歴**: 受信したWebhookの履歴保存・検索

---

## APIエンドポイント

### Webhook受信

```
POST /api/v1/webhook/github
```

**リクエストヘッダー:**
- `X-Hub-Signature-256`: HMAC-SHA256署名（GitHub形式）
- または `X-Signature`: カスタム署名ヘッダー

**レスポンス:**
| ステータス | 説明 |
|------------|------|
| `202 Accepted` | Webhook受信成功、処理開始 |
| `400 Bad Request` | リクエストボディの読み取りエラー |
| `401 Unauthorized` | 署名検証失敗 |
| `403 Forbidden` | IPアドレスが許可されていない |

### ヘルスチェック

```
GET /health
```

**レスポンス:**
```json
{
  "status": "healthy",
  "timestamp": 1234567890
}
```

---

## ライセンス

MIT License
