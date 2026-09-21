terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.64"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.4"
    }
  }

  # まずはローカルstateで開始する。チームで共有する場合はS3 backend等への
  # 移行を検討すること(例: bucket/key/region/dynamodb_table を指定)。
  # backend "s3" {}
}

provider "aws" {
  region = var.aws_region
}
