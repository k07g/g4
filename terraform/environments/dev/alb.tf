resource "aws_lb" "app" {
  name               = "${var.project_name}-dev"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = var.public_subnet_ids

  # dev環境なので削除保護は無効(作り直しやすさを優先)
  enable_deletion_protection = false

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_lb_target_group" "app" {
  # vpc_idはforce-new属性のため、VPCを変更するapplyでは作り直しが発生する。
  # リスナーがtarget_group_arnを参照した状態で古いものを先に消そうとして
  # 失敗しないよう、create_before_destroyとname_prefix(ELBv2は最大6文字)
  # を使う。
  name_prefix = "dev-"
  port        = var.container_port
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.this.id
  target_type = "ip"

  lifecycle {
    create_before_destroy = true
  }

  health_check {
    path                = "/healthz"
    matcher             = "200"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  tags = {
    Project   = var.project_name
    ManagedBy = "terraform"
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.app.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
}
