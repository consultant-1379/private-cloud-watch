provider "aws" {
  region = "${var.region}"
}

provider "kubernetes" {}

# Create an ECR repository for crux.
resource "aws_ecr_repository" "crux" {
  name = "crux"
}

locals {
  crux_repo_url = "${aws_ecr_repository.crux.repository_url}"
}

# Locally run the crux build-and-push script.
resource "null_resource" "crux-build" {
  depends_on = ["aws_ecr_repository.crux"]

  provisioner "local-exec" {
    command = "./build-and-push-crux.sh ${var.github_token} ${local.crux_repo_url}"
  }
}

# Create a kubernetes deployment for beacon.
resource "kubernetes_deployment" "beacon" {
  depends_on = ["null_resource.crux-build"]

  metadata {
    name = "crux-beacon"
    labels {
      app = "beacon"
    }
  }

  spec {
    replicas = "1"

    selector {
      match_labels {
        app = "beacon"
      }
    }

    template {
      metadata {
        labels {
          app = "beacon"
        }
      }

      spec {
        container {
          image = "${local.crux_repo_url}:latest"
          name = "beacon"
          port {
            container_port = "${var.beacon_port}"
            protocol = "UDP"
          }
          env {
            name = "POD_IP"
            value_from {
              field_ref {
                field_path = "status.podIP"
              }
            }
          }
          # In testing, I had to run this as a loop, because it quits as soon
          # as it detects any sort of flock, and if we let the platform restart
          # it, kubernetes will think it's in a crashloop and will back off.
          # I don't know why beacon exits in this case.
          command = ["/bin/sh"]
          args = [
            "-c",
            "while true; do ripstop watch --beacon $(POD_IP):${var.beacon_port} --key ${var.flock_key} --n ${var.replicas}; sleep 1; done"
          ]

          # If this becomes unnecessary someday, you can use this instead.
          #command = ["ripstop"]
          #args = [
          #  "watch",
          #  "--beacon", "$(POD_IP):${var.beacon_port}",
          #  "--key", "${var.flock_key}",
          #  "--n", "${var.replicas}"
          #]
        }
      }
    }
  }
}

# Create a beacon service internal to the cluster.
resource "kubernetes_service" "beacon" {
  metadata {
    name = "crux-beacon"
  }

  spec {
    selector {
      app = "beacon"
    }
    session_affinity = "ClientIP"
    type = "ClusterIP"
    port {
      port = "${var.beacon_port}"
      protocol = "UDP"
    }
  }
}

# Create a kubernetes deployment for flock.
resource "kubernetes_deployment" "flock" {
  # This uses environment variables for beacon service discovery. As such, this
  # must be created after the beacon service is created. If you try to use DNS
  # discovery instead, it's the same deal, because if this starts before the
  # service, the flock processes will exit with:
  # dialing() failed: bad ip string 'beacon' (lookup beacon on 127.0.0.11:53: no such host)
  depends_on = [
    "null_resource.crux-build",
    "kubernetes_service.beacon"
  ]

  metadata {
    name = "crux-flock"
    labels {
      app = "flock"
    }
  }

  spec {
    replicas = "${var.replicas}"

    selector {
      match_labels {
        app = "flock"
      }
    }

    template {
      metadata {
        labels {
          app = "flock"
        }
      }

      spec {
        container {
          image = "${local.crux_repo_url}:latest"
          name = "flock"
          env = [
            {
              name = "POD_IP"
              value_from {
                field_ref {
                  field_path = "status.podIP"
                }
              }
            },
            {
              name = "POD_NAME"
              value_from {
                field_ref {
                  field_path = "metadata.name"
                }
              }
            }
          ]
          port {
            container_port = "${var.flock_port}"
            protocol = "UDP"
          }
          # Wrapped in an ssh agent because otherwise you'll get a fatal error
          # that looks like this: "SSH_AUTH_SOCK not set"
          command = ["ssh-agent"]
          args = [
            "/bin/sh",
            "-c",
            "ripstop flock --beacon $(CRUX_BEACON_SERVICE_HOST):$(CRUX_BEACON_SERVICE_PORT) --key ${var.flock_key} --ip $(POD_IP) --name $(POD_NAME) --networks ${var.cidr_block}"
          ]
        }
      }
    }
  }
}

