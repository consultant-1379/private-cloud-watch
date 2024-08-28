This terraform configuration creates a docker swarm cluster inside Foglight.

You must apply this from a node inside Foglight, such as the head node. If you
try to apply it from outside Foglight, you won't be able to provision the
instances you create, so the apply will fail.
