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

resource "aws_vpc" "this" {
  cidr_block = var.cidr_block
}

variable "region" {
  description = "AWS region"
}

variable "cidr_block" {
  description = "VPC CIDR block"
}

output "vpc_id" {
  value = aws_vpc.this.id
}
