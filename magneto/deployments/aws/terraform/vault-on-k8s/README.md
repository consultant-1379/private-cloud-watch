This Terraform config can be used to start an HA Vault on Kubernetes.

## Caveats

You might be in a position where you're working with multiple Kubernetes
clusters and/or multiple instances of Helm. This work does NOT account for
that, and happily uses ~/.kube/config and ~/.helm. Pay attention and don't
overwrite or mutate anything that you care about. In the future, I'll account
for this possibility in some better way.

Also, this Vault is for development/demonstration purposes only. It is missing
things like TLS which we have on our production Vault and Consul. In order to
put any part of this into production somewhere, it will need to be made more
secure.

## How does this work?

Terraform will perform the following tasks, in order:

- Install Helm into your kubernetes cluster (it assumes RBAC is used such as in
  EKS, and installs a cluster role binding for Tiller as well)
- Use a Helm chart to install an HA Consul server cluster of 3 nodes using a
  StatefulSet, along with a PersistentVolumeClaim for each server instance
- Use that same Helm chart to install one Consul agent per worker node using a
  DaemonSet
- Use a Helm chart to install three HA Vault servers, pointing at the local
  Consul agents

## Where did these Helm charts come from?

The Consul Helm chart is the officially-provided Helm chart by the Consul
project, version 0.5.0.

The Vault Helm chart is part of the "Helm incubator", but I had to make a few
modifications to it in order to get it to substitute the local "status.hostIP"
into the Vault config on each Pod before running.

## How do you run this?

### Prerequisites

A kubernetes cluster using RBAC must exist, and your `~/.kube/config` must
be configured to point at it, with administrative rights. (You can use the
Terraform config at `../eks` to set something up for yourself.)

If you don't already have Helm and Tiller installed locally, you can use the
script at `./helm/get-helm.sh` to download and install it into /usr/local/bin.
It would be a good idea to update it first, though, like this:

    curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get > ./helm/get_helm.sh

You will also need a copy of Terraform, which you'll already have if you ran
`../eks/install-prerequisites.sh` during your kubernetes cluster creation.

### Set up the state file storage location

In the file `global.tf`, there is an s3 bucket name and a key file name which
is used to save the terraform state. Each instance needs to use a different key
file, so two people can't run this on two separate Kubernetes clusters without
one of them changing at least the key name.

If you aren't planning on running this in a shared environment, you can instead
delete this stanza from `global.tf`, and terraform will store the state on your
local disk.

### Run the terraform config

Run the terraform config in this manner:

    terraform init && terraform plan

You should see a plan of 5 actions. To execute them:

    terraform apply

To undo all of these actions and remove most of what was created:

    terraform destroy

To use the Vault service you've created, you can port-forward yourself
to the Vault service:

    kubectl port-forward service/vault-vault 8200

And then set up your vault client to talk to localhost:

    export VAULT_ADDR="http://localhost:8200"
    vault status

To connect to each individual Vault instance (e.g. to unseal each one after
a failure), you can get their ids with `kubectl get pods`, then run something
like this for a given pod:

    kubectl port-forward vault-vault-d6ff79df9-b8qkv 8200

The PersistentVolumeClaims will still stick around after you destroy, so if you
need to get your cluster back, another `terraform apply` will get you back to
the same Consul database state. To destroy those PVCs and reset the Consul
state, run this:

    kubectl delete pvc --selector="app=consul"


