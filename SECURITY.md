# セキュリティガイドライン

## 環境変数

以下の環境変数は**絶対にコミットしないでください**：

- `JWT_SECRET` - JWT署名用の秘密鍵
- `GOOGLE_CLIENT_SECRET` - Google OAuth クライアントシークレット
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` - AWS認証情報

### JWT_SECRET の生成

本番環境では、十分に長いランダムな文字列を使用してください：

```bash
# Linux/Mac
openssl rand -base64 32

# または
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

最低でも32文字以上の文字列を使用してください。

## 本番環境チェックリスト

- [ ] すべての環境変数が安全に管理されている（AWS Secrets Manager, Parameter Store等）
- [ ] JWT_SECRET が十分に複雑（32文字以上のランダム文字列）
- [ ] HTTPS のみを使用
- [ ] CORS が適切に設定されている（本番フロントエンドURLのみ許可）
- [ ] DynamoDB テーブルに適切なIAMポリシーが設定されている
- [ ] API Gateway に適切なスロットリングが設定されている
- [ ] ログに機密情報が出力されていない

## DynamoDB セキュリティ

- IAMロールを使用してアクセス（アクセスキーではなく）
- 必要最小限の権限のみ付与
- VPCエンドポイント経由でのアクセスを推奨

## 入力検証

- すべてのユーザー入力はサーバーサイドで検証
- SQLインジェクション: DynamoDBはNoSQLですが、NoSQLインジェクションに注意
- XSS: フロントエンドでの適切なエスケープ（Reactのデフォルト動作で対応）

## 依存関係

定期的に脆弱性をチェックしてください：

```bash
# Frontend
cd frontend && npm audit

# Backend
cd backend && go list -m -json all | nancy sleuth
```
