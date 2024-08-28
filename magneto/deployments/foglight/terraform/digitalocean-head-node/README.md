This is the terraform configuration that is used to create (and destroy) the
headnode and associated resources in DigitalOcean.

If a head node is already present, you'll need the private ssh key in order to
log in, so ask around. (It's probably Loren.)

In order to run this, the following is expected:

- You must use at least terraform v0.11.1 (any earlier won't be able to use
  Digitalocean Spaces as a backend).
- Your working path must be this directory when you run terraform.
- You must run `terraform init` using the -backend-config option to provide the
  `access_key` and `secret_key` for digitalocean spaces (get these from Loren).
- You've exported `TF_VAR_digitalocean_token` and `TF_VAR_github_token` with
  your authentication tokens before running `terraform apply` or any other
  terraform command after the init step.

Good luck.
