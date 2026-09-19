variable "aws_region" {
  description = "AWSリージョン"
  type        = string
  default     = "ap-northeast-1"
}

variable "project_name" {
  description = "リソース名のプレフィックス"
  type        = string
  default     = "g4"
}

variable "state_bucket_name" {
  description = "Terraform state用S3バケット名。S3バケット名はAWS全体で一意である必要があるため、既定値は与えずリポジトリ/アカウント固有の値を明示的に指定する"
  type        = string
}

variable "state_lock_table_name" {
  description = "Terraform stateロック用DynamoDBテーブル名"
  type        = string
  default     = "g4-terraform-locks"
}

variable "github_repository" {
  description = "GitHub Actions OIDCの信頼対象リポジトリ(\"owner/repo\"形式)"
  type        = string
  default     = "k07g/g4"
}

variable "github_actions_environment" {
  description = "dev環境のTerraform applyロールをAssumeRoleWithWebIdentityできるGitHub Actions Environment名。ワークフロー側でjobsに`environment:`を指定すると、OIDCトークンのsubクレームはrepo:OWNER/REPO:ref:refs/heads/BRANCHではなくrepo:OWNER/REPO:environment:ENV_NAME形式になるため、ブランチではなくEnvironment名で信頼関係を設定する"
  type        = string
  default     = "dev"
}

variable "create_github_oidc_provider" {
  description = "GitHub ActionsのOIDCプロバイダをこのTerraformで新規作成するか。AWSアカウントに既に存在する場合(1アカウントにつき1つまで)はfalseにし、existing_github_oidc_provider_arnを指定する"
  type        = bool
  default     = true
}

variable "existing_github_oidc_provider_arn" {
  description = "create_github_oidc_provider = false の場合に使う既存のGitHub Actions OIDCプロバイダのARN"
  type        = string
  default     = ""
}
