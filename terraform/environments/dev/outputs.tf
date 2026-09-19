output "aws_region" {
  value = var.aws_region
}

output "cognito_user_pool_id" {
  description = ".env の COGNITO_USER_POOL_ID に設定する値"
  value       = module.cognito.user_pool_id
}

output "cognito_client_id" {
  description = ".env の COGNITO_CLIENT_ID に設定する値"
  value       = module.cognito.app_client_id
}

output "cognito_client_secret" {
  description = ".env の COGNITO_CLIENT_SECRET に設定する値"
  value       = module.cognito.app_client_secret
  sensitive   = true
}
