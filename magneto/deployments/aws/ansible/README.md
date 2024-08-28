Ansible is used to provision these Packer images.

The playbooks here are:

- base: a base role, which sets up things common to all images.
- head: a head node, including ldap, vault, and other infrastructure services.
- jump: a jump node, including persistent home directories.
- buildbot: a buildbot master node, which gets created using docker-compose,
  and some worker images in a separate directory which you can build and push
  to the registry whenever you need to update them.
- gerrit: a gerrit node, which gets created using docker-compose.

In order to read a playbook, start with the `main.yaml` file. Each one of them
imports a number of sub-playbooks, one for each group of tasks. I tried to group
them by function in ways that made sense.

There are also `files` and `scripts` and `certs` subdirectories. `files`
includes any configuration files and templates used by the playbook. All of the
templates use "jinja2" and have the suffix `.j2`. The other two subdirectories
should be self-explanatory.
