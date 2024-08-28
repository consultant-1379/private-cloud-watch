Packer configurations go here.

Packer is used to bake images which are then used by Terraform to create
instances in AWS.

The main driving force for all of this work is that doing all the provisioning
in Terraform is far from ideal... it is hard to write, hard to read, hard to
maintain, and runs very slowly.

This also gets us closer to the "immutable node" perfect devops wonderland.

I've decided to use Ansible to do all of the provisioning in these images,
because Packer's built-in provisioning abilities are even worse than
Terraform's. On a positive note though, after deailng with Hashicorp's built-in
stuff, Ansible is absolutely baller. This version has way, way less shell
scripting, as you can see.

Configuration descriptions
--------------------------

- node.json: a generic image that only uses the base role.
- head.json: a head node which runs an ldap service and a vault service.
- jump.json: a jump node that we use to log in and run builds, which includes a
  persistent /home volume.
- buildbot.json: a buildbot node, created using docker-compose.
- gerrit.json: a gerrit node, created using docker-compose.

How to run Packer for AWS
-------------------------

These Packer templates use the AWS credentials file at ~/.aws/credentials so
you have to create that first. I did this by installing the awscli tool:

     pip install awscli --upgrade --user

And then running `aws configure`:

    $ aws configure
    AWS Access Key ID [None]: ********
    AWS Secret Access Key [None]: ********
    Default region name [None]: us-east-1
    Default output format [None]: json

In order to get an access key and secret key, you have to create an IAM user in
the AWS console. I created mine with the "administrator" access policy
attached, which seems to be the one you need in order to use the AWS API to do
anything useful.

Some of the Packer templates pull values from the Erix production Vault, so you
need to login to Vault before running those. For some unknown reason, Packer
doesn't look in the .vault-token file for the token, so you'll have to run this
really awesome command after you log in to Vault:

    export VAULT_TOKEN=$(cat ~/.vault-token)

You will also need to install Packer. Something like this if you're on linux:

    wget https://releases.hashicorp.com/packer/1.3.1/packer_1.3.1_linux_amd64.zip
    unzip packer_1.3.1_linux_amd64.zip
    sudo mv packer /usr/local/bin/

And furthermore, you'll need to install ansible locally. Since it's Python
code, installing it is almost certainly going to be painful, but if you're
lucky, this might work (as of May 2018):

    python -m pip install --user --upgrade pip \
        # (may require pip 10+ in order to install without a segfault)
    pip install --user pysimplesoap \
        # (because somebody messed up when declaring their dependencies)
    pip install --user ansible \
        # (result is in $HOME/.local/bin which you should put in your PATH)

Once you've done all of this, you can invoke Packer in this way:

    packer build head.json

Debugging
---------

To debug packer, there are a couple tricks you can use...

The logging trick:

    PACKER_LOG=1 packer build node.json

The debugging trick, with which you can "stop time" and log in manually using the ssh key it provides:

    PACKER_LOG=1 packer build -debug node.json

And finally, the "ansible verbose" trick, where you add -vvvv to this line, like so:

    "extra_arguments": [ "-vvvv", "--extra-vars", ...
