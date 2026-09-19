# アプリはメールアドレスをそのままユーザー名として SignUp / InitiateAuth
# するため、username_attributes = ["email"] で email をユーザー名として扱う。
# サインアップ確認は internal/auth の ConfirmSignUp(email, code) 実装に
# 合わせて、確認コードをメール送信する方式(デフォルト)を利用する。
resource "aws_cognito_user_pool" "this" {
  name = "${var.project_name}-${var.environment}"

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  mfa_configuration = var.mfa_configuration

  password_policy {
    minimum_length                   = var.password_minimum_length
    require_lowercase                = false
    require_uppercase                = false
    require_numbers                  = false
    require_symbols                  = false
    temporary_password_validity_days = 7
  }

  admin_create_user_config {
    # 自己サインアップを許可する(管理者作成のみに限定しない)
    allow_admin_create_user_only = false
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  email_configuration {
    # dev環境ではSESを構築せず、Cognito標準のメール送信を利用する
    email_sending_account = "COGNITO_DEFAULT"
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }

  deletion_protection = var.deletion_protection ? "ACTIVE" : "INACTIVE"

  tags = var.tags
}

# internal/auth/cognito.go は SignUp/InitiateAuth(USER_PASSWORD_AUTH) を
# 使うサーバーサイドの機密クライアントとして実装されているため、
# generate_secret = true とし、対応する認証フローのみを許可する。
resource "aws_cognito_user_pool_client" "app" {
  name         = "${var.project_name}-${var.environment}-app-client"
  user_pool_id = aws_cognito_user_pool.this.id

  generate_secret = true

  explicit_auth_flows = [
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  prevent_user_existence_errors = "ENABLED"

  access_token_validity  = var.access_token_validity_minutes
  id_token_validity      = var.id_token_validity_minutes
  refresh_token_validity = var.refresh_token_validity_days

  token_validity_units {
    access_token  = "minutes"
    id_token      = "minutes"
    refresh_token = "days"
  }
}
