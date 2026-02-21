# DynamoDB 設計ドキュメント

## ローカル開発環境

Docker で DynamoDB Local を起動:

```bash
docker run -d --name dynamodb-local -p 8000:8000 amazon/dynamodb-local
```

## テーブル作成

```bash
# items テーブル (問題データ)
aws dynamodb create-table --table-name syun-eng-items \
  --attribute-definitions AttributeName=PK,AttributeType=S AttributeName=SK,AttributeType=S \
  --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000

# users テーブル
aws dynamodb create-table --table-name syun-eng-users \
  --attribute-definitions AttributeName=PK,AttributeType=S AttributeName=SK,AttributeType=S \
  --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000

# srs テーブル (学習進捗)
aws dynamodb create-table --table-name syun-eng-srs \
  --attribute-definitions AttributeName=PK,AttributeType=S AttributeName=SK,AttributeType=S \
  --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000

# answers テーブル (回答履歴)
aws dynamodb create-table --table-name syun-eng-answers \
  --attribute-definitions AttributeName=PK,AttributeType=S AttributeName=SK,AttributeType=S \
  --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000
```

## シードデータ投入

```bash
DYNAMODB_ENDPOINT=http://localhost:8000 go run ./cmd/seed
```

## テーブル設計

全テーブル共通で Single Table Design (PK/SK パターン) を採用:

| テーブル | PK | SK | 用途 |
|---------|----|----|------|
| syun-eng-items | `ITEM#{item_id}` | `METADATA` | 問題データ |
| syun-eng-users | `USER#{user_id}` | `PROFILE` | ユーザー情報 |
| syun-eng-srs | `USER#{user_id}` | `SRS#{item_id}` | SRS状態 (間隔反復) |
| syun-eng-answers | `USER#{user_id}` | `ANS#{timestamp}` | 回答履歴 |

## items テーブル属性

| 属性 | 型 | 説明 |
|------|-----|------|
| PK | S | `ITEM#{item_id}` |
| SK | S | `METADATA` |
| item_id | S | 問題ID |
| japanese | S | 日本語文 |
| answers | L | 模範解答リスト |
| acceptable | L | 許容解答リスト |
| situations | L | シチュエーションタグ |
| patterns | L | パターンタグ |
| word_count | N | 単語数 |
| length_bucket | S | S/M/L/XL |
| difficulty | N | 難易度 1-5 |

## srs テーブル属性

| 属性 | 型 | 説明 |
|------|-----|------|
| PK | S | `USER#{user_id}` |
| SK | S | `SRS#{item_id}` |
| interval | N | 復習間隔（日） |
| ease_factor | N | 難易度係数 |
| due_date | S | 次回復習日 |
| repetitions | N | 復習回数 |

## 便利コマンド

```bash
# テーブル一覧
aws dynamodb list-tables --endpoint-url http://localhost:8000

# テーブル削除
aws dynamodb delete-table --table-name syun-eng-items --endpoint-url http://localhost:8000

# アイテム数確認
aws dynamodb scan --table-name syun-eng-items --select COUNT --endpoint-url http://localhost:8000
```
