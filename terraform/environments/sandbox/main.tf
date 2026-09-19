module "cognito" {
  source = "../../modules/cognito"

  project_name = var.project_name
  environment  = "sandbox"

  tags = {
    Project     = var.project_name
    Environment = "sandbox"
    ManagedBy   = "terraform"
  }
}
