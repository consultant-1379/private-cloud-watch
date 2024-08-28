provider "kubernetes" {
}

# Set up RBAC for tiller.

resource "kubernetes_service_account" "tiller" {
  metadata {
    name = "tiller"
    namespace = "kube-system"
  }
}

resource "kubernetes_cluster_role_binding" "tiller" {
  metadata {
    name = "tiller"
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind = "ClusterRole"
    name = "cluster-admin"
  }
  subject {
    # Isn't this cute? If it's not set to empty, this won't work.
    # See: https://github.com/terraform-providers/terraform-provider-kubernetes/issues/204
    api_group = ""
    kind = "ServiceAccount"
    name = "tiller"
    namespace = "kube-system"
  }
}

# Install helm into kubernetes.

resource "null_resource" "helm_install" {
  depends_on = [
    "kubernetes_service_account.tiller",
    "kubernetes_cluster_role_binding.tiller"
  ]
  provisioner "local-exec" {
    command = "helm init --wait --upgrade --service-account tiller"
  }
  provisioner "local-exec" {
    when = "destroy"
    command = "kubectl -n kube-system delete deployment tiller-deploy"
  }
}

# Install consul and vault using helm.

provider "helm" {
  install_tiller = false
  namespace = "kube-system"
  service_account = "tiller"
}

resource "helm_release" "consul" {
  depends_on = ["null_resource.helm_install"]

  name = "consul"
  chart = "./consul/helm/"
}

resource "helm_release" "vault" {
  depends_on = ["helm_release.consul", "null_resource.helm_install"]

  name = "vault"
  chart = "./vault/helm/"
}
