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

variable "vpc_cidr" {
  description = "dev環境用VPCのCIDR"
  type        = string
  default     = "10.20.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "パブリックサブネットのCIDR(ALB/ECS/RDSをここに配置し、NAT Gatewayなしで運用する)"
  type        = list(string)
  default     = ["10.20.0.0/20", "10.20.16.0/20"]
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
