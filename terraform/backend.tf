terraform {
  backend "s3" {
    bucket = "loilo-terraform-state"
    key = "canarycage/test.tfstate"
    region = "us-west-2"
  }
}