data "aws_route53domains_registered_domains" "all" {
}

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "5.57.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
  allowed_account_ids = [
    "640581734156",
    # "088757392028"
  ]

  profile = "InfraDev-ViewOnly"
  # profile = "MasterPayer-PowerUser"
}