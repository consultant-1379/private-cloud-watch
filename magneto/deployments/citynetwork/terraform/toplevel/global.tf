terraform {
  backend "swift" {
    container = "terraform-toplevel-state"
  }
}
