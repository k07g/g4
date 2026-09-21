terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    # ses_sender_email 指定時、Custom Message Lambda (lambda/custom-message.js)
    # をzip化するために使う。
    archive = {
      source = "hashicorp/archive"
    }
  }
}
