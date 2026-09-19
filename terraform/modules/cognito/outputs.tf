output "user_pool_id" {
  description = "Cognito User Pool ID (COGNITO_USER_POOL_ID)"
  value       = aws_cognito_user_pool.this.id
}

output "user_pool_arn" {
  description = "Cognito User Pool ARN"
  value       = aws_cognito_user_pool.this.arn
}

output "app_client_id" {
  description = "アプリクライアントID (COGNITO_CLIENT_ID)"
  value       = aws_cognito_user_pool_client.app.id
}

output "app_client_secret" {
  description = "アプリクライアントシークレット (COGNITO_CLIENT_SECRET)"
  value       = aws_cognito_user_pool_client.app.client_secret
  sensitive   = true
}
