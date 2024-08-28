This is the terraform configuration that manages the resources that are needed
in order to run Packer in City Network. At the very least, this creates a
private network, as well as a node which can be used to run Packer.

It expects that your OS env variables are set, which can be accomplished by
sourcing the `citynetwork_openrc.sh` script in `../../bin`.

After running `terraform apply` successfully, you can log in to the packer node
and accomplish a packer build of the base image using approximately these
steps (please modify them to suit your environment, if necessary):

```
loren@local:~/gerrit/magneto/deployments/citynetwork/terraform/packer$ scp -r ../../. ubuntu@$(terraform output packer-address):
loren@local:~/gerrit/magneto/deployments/citynetwork/terraform/packer$ ssh ubuntu@$(terraform output packer-address)
Welcome to Ubuntu 16.04 LTS (GNU/Linux 4.4.0-21-generic x86_64)
...
Last login: Tue May 29 19:34:25 2018 from 74.103.239.241

ubuntu@packer:~$ source bin/citynetwork-openrc.sh
Please enter your OpenStack username for project Default Project 33026: loren
Please enter your OpenStack Password for project Default Project 33026 as user loren:
Done!
ubuntu@packer:~$ export PATH=~/.local/bin:$PATH
ubuntu@packer:~$ cd packer
ubuntu@packer:~$ packer build node.json
(packer output follows...)
```


