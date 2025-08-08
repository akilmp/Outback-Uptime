terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

resource "aws_s3_bucket" "logs" {
  bucket = var.bucket
}

variable "region" {}
variable "bucket" {}

output "bucket_name" {
  value = aws_s3_bucket.logs.bucket
}
