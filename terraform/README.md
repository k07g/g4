# terraform

dev / sandbox 環境向けのAmazon Cognito(ユーザープール/アプリクライアント)をTerraformで構築する。

## 構成

```
terraform/
  modules/cognito/          # Cognitoユーザープール+アプリクライアントの再利用可能モジュール
  environments/dev/          # dev環境のエントリーポイント(モジュールを呼び出す)
  environments/sandbox/      # sandbox環境のエントリーポイント(モジュールを呼び出す)
```

state はまずローカルファイルで管理する構成になっている
(各環境の `versions.tf`)。チームで共有する場合は S3 backend 等への移行を検討すること。

## 前提

- Terraform >= 1.5
- AWS認証情報(環境変数 / `~/.aws/credentials` など)が設定済みであること

## 使い方

対象環境のディレクトリ(`dev` または `sandbox`)で実行する。

```sh
cd terraform/environments/dev       # または terraform/environments/sandbox
terraform init
terraform plan
terraform apply
```

`apply` 後、出力値をアプリの `.env` に設定する。

```sh
terraform output cognito_user_pool_id
terraform output cognito_client_id
terraform output -raw cognito_client_secret   # シークレットなので -raw で表示
```

| Terraform output | .env の変数 |
| --- | --- |
| `aws_region` | `AWS_REGION` |
| `cognito_user_pool_id` | `COGNITO_USER_POOL_ID` |
| `cognito_client_id` | `COGNITO_CLIENT_ID` |
| `cognito_client_secret` | `COGNITO_CLIENT_SECRET` |

## dev / sandbox 環境の設定内容

dev・sandbox とも同じ緩めの設定(`modules/cognito` の既定値)を使う。

- サインインID: メールアドレス(`username_attributes = ["email"]`)
- 自己サインアップ許可、メール確認コードによる本人確認(Cognito標準メール送信)
- MFA: 無効
- パスワードポリシー: 最小8文字、文字種別の制約なし(検証しやすいよう緩和)
- 削除保護: 無効(作り直しやすくするため)
- アプリクライアント: `generate_secret = true`、`USER_PASSWORD_AUTH` / `REFRESH_TOKEN_AUTH` フローのみ許可

リソース名は `${project_name}-${environment}` で区別されるため、dev と sandbox は
それぞれ独立したユーザープールとして共存する。

本番相当の環境を作る場合は、`environments/` 配下に `stg` / `prod` などを追加し、
[modules/cognito](modules/cognito) の変数(MFA必須化、パスワードポリシー強化、
削除保護有効化など)を環境ごとに上書きすること。

## 破棄

```sh
cd terraform/environments/dev       # または terraform/environments/sandbox
terraform destroy
```
