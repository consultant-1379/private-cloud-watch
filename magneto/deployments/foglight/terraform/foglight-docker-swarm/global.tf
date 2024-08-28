terraform {
  backend "swift" {
    container = "terraform-docker-swarm-state"
  }
}
