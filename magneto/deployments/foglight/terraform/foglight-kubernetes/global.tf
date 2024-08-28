terraform {
  backend "swift" {
    container = "terraform-kubernetes-state"
  }
}
