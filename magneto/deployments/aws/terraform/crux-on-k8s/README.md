This terraform configuration builds and runs crux on kubernetes.

It was tested using ECR and EKS. The ECR parts could be changed to work
with any registry, and there is nothing EKS-specific about the config, so
in theory this should work on any kubernetes.

### Running this requires

- an amazon ec2 config on disk (for docker machine)
- a kubernetes config on disk (pointing to amazon EKS or somewhere else)
- a docker registry url (e.g. amazon ECR)
- a github token (to pull the crux repo)
- some utilities to start and use EKS. (see ../eks/README.md for details)
- some more utilities that the build script uses. (docker-machine v0.16.0, jq)

### General usage outline:

- Create your EKS cluster by applying the "../eks" terraform config.

    ( cd ../eks ; terraform init && terraform apply )

- Point your local kubeconfig at the EKS cluster you just created:

    aws eks update-kubeconfig --name erix

- Write the config map which is output after the terraform run to a file, and
  apply it to your cluster so that the workers can talk to each other:

    kubectl apply -f ~/config-map-aws-auth.yaml

- Set $TF_VAR_github_token to a github token that can access the crux repo.

    export TF_VAR_github_token="your token goes here"

- Build and deploy crux using this terraform config.

    terraform init && terraform apply

### What does this terraform config do, exactly?

When you run `terraform apply` on this config, the following things should
happen:

1. An ECR repository will be created to hold the crux builds.
2. A script will build the crux image and push it to the ECR repository. In
   order to keep from cluttering the local environment, it will use a temporary
   docker-machine node in AWS.
3. A kubernetes deployment will be created for the crux image which was just
   built.


