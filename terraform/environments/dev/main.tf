module "cognito" {
  source = "../../modules/cognito"

  project_name = var.project_name
  environment  = "dev"

  tags = {
    Project     = var.project_name
    Environment = "dev"
    ManagedBy   = "terraform"
  }
}
