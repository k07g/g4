# terraform

dev / sandbox 環境向けのAmazon Cognito(ユーザープール/アプリクライアント)、
アプリのDockerイメージを保存するECRリポジトリ、そしてdev環境のAPIを実際に
稼働させるVPC/ALB/RDS/ECS FargateをTerraformで構築する。

## 構成

```
terraform/
  bootstrap/                 # state用S3、GitHub Actions用OIDC IAMロール、ECRリポジトリを作る(初回のみ手動実行)
  modules/cognito/           # Cognitoユーザープール+アプリクライアントの再利用可能モジュール
  environments/dev/          # dev環境のエントリーポイント。mainマージ時にCIが自動applyする
  environments/sandbox/      # sandbox環境のエントリーポイント(ローカルから手動運用)
```

| 環境 | state | apply方法 | 実データ |
| --- | --- | --- | --- |
| dev | S3 backend(`bootstrap`で作成) | **mainブランチへのマージ時にGitHub Actionsが自動apply** | Cognito + VPC/ALB/RDS/ECS Fargate(APIが実際に稼働) |
| sandbox | ローカルファイル | ローカルから手動で `terraform apply` | Cognitoのみ |

## 前提

- Terraform >= 1.5
- AWS認証情報(環境変数 / `~/.aws/credentials` など)が設定済みであること

## 0. 初回セットアップ(bootstrap、手動・一度だけ)

dev環境をCIから自動applyするには、事前にTerraform state用のS3バケットと、
GitHub ActionsがOIDCでAssumeRoleするためのIAMロールが必要。stateのロックは
DynamoDBではなくS3ネイティブロック(`use_lockfile`、Terraform 1.10+)を使うため
別途ロック用テーブルは不要。これは`terraform/bootstrap`で構築するが、循環依存
(stateを保存する場所自体をTerraformで作る)を避けるためローカルstateのまま、
AWS管理者権限を持つ人がローカルから一度だけ実行する。

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

## 2. アプリ: mainマージで自動デプロイ(ECR push → ECS更新)

[.github/workflows/deploy-dev.yml](../.github/workflows/deploy-dev.yml) が、
`main` ブランチへのpushのうちアプリのソース(`cmd/**`, `internal/**`, `go.mod`,
`go.sum`, `Dockerfile`)に変更があった場合に、以下を自動で行う。

1. イメージをビルドしECR(bootstrapで作成)に `<commit SHA>` タグと `latest` タグでpush
2. dev環境のECSタスク定義(`g4-dev`)の最新リビジョンを取得し、イメージだけを
   新しいSHAタグに差し替えて新しいリビジョンを登録
3. ECSサービス(`g4-dev`)をその新しいリビジョンに更新し、安定するまで待機

認証はdevのTerraform applyと同様GitHub ActionsのOIDCを使うが、`environment:`
は指定せず main ブランチへのpushを直接信頼するロール(`AWS_ECR_PUSH_ROLE_ARN`。
ECR pushとECSデプロイの両方の権限を持つ)を使う。

ECSサービス・タスク定義そのもの(CPU/メモリ、ロール、ログ設定、Secrets参照
など)は `terraform-dev-apply.yml` が管理するが、`aws_ecs_service` の
`task_definition` は `lifecycle.ignore_changes` で無視しているため、
このワークフローが登録する新しいリビジョンをTerraform applyが巻き戻すことはない。

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

## dev環境のAPIインフラ

dev環境のみ、Cognitoに加えてAPIを実際に稼働させるインフラを構築する。

- VPCはこのTerraformでは作成せず、[k07g/aws-bootstrap](https://github.com/k07g/aws-bootstrap)
  で作成済みの既存VPC(`dev-vpc`、`var.vpc_id`)を利用する。ALB・ECSタスク・
  RDSは `var.public_subnet_ids` で明示的に指定した既存のパブリックサブネットに
  配置する(タグ付け規則に依存した自動検出はしない)。NAT Gatewayは使わず、
  ECSタスクにパブリックIPを付与して直接ECR/Cognitoに到達する
  - **ALB・RDSサブネットグループは異なるAZのサブネットが2つ以上必須。**
    現時点の`dev-vpc`にはパブリックサブネットが1つ(`ap-northeast-1a`)しか
    ないため、`public_subnet_ids`に2つ目のサブネットIDを追加するまでapplyは
    失敗する(`validation`ブロックで明示的にエラーになる)。2つ目のサブネットは
    aws-bootstrap側で作成する予定
- ALB: AWS提供ドメインでHTTP(80番)公開。独自ドメイン/HTTPSは未設定
- ECS Fargate: 最小構成(0.25 vCPU / 512MiB)、`desired_count = 1`
- RDS PostgreSQL(`db.t4g.micro`): `publicly_accessible = false`。
  ECSサービスのセキュリティグループからの接続のみ許可
- DBの接続文字列・Cognitoクライアントシークレットはどちらも平文で
  タスク定義に埋め込まず、Secrets Manager経由でコンテナに注入する
- マイグレーションは手動psqlではなく、アプリ起動時に自動実行される
  (`internal/db/migrate.go`。`CREATE ... IF NOT EXISTS` のみなので冪等)

apply後、APIのURLは以下で確認できる。

```sh
cd terraform/environments/dev
terraform output api_url
```

初回applyの時点ではECRにまだイメージが無く、ECSタスクは起動に失敗し続ける
(想定内)。[.github/workflows/deploy-dev.yml](../.github/workflows/deploy-dev.yml)
が一度実行されてイメージがpushされると正常化する。

本番相当の環境を作る場合は、`environments/` 配下に `stg` / `prod` などを追加し、
[modules/cognito](modules/cognito) の変数(MFA必須化、パスワードポリシー強化、
削除保護有効化など)を環境ごとに上書きすること。CIから自動applyする場合は
`bootstrap`のIAMロールの信頼ブランチ・権限範囲を環境ごとに分けることを検討する。

## 破棄

```sh
cd terraform/environments/dev       # または terraform/environments/sandbox
terraform destroy
```

state用のS3バケットやOIDC IAMロール自体を破棄する場合は
`terraform/bootstrap` で `terraform destroy` するが、他環境が同じバケットを
参照していないことを確認してから実行すること。
