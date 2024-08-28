You can use this terraform spec to create a Kubernetes cluster using Amazon
EKS! Maybe!

## Caveat

You might be in a position where you're working with multiple kubernetes
clusters. This work does NOT account for that. Pay attention and don't
overwrite your ~/.kube/config with something that you didn't want. In the
future, I'll account for this possibility in some better way.

## Background info

### Why is this necessary?

Amazon EKS gives you:

- The ability to create an AWS-specific cluster controller/control plane, into
  which you have basically no visibility. (This is what you get when you
  "create an EKS cluster".) It's supposedly redundant and you can rely on it.
- A worker AMI, maintained by Amazon, which includes a "bootstrap.sh" script
  that sets up enough stuff to give a worker the ability to connect to the
  controller.

Everything else, though, you create yourself. This includes everything in the
following list.

### The basic steps to build an EKS cluster and connect the workers:

- set up networking
    - create vpc
    - create at least two subnets
    - create an internet gateway
    - set up route table to route external traffic through internet gateway
- create iam role for cluster
    - create assume role policy
    - attach AmazonEKSClusterPolicy
    - attach AmazonEKSServicePolicy
- create cluster security group for communication with workers
    - allow communication on 443 to cluster API server from your jump node's ip, or wherever else you'll be running kubectl
    - (add another rule after creating workers)
- create eks master cluster ("control plane")
    - associate cluster security group
    - associate both subnets
    - depend on the iam role so we don't try to create it before the role is created? or depend on the policy attachments?
- create iam role for workers
    - create role
    - create assume role policy (same as the cluster one)
    - attach AmazonEKSWorkerNodePolicy
    - attach AmazonEKS_CNI_Policy
    - attach AmazonEC2ContainerRegistryReadOnly
    - create iam instance profile to use in the autoscaling launch configuration
- create security group for workers
    - add rule to let workers talk to each other on all ports
    - add rule to let cluster controller talk to workers on any port from 1025 to 65535
- add rule to cluster security group to allow workers to contact cluster controller on 443
- pull data for latest EKS worker AMI
- create aws autoscaling launch configuration for workers
    - set up userdata to run eks bootstrap script
    - use worker security group
    - use iam instance profile
    - associate_public_ip_address = true
    - create_before_destroy = true
- create autoscaling group
- configure kubectl on your client
    - kubectl requires aws-iam-authenticator
- apply a configmap that allows worker nodes to join the cluster via AWS IAM

### What's in each terraform file in this directory?

- global.tf - Connectivity information to get to the state file in S3.
- variables.tf - Variables that you can set in order to affect the installation.
- main.tf - Just the AWS provider configuration.
- iam.tf - IAM roles that the EKS nodes use to access each other and create AWS resources.
- networking.tf - VPC, subnets, and routing configuration.
- security.tf - Security groups and rules.
- cluster.tf - Creates the EKS cluster.
- workers.tf - Creates the EKS worker nodes.
- output.tf - Outputs the config map to grant the workers access to the cluster
  control plane, which you'll upload to your cluster using `kubectl apply`.

## How to actually run this code

### Prep your environment if you aren't running on jump.aws.erixzone.net:

The easiest way is to run this from the AWS jump node in us-east-1,
`jump.aws.erixzone.net`. These steps are already done for you there.

If you're choosing to do this on a different node, you'll need to install some
prerequisites. You can either run the provided `install-prerequisites.sh`
script, which installs into $HOME/.local/bin, or follow these steps by hand:

- Install a terraform binary. Currently this requires v0.11.10.

    wget https://releases.hashicorp.com/terraform/0.11.10/terraform_0.11.10_linux_amd64.zip && unzip terraform* && rm terraform_*

- Install a kubectl binary that matches the current EKS version. The current
  one from AWS is 1.10.3:

    curl -o kubectl https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/kubectl
    chmod 755 kubectl

- Install an aws-iam-authenticator binary that matches the EKS version:

    curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/aws-iam-authenticator
    chmod 755 aws-iam-authenticator

- Install the aws cli client. Something like this:

    pip install awscli --upgrade --user

- Make sure those binaries are in your path. (pip install --user installs to
  ~/.local/bin on my system)

- Change the terraform files so that they add a security group rule for
  whatever you're using as a jump node. (By default, it searches for a node
  tagged with the name "jump" in whatever region you're in.)

NOTE: Even though this last step is recommended by the docs, the security group
does not seem to have any effect if you have proper AWS credentials. I think
this is just to enable connectivity for people who don't have AWS credentials.
For the purpose of this doc, you need AWS creds anyway, so this last step
doesn't matter.

Then, continue on.

### Set up the state file storage location

In the file `global.tf`, there is an s3 bucket name and a key file name which
is used to save the terraform state. Each EKS instance needs to use a different
key file name. If you aren't on the erix team, you will also need to change the
bucket name to an S3 bucket that you create in your AWS account.

If you aren't planning on running this in a shared environment, you can also
delete this stanza, and terraform will store the state on your local disk.

### Customize the execution

Take a look in the variables.tf file and see if you want anything to be
different from the defaults.

If you want to create "your own" EKS cluster rather than the default "erix"
cluster, change these two things:

- the cluster name in variables.tf
- the state file name in global.tf (try something like "yourname-eks.tfstate")

### Create the EKS cluster

- Run `aws configure` and put in the necessary info.
    - If you don't have AWS credentials, you'll need to create a programmatic
      account in the IAM web UI.
    - Use region "us-east-1" if you're using the defaults.
- Change to this directory and run `terraform init`. (If it asks you to
  overwrite state in the new location, please type "no"!)
- Set the cluster name in `variables.tf` if you don't want the default of "erix".
- Run `terraform plan` and you should see that it wants to create a bunch of resources.
- Run `terraform apply`.

At this point, the cluster exists, but the workers can't join the cluster. In
order to get that going, you need to apply a config map.

### Apply the worker config map

- Point your kubeconfig at the cluster controller, substituting your cluster
  name for "erix" here:

    aws eks update-kubeconfig --name erix

- Test connectivity to the cluster controller:

    loren@jump:~$ kubectl get svc
    NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
    kubernetes   ClusterIP   172.20.0.1   <none>        443/TCP   35m

- In the output of the `terraform apply` that you ran, there will be a block of
  text beneath the line `config-map-aws-auth`. This block was also written to
  `config-map-aws-auth.yaml` in the current directory. Apply this file to the
  cluster controller:

    kubectl apply -f config-map-aws-auth.yaml

- Wait for your worker nodes to reach "ready" status:

    watch kubectl get nodes

You're done!

### Abort, abort!

To delete this EKS cluster and associated resources, run `terraform destroy`.

## How to use EKS

If the cluster already exists, and you just want to use it, just do the following:

- Run `aws configure` and put in the necessary info.
    - If you don't have AWS credentials, you'll need to create a programmatic
      account in the IAM web UI.
    - Use region "us-east-1".
- Point your kubeconfig at the cluster controller, substituting the cluster
  name for "erix" here if you aren't using the default:

    aws eks update-kubeconfig --name erix

That's it. You should now be able to run kubectl.

