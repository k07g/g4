# terraform

dev / sandbox 環境向けのAmazon Cognito(ユーザープール/アプリクライアント)と、
アプリのDockerイメージを保存するECRリポジトリをTerraformで構築する。

## 構成

```
terraform/
  bootstrap/                 # state用S3/DynamoDB、GitHub Actions用OIDC IAMロール、ECRリポジトリを作る(初回のみ手動実行)
  modules/cognito/           # Cognitoユーザープール+アプリクライアントの再利用可能モジュール
  environments/dev/          # dev環境のエントリーポイント。mainマージ時にCIが自動applyする
  environments/sandbox/      # sandbox環境のエントリーポイント(ローカルから手動運用)
```

| 環境 | state | apply方法 |
| --- | --- | --- |
| dev | S3 backend(`bootstrap`で作成) | **mainブランチへのマージ時にGitHub Actionsが自動apply** |
| sandbox | ローカルファイル | ローカルから手動で `terraform apply` |

## 前提

- Terraform >= 1.5
- AWS認証情報(環境変数 / `~/.aws/credentials` など)が設定済みであること

## 0. 初回セットアップ(bootstrap、手動・一度だけ)

dev環境をCIから自動applyするには、事前にTerraform state用のS3バケット/DynamoDBロック
テーブルと、GitHub ActionsがOIDCでAssumeRoleするためのIAMロールが必要。
これは`terraform/bootstrap`で構築するが、循環依存(stateを保存する場所自体をTerraformで
作る)を避けるためローカルstateのまま、AWS管理者権限を持つ人がローカルから一度だけ実行する。

```sh
cd terraform/bootstrap
terraform init
terraform apply \
  -var="state_bucket_name=<グローバルに一意なバケット名>"
```

- `state_bucket_name` は必須(S3バケット名はAWS全体で一意である必要がある)
- AWSアカウントに既にGitHub Actions用のOIDCプロバイダが存在する場合は
  `-var="create_github_oidc_provider=false" -var="existing_github_oidc_provider_arn=<既存のARN>"`
  を追加する(1アカウントにつきプロバイダは1つまでしか作成できないため)

apply後、以下をGitHubリポジトリの **Settings > Secrets and variables > Actions > Variables**
に登録する(値はいずれも機密ではないため Secrets ではなく Variables でよい)。

| GitHub Actions variable | 値 |
| --- | --- |
| `AWS_DEV_TERRAFORM_ROLE_ARN` | `terraform output github_actions_role_arn` |
| `TF_STATE_BUCKET` | `terraform output state_bucket_name` |
| `TF_STATE_LOCK_TABLE` | `terraform output state_lock_table_name` |
| `AWS_ECR_PUSH_ROLE_ARN` | `terraform output github_actions_ecr_push_role_arn` |
| `ECR_REPOSITORY` | `terraform output ecr_repository_name`(既定値 `g4`) |
| `AWS_REGION` | 任意(未設定時は `ap-northeast-1`) |

`terraform/bootstrap` の `terraform.tfstate` はこのbootstrap自体の管理に必要なので、
誤って削除しないこと(このディレクトリはめったに変更しない想定)。

## 1. dev環境: mainマージで自動apply

[.github/workflows/terraform-dev-apply.yml](../.github/workflows/terraform-dev-apply.yml) が、
`main` ブランチへのpush(マージ)のうち `terraform/environments/dev/**` または
`terraform/modules/**` に変更があった場合に、GitHub ActionsのOIDCでAWSにAssumeRoleし
`terraform init && plan && apply` を自動実行する。

- 手動での再実行は Actions タブから `workflow_dispatch` で可能
- apply前に人手のレビューを挟みたい場合は、リポジトリの
  **Settings > Environments > dev** で Required reviewers を設定すると、
  ワークフロー変更なしに承認ゲートを追加できる
- 同時実行はconcurrency groupで直列化され、state競合を防いでいる

ローカルから同じdev stateを操作したい場合は、`backend.hcl.example` を参考に
`backend.hcl` を作成してから初期化する(`backend.hcl` は秘密情報ではないが
バケット名等が環境ごとに異なるため `.gitignore` 対象)。

```sh
cd terraform/environments/dev
cp backend.hcl.example backend.hcl   # 値をbootstrap出力に合わせて編集
terraform init -backend-config=backend.hcl
terraform plan
```

## 2. Dockerイメージ: mainマージで自動push

[.github/workflows/docker-publish.yml](../.github/workflows/docker-publish.yml) が、
`main` ブランチへのpushのうちアプリのソース(`cmd/**`, `internal/**`, `go.mod`,
`go.sum`, `Dockerfile`)に変更があった場合に、イメージをビルドしECR(bootstrapで
作成)に `<commit SHA>` タグと `latest` タグでpushする。認証はdevと同様
GitHub ActionsのOIDCを使うが、`environment:` は指定せず main ブランチへの
pushを直接信頼するロール(`AWS_ECR_PUSH_ROLE_ARN`)を使う。

手動での再実行は Actions タブから `workflow_dispatch` で可能。

## 3. sandbox環境: ローカルから手動apply

sandboxはローカルstateのまま、これまで通り手動で操作する。

```sh
cd terraform/environments/sandbox
terraform init
terraform plan
terraform apply
```

## apply後の値をアプリの.envに設定する

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
削除保護有効化など)を環境ごとに上書きすること。CIから自動applyする場合は
`bootstrap`のIAMロールの信頼ブランチ・権限範囲を環境ごとに分けることを検討する。

## 破棄

```sh
cd terraform/environments/dev       # または terraform/environments/sandbox
terraform destroy
```

state用のS3バケット/DynamoDBテーブルやOIDC IAMロール自体を破棄する場合は
`terraform/bootstrap` で `terraform destroy` するが、他環境が同じバケットを
参照していないことを確認してから実行すること。
