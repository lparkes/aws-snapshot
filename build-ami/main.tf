terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      Owner        = "Lloyd Parkes"
      IaC-Source   = "ssh://github.com/lparkes/netbsd-aws"
      #Cost-Centre = ""
    }
}