resource "aws_db_subnet_group" "this" {
  name       = "${var.project_name}-dev"
  subnet_ids = aws_subnet.public[*].id

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "random_password" "db" {
  length  = 32
  special = false
}

resource "aws_db_instance" "app" {
  identifier     = "${var.project_name}-dev"
  engine         = "postgres"
  engine_version = "17"

  instance_class    = var.db_instance_class
  allocated_storage = var.db_allocated_storage
  storage_type      = "gp3"
  storage_encrypted = true

  db_name  = var.db_name
  username = var.db_username
  password = random_password.db.result

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false

  # dev環境なので作り直しやすさを優先する
  multi_az                = false
  backup_retention_period = 0
  deletion_protection     = false
  skip_final_snapshot     = true
  apply_immediately       = true

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

# アプリが単一のDATABASE_URLとして読めるよう、接続文字列をまとめて
# Secrets Managerに保存する(ECSタスク定義からsecretsとして参照する)。
resource "aws_secretsmanager_secret" "database_url" {
  name                    = "${var.project_name}/dev/database-url"
  recovery_window_in_days = 0

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id     = aws_secretsmanager_secret.database_url.id
  secret_string = "postgres://${var.db_username}:${urlencode(random_password.db.result)}@${aws_db_instance.app.address}:${aws_db_instance.app.port}/${var.db_name}?sslmode=require"
}

resource "aws_secretsmanager_secret" "cognito_client_secret" {
  name                    = "${var.project_name}/dev/cognito-client-secret"
  recovery_window_in_days = 0

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "cognito_client_secret" {
  secret_id     = aws_secretsmanager_secret.cognito_client_secret.id
  secret_string = module.cognito.app_client_secret
}
