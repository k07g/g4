module "cognito" {
  source = "../../modules/cognito"

  project_name = var.project_name
  environment  = "dev"

  ses_sender_email  = var.ses_sender_email
  frontend_base_url = var.frontend_base_url

  tags = {
    Project     = var.project_name
    Environment = "dev"
    ManagedBy   = "terraform"
  }
}
