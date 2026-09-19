# --- Terraform state用バックエンド(S3 + DynamoDBロック) ---

resource "aws_s3_bucket" "terraform_state" {
  bucket = var.state_bucket_name

  # 誤ってstateバケットを削除してしまうと各環境のstateを失うため保護する
  lifecycle {
    prevent_destroy = true
  }

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
    Purpose   = "terraform-state"
  }
}

resource "aws_s3_bucket_versioning" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_dynamodb_table" "terraform_locks" {
  name         = var.state_lock_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
    Purpose   = "terraform-state-lock"
  }
}

# --- GitHub Actions用 OIDC IAMロール ---
# CIから長期クレデンシャルを使わずAWSを操作できるようにする。
# 信頼関係は github_repository の github_actions_branch ブランチへの
# push(= mainマージ)のみに限定する。

data "tls_certificate" "github_actions" {
  count = var.create_github_oidc_provider ? 1 : 0
  url   = "https://token.actions.githubusercontent.com"
}

resource "aws_iam_openid_connect_provider" "github_actions" {
  count = var.create_github_oidc_provider ? 1 : 0

  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.github_actions[0].certificates[0].sha1_fingerprint]
}

locals {
  github_oidc_provider_arn = var.create_github_oidc_provider ? aws_iam_openid_connect_provider.github_actions[0].arn : var.existing_github_oidc_provider_arn
}

data "aws_iam_policy_document" "github_actions_trust" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [local.github_oidc_provider_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_repository}:ref:refs/heads/${var.github_actions_branch}"]
    }
  }
}

resource "aws_iam_role" "terraform_ci" {
  name               = "${var.project_name}-dev-terraform-ci"
  assume_role_policy = data.aws_iam_policy_document.github_actions_trust.json

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

data "aws_iam_policy_document" "terraform_ci_permissions" {
  statement {
    sid    = "TerraformStateObjects"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["${aws_s3_bucket.terraform_state.arn}/*"]
  }

  statement {
    sid       = "TerraformStateBucketList"
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.terraform_state.arn]
  }

  statement {
    sid    = "TerraformStateLock"
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:DeleteItem",
      "dynamodb:DescribeTable",
    ]
    resources = [aws_dynamodb_table.terraform_locks.arn]
  }

  statement {
    # dev環境のCognitoユーザープール/アプリクライアントを作成・変更・削除
    # するために必要。このロールはmainブランチのCIワークフローからのみ
    # AssumeRoleWithWebIdentityできるため、実行経路はCIに限定される。
    sid       = "CognitoManagement"
    effect    = "Allow"
    actions   = ["cognito-idp:*"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "terraform_ci" {
  name   = "${var.project_name}-dev-terraform-ci"
  role   = aws_iam_role.terraform_ci.id
  policy = data.aws_iam_policy_document.terraform_ci_permissions.json
}
