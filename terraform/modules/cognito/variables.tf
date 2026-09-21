variable "project_name" {
  description = "リソース名のプレフィックスに使うプロジェクト名"
  type        = string
}

variable "environment" {
  description = "環境名(dev, stg, prod など)"
  type        = string
}

variable "deletion_protection" {
  description = "ユーザープールの削除保護。dev環境では作り直しやすいよう既定で無効"
  type        = bool
  default     = false
}

variable "password_minimum_length" {
  description = "パスワードの最小文字数"
  type        = number
  default     = 8
}

variable "mfa_configuration" {
  description = "MFA設定。OFF / OPTIONAL / ON"
  type        = string
  default     = "OFF"

  validation {
    condition     = contains(["OFF", "OPTIONAL", "ON"], var.mfa_configuration)
    error_message = "mfa_configuration must be one of: OFF, OPTIONAL, ON."
  }
}

variable "access_token_validity_minutes" {
  description = "アクセストークンの有効期限(分)"
  type        = number
  default     = 60
}

variable "id_token_validity_minutes" {
  description = "IDトークンの有効期限(分)"
  type        = number
  default     = 60
}

variable "refresh_token_validity_days" {
  description = "リフレッシュトークンの有効期限(日)"
  type        = number
  default     = 30
}

variable "tags" {
  description = "リソースに付与する共通タグ"
  type        = map(string)
  default     = {}
}

variable "ses_sender_email" {
  description = <<-EOT
    パスワードリセットメールの送信元として使う、SES検証済みのメール
    アドレス。空文字(既定値)の場合はCognito標準のメール送信
    (COGNITO_DEFAULT、コードのみの固定メール文面)を使い、SES/Lambdaに
    よるメール本文のカスタマイズ(URL埋め込み)は行わない。
    指定する場合、事前にそのアドレスでSESのメールアドレス検証を
    完了させておく必要がある(確認メールのリンクをクリックする)。
  EOT
  type        = string
  default     = ""
}

variable "frontend_base_url" {
  description = <<-EOT
    パスワードリセットメールに埋め込むリンクの起点となるフロントエンドの
    ベースURL (例: https://main.xxxxx.amplifyapp.com)。ses_sender_email
    を指定する場合は必須。
  EOT
  type        = string
  default     = ""

  validation {
    condition     = var.ses_sender_email == "" || var.frontend_base_url != ""
    error_message = "ses_sender_email を指定する場合は frontend_base_url も指定してください。"
  }
}
