terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.64"
    }
  }

  # CI(GitHub Actions)からterraform applyを実行するため、S3 backendで
  # stateを永続化する。bucket/key/region/dynamodb_table はこのファイルに
  # 直書きせず、`terraform init -backend-config=...` で渡す(ローカルでは
  # backend.hcl、CIでは環境変数/GitHub Actions変数を利用)。
  # 値の生成方法は ../../bootstrap を参照。
  backend "s3" {}
}

provider "aws" {
  region = var.aws_region
}
