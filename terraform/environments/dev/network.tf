# VPC・サブネットはこのTerraformでは作成しない。k07g/aws-bootstrapで
# 作成された既存のdev-vpc(var.vpc_id)と、そこに用意されたパブリック
# サブネット(var.public_subnet_ids)をそのまま利用する。サブネットは
# aws-bootstrap側のタグ付け規則に依存したくないため、データソースでの
# 自動検出ではなく変数で明示的に指定する方式にしている。
#
# data "aws_vpc" はvar.vpc_idの存在を早期に検証する目的で参照する。

data "aws_vpc" "this" {
  id = var.vpc_id
}
