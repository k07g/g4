output "state_bucket_name" {
  description = "GitHub Actions変数 TF_STATE_BUCKET に設定する値"
  value       = aws_s3_bucket.terraform_state.bucket
}

output "state_lock_table_name" {
  description = "GitHub Actions変数 TF_STATE_LOCK_TABLE に設定する値"
  value       = aws_dynamodb_table.terraform_locks.name
}

output "github_actions_role_arn" {
  description = "GitHub Actions変数 AWS_DEV_TERRAFORM_ROLE_ARN に設定する値"
  value       = aws_iam_role.terraform_ci.arn
}
