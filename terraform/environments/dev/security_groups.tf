resource "aws_security_group" "alb" {
  # vpc_id はforce-new属性のため、VPCを変更するapplyでは一度この
  # セキュリティグループが作り直される。name固定だと新旧の名前が
  # 衝突しうるため、create_before_destroyとあわせてname_prefixを使う。
  name_prefix = "${var.project_name}-dev-alb-"
  description = "Allow inbound HTTP from the internet"
  vpc_id      = data.aws_vpc.this.id

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name      = "${var.project_name}-dev-alb"
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_vpc_security_group_ingress_rule" "alb_http" {
  security_group_id = aws_security_group.alb.id
  description       = "HTTP from anywhere"
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 80
  to_port           = 80
  ip_protocol       = "tcp"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_egress_rule" "alb_all" {
  security_group_id = aws_security_group.alb.id
  description       = "Allow all outbound"
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "ecs_service" {
  name_prefix = "${var.project_name}-dev-ecs-service-"
  description = "ECS service tasks"
  vpc_id      = data.aws_vpc.this.id

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name      = "${var.project_name}-dev-ecs-service"
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_vpc_security_group_ingress_rule" "ecs_from_alb" {
  security_group_id            = aws_security_group.ecs_service.id
  description                  = "App traffic from the ALB only"
  referenced_security_group_id = aws_security_group.alb.id
  from_port                    = var.container_port
  to_port                      = var.container_port
  ip_protocol                  = "tcp"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_egress_rule" "ecs_all" {
  security_group_id = aws_security_group.ecs_service.id
  description       = "Allow all outbound (ECR pull, Cognito, RDS)"
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "rds" {
  name_prefix = "${var.project_name}-dev-rds-"
  description = "RDS PostgreSQL, reachable only from the ECS service"
  vpc_id      = data.aws_vpc.this.id

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name      = "${var.project_name}-dev-rds"
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_vpc_security_group_ingress_rule" "rds_from_ecs" {
  security_group_id            = aws_security_group.rds.id
  description                  = "PostgreSQL from the ECS service only"
  referenced_security_group_id = aws_security_group.ecs_service.id
  from_port                    = 5432
  to_port                      = 5432
  ip_protocol                  = "tcp"

  lifecycle {
    create_before_destroy = true
  }
}
