terraform {
  backend "swift" {
    container = "terraform-packer-state"
  }
}
