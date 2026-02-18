# syuu-backend

瞬間英作文トレーニングアプリのバックエンドAPI

## 技術スタック

- Go (Gin)
- DynamoDB
- JWT認証
- Google OAuth 2.0

## セットアップ

```bash
# 依存関係インストール
go mod download

# 環境変数設定
cp .env.example .env
# .envを編集

# サーバー起動
go run cmd/api/main.go
```

## 環境変数

```
PORT=8080
ENV=development
AWS_REGION=ap-northeast-1
DYNAMODB_ENDPOINT=http://localhost:8000
DYNAMODB_ITEMS_TABLE=syuu-items
DYNAMODB_USERS_TABLE=syuu-users
DYNAMODB_SRS_TABLE=syuu-srs
DYNAMODB_ANSWERS_TABLE=syuu-answers
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
JWT_SECRET=your-secret-at-least-32-chars
FRONTEND_URL=http://localhost:3000
```

## API

### 認証
- `POST /auth/google/callback` - Google OAuth
- `POST /auth/refresh` - トークンリフレッシュ

### 学習
- `POST /api/practice/start` - セッション開始
- `GET /api/practice/:session_id/next` - 次の問題取得
- `POST /api/practice/:session_id/answer` - 回答送信

### 統計
- `GET /api/stats/summary` - 統計サマリー
- `GET /api/stats/weakness` - 弱点分析

## テスト

```bash
go test ./...
```

## シードデータ

```bash
go run cmd/seed/main.go
```
"# syuu-backend" 
