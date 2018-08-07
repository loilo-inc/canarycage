variable "s3_bucket" {
  default = "loilonote-terraform-state"
}
terraform {
  backend "s3" {
    bucket = "${var.s3_bucket}"
    key = "canarycage/test.tfstate"
    region = "us-west-2"
  }
}
provider "aws" {
  profile = "default"
  region = "us-west-2"
}