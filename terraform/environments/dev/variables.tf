variable "aws_region" {
  description = "AWSリージョン"
  type        = string
  default     = "ap-northeast-1"
}

variable "project_name" {
  description = "リソース名のプレフィックスに使うプロジェクト名"
  type        = string
  default     = "g4"
}

variable "vpc_id" {
  description = "アプリを配置する既存VPCのID。k07g/aws-bootstrapで作成されたdev-vpcを利用し、このTerraformではVPC自体は作成しない"
  type        = string
  default     = "vpc-065a077399a856916"
}

variable "public_subnet_ids" {
  description = "ALB/ECS/RDSを配置する既存のパブリックサブネットID(k07g/aws-bootstrap側で作成したもの)。ALBおよびRDSサブネットグループの要件により、異なるAZのサブネットを2つ以上指定する必要がある"
  type        = list(string)
  default     = ["subnet-0280760eb56c3d720"]

  validation {
    condition     = length(var.public_subnet_ids) >= 2
    error_message = "public_subnet_ids には異なるAZのサブネットを2つ以上指定してください(ALB/RDSサブネットグループの要件のため)。"
  }
}

variable "container_port" {
  description = "アプリコンテナがリッスンするポート"
  type        = number
  default     = 8080
}

variable "container_cpu" {
  description = "ECSタスクのCPUユニット(dev向けに最小構成)"
  type        = number
  default     = 256
}

variable "container_memory" {
  description = "ECSタスクのメモリ(MiB、dev向けに最小構成)"
  type        = number
  default     = 512
}

variable "desired_count" {
  description = "ECSサービスの希望タスク数"
  type        = number
  default     = 1
}

variable "log_retention_days" {
  description = "CloudWatch Logsの保持日数"
  type        = number
  default     = 14
}

variable "db_instance_class" {
  description = "RDSインスタンスクラス(dev向けに最小構成)"
  type        = string
  default     = "db.t4g.micro"
}

variable "db_allocated_storage" {
  description = "RDSのストレージサイズ(GB)"
  type        = number
  default     = 20
}

variable "db_name" {
  description = "アプリが使うデータベース名"
  type        = string
  default     = "g4"
}

variable "db_username" {
  description = "RDSマスターユーザー名"
  type        = string
  default     = "g4"
}
