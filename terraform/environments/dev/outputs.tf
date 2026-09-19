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

output "api_url" {
  description = "デプロイされたAPIのURL(ALBのAWS提供ドメイン、HTTP)"
  value       = "http://${aws_lb.app.dns_name}"
}

output "ecs_cluster_name" {
  value = aws_ecs_cluster.this.name
}

output "ecs_service_name" {
  value = aws_ecs_service.app.name
}

output "ecs_task_definition_family" {
  value = aws_ecs_task_definition.app.family
}
