# g4

Go + Amazon Cognito + PostgreSQL によるユーザー認証API。

## 機能

- サインアップ (`POST /auth/signup`)
- サインアップ確認 (`POST /auth/confirm`) — Cognito が要求する確認コードの検証
- サインイン (`POST /auth/signin`)
- パスワードを忘れた場合のリセットコード送信 (`POST /auth/forgot-password`)
- パスワードリセットの確定 (`POST /auth/reset-password`)
- サインアウト (`POST /auth/signout`, 要アクセストークン)
- アカウント削除 (`DELETE /auth/me`, 要アクセストークン)

認証情報(パスワード等)は Amazon Cognito が保持し、アプリ側の PostgreSQL には
Cognito の `sub` に紐づくプロフィール情報のみを保存します。

## セットアップ(本番 / Cognito 接続)

1. `.env.example` を `.env` にコピーし、Cognito ユーザープールと PostgreSQL の接続情報を設定
2. `internal/db/migrations/0001_create_users_table.up.sql` を対象データベースに適用
3. AWS 認証情報(環境変数 / `~/.aws/credentials` など)を用意
4. `go run ./cmd/server`

## ローカルでの動作検証(AWSアカウント不要)

`AUTH_PROVIDER=memory` を指定すると、Amazon Cognito の代わりにプロセス内蔵の
インメモリ認証プロバイダ([internal/auth/memory.go](internal/auth/memory.go))が使われます。
サインアップ〜削除までの一連のAPIをAWSアカウントなしでローカル確認できます。
本番では絶対に使用しないでください(データは再起動で消え、パスワードも平文保持です)。

1. ローカルPostgresを起動

   ```sh
   docker compose up -d postgres
   ```

2. マイグレーションを適用(初回のみ)

   ```sh
   docker compose exec -T postgres psql -U g4 -d g4 < internal/db/migrations/0001_create_users_table.up.sql
   ```

3. `.env.local.example` を参考に環境変数を設定してサーバーを起動

   ```sh
   PORT=8080 \
   AUTH_PROVIDER=memory \
   DATABASE_URL="postgres://g4:g4@localhost:5433/g4?sslmode=disable" \
   go run ./cmd/server
   ```

4. 別ターミナルでスモークテストを実行(サインアップ→確認→サインイン→サインアウト→削除を通しで検証)

   ```sh
   ./scripts/smoke-test.sh
   ```

   確認コードは `memory` プロバイダでは常に `000000` 固定です(実メール送信がないため)。

5. 個別に curl で叩く場合は下記の API 一覧を参照してください。

## API

### POST /auth/signup

```json
{ "email": "user@example.com", "password": "..." }
```

### POST /auth/confirm

```json
{ "email": "user@example.com", "code": "123456" }
```

### POST /auth/signin

```json
{ "email": "user@example.com", "password": "..." }
```

レスポンスの `access_token` を以降のリクエストの `Authorization: Bearer <token>` に使用します。

### POST /auth/forgot-password

```json
{ "email": "user@example.com" }
```

登録済みのメールアドレスにパスワードリセットコードを送信します。メール
アドレスが未登録の場合でも常に `204 No Content` を返します(このエンド
ポイントからアカウントの存在有無が分からないようにするため)。`memory`
プロバイダではコードは常に `000000` 固定です。

### POST /auth/reset-password

```json
{ "email": "user@example.com", "code": "123456", "new_password": "..." }
```

`POST /auth/forgot-password` で送信されたコードと新しいパスワードを指定して、
パスワードをリセットします。

### POST /auth/signout

要 `Authorization: Bearer <access_token>`。全デバイスのトークンを失効させます。

### DELETE /auth/me

要 `Authorization: Bearer <access_token>`。Cognito ユーザープールおよび
PostgreSQL 上のプロフィールを削除します。
