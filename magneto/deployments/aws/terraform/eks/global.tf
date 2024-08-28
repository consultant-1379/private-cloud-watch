# The bucket below must be created by hand before terraform is run.
terraform {
  backend "s3" {
    bucket = "erix-terraform-state"
    key = "eks.tfstate"
    region = "us-east-2"
  }
}
