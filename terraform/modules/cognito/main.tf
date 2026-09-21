locals {
  # ses_sender_email が指定されている場合のみ、SES経由のカスタムメール
  # 送信 (DEVELOPER) + パスワードリセットリンクを埋め込むLambdaを使う。
  # 空文字の場合はCognito標準送信のままで、コードのみの固定文面になる。
  use_custom_email = var.ses_sender_email != ""
}

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
    # ses_sender_email未指定の場合はSESを構築せず、Cognito標準のメール送信
    # (コードのみの固定文面)を使う。指定された場合のみSES経由のDEVELOPER
    # 送信に切り替え、Custom Message Lambda (下記) でメール本文を
    # カスタマイズ(パスワードリセットリンクの埋め込み)できるようにする。
    email_sending_account = local.use_custom_email ? "DEVELOPER" : "COGNITO_DEFAULT"
    source_arn            = local.use_custom_email ? aws_ses_email_identity.sender[0].arn : null
    from_email_address    = local.use_custom_email ? "career-sheet <${var.ses_sender_email}>" : null
  }

  dynamic "lambda_config" {
    for_each = local.use_custom_email ? [1] : []
    content {
      custom_message = aws_lambda_function.custom_message[0].arn
    }
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }

  deletion_protection = var.deletion_protection ? "ACTIVE" : "INACTIVE"

  tags = var.tags
}

# --- パスワードリセットメールへのリンク埋め込み (ses_sender_email指定時のみ) ---

resource "aws_ses_email_identity" "sender" {
  count = local.use_custom_email ? 1 : 0
  email = var.ses_sender_email
}

# Custom Message Lambda triggerはNode.jsで実装している(lambda/custom-message.js
# のコメント参照)。archiveプロバイダでソースファイルを直接zip化するだけで
# デプロイできるため、Goのビルド・別デプロイパイプラインが不要になる。
data "archive_file" "custom_message" {
  count       = local.use_custom_email ? 1 : 0
  type        = "zip"
  source_file = "${path.module}/lambda/custom-message.js"
  output_path = "${path.module}/lambda/custom-message.zip"
}

data "aws_iam_policy_document" "custom_message_trust" {
  count = local.use_custom_email ? 1 : 0
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "custom_message" {
  count              = local.use_custom_email ? 1 : 0
  name               = "${var.project_name}-${var.environment}-cognito-custom-message"
  assume_role_policy = data.aws_iam_policy_document.custom_message_trust[0].json

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "custom_message_logs" {
  count      = local.use_custom_email ? 1 : 0
  role       = aws_iam_role.custom_message[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "custom_message" {
  count            = local.use_custom_email ? 1 : 0
  function_name    = "${var.project_name}-${var.environment}-cognito-custom-message"
  role             = aws_iam_role.custom_message[0].arn
  handler          = "custom-message.handler"
  runtime          = "nodejs24.x"
  filename         = data.archive_file.custom_message[0].output_path
  source_code_hash = data.archive_file.custom_message[0].output_base64sha256
  timeout          = 5

  environment {
    variables = {
      FRONTEND_BASE_URL = var.frontend_base_url
    }
  }

  tags = var.tags
}

resource "aws_lambda_permission" "cognito_invoke" {
  count         = local.use_custom_email ? 1 : 0
  statement_id  = "AllowCognitoInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.custom_message[0].function_name
  principal     = "cognito-idp.amazonaws.com"
  source_arn    = aws_cognito_user_pool.this.arn
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
