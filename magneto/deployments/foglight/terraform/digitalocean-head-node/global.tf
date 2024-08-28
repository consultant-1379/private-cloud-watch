# We expect "access_key" and "secret_key" to be set in a tfvars file and
# passed to terraform init using the -backend-config flag.
#
# The "region" below is obviously not correct since we're using Digitalocean
# Spaces instead of Amazon S3, but this string must be set to an existing
# amazon S3 region to avoid error. Very strange...
#
terraform {
  backend "s3" {
    bucket = "terraform-state"
    key = "headnode/terraform.tfstate"
    region = "us-east-1"
    endpoint = "https://nyc3.digitaloceanspaces.com/"
    skip_credentials_validation = true
    skip_get_ec2_platforms = true
    skip_requesting_account_id = true
    skip_metadata_api_check = true
  }
}
