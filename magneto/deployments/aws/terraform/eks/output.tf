locals {
  config-map-aws-auth = <<EOF

apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    - rolearn: ${aws_iam_role.worker.arn}
      username: system:node:{{EC2PrivateDNSName}}
      groups:
        - system:bootstrappers
        - system:nodes

EOF
}

output "config-map-aws-auth" {
  value = "${local.config-map-aws-auth}"
}

resource "null_resource" "write-config-map" {
  provisioner "local-exec" {
    command = "echo '${local.config-map-aws-auth}' >./config-map-aws-auth.yaml"
  }
}
