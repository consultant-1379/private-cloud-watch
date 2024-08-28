# `ripstop` 

Executable that runs an integration test of a simple cluster "fabric".

Initializes the cluster via the flocking protocol in the same way as 
[`fulcrum`](https://github.com/erixzone/crux/tree/master/cmd/fulcrum).  

Adds on serial bootstrapping of gRPC cluster security infrastructure (Layer 7).  

Provides [`http-signatures`](https://tools.ietf.org/html/draft-cavage-http-signatures-10#section-3.1) 
with public-key distribution, client/endpoint service 
whitelisting rules, service registration and endpoint address distribution.  

Demonstrates how crux provides an automated bootstrap of a cluster to a running 
set of interconnected, secured, gRPC based API services.  

Following the principle of least priviledge, each client in the cluster knows 
what outbound service endpoints it can access, and each endpoint is automatically 
provided with a whitelist of allowed inbound clients.

#### Why Serial Bootstrap?

The `ripstop` executable builds on top of `fulcrum`, working through stages, 
one at a time, to bring up each node's services. This is a simple way of starting 
up, where if anything fails, it all fails. The term "serial bootstrap" distinguishs 
`ripstop` from a more sophisticated implementation - e.g.: a daemon based fabric 
bootstrap that monitors and restarts services or upgrades them from plugins. 
Presenting these in a linear or serial form allows us to reason about the 
"control plane" service behavior and understand how they interact - before 
building a more complicated bootstrap framework.

##### Stages in `ripstop`'s serial bootstrap on each node:

1. Initialize cluster as in `fulcrum`, wait for stable leader to emerge.
<hr><p>
Start Managing Services on Node and Cluster Leader.
2. Start `Reeve` service, which manages/checkpoints local node security 
information, and provides API access to local endpoints and clients.  
3. If this is the leader node, start the centralized `Registry` and 
`Steward` services.
4. Node is registered on the `Registry` encrypted gRPC port via a 2-factor, 
2-channel handshake, including a callback from `Registry` to the initiating 
`Reeve` that must be responded to with a proper signature. Thereafter, all 
gRPC communication is signed by client `ed25519` signatures, and validated 
at endpoint via a whitelist of pertinent client public keys using the 
[http-signatures](https://tools.ietf.org/html/draft-cavage-http-signatures-10#section-3.1) 
protocol on Layer 7* (Application Layer).
<hr><p>
Finish Connecting Managing Services `Reeve` and `Steward`
5. Node's `Reeve` waits for `Steward` to appear, and estabishes secured 
gRPC communication.
6. `Reeve` and `Steward` start using cluster service connectivity rules 
and the principle of least priviledge to distribute client public keys and 
endpoint addresses across the cluster. `Steward` pushes only relevant data 
to `Reeve`, it does not support queries.
7. Node's `Reeve` is used to register its own `Reeve` client and service 
with itself.
8. If this is the leader node, also register `Steward` client and service 
with leader node `Reeve`.
<hr><p>
Start Demo Service `Pastiche`
9. Node initializes pre-flight security information for `Pastiche`, a file 
transfer system, via it's local `ReeveAPI` (in-process, i.e. not via `gRPC`). 
10. Node starts `Pastiche` `gRPC` service and client with this information.
11. Node registers the running `Pastiche` client and endpoint with its local 
`Reeve`, which forwards the information to the cluster leader's `Steward`. 
`Steward` applies rules, distributes filtered info to `Reeve` on relevant 
nodes in the cluster, which in turn updates whitelist and endpoint 
information. Hereafter, `Pastiche` can do it's thing.
<hr><p>
Send Files To Another Pastiche In The Cluster
12. `Pastiche` asks local `Reeve` for a catalog of services it is allowed to 
use as a client. (i.e. it can only talk to other instances of `Pastiche`)
13. `Pastiche` asks local `Reeve` for a list of `Pastiche` endpoints to which 
it can connect. (i.e. a list of `Pastiche` endpoint addresses in the same horde)  
14. `Pastiche` file transfer is tested with one of those endpoints. 

*A Layer 4 transport security system (e.g. TLS) can be added to encrypt 
gRPC traffic. 

### `ripstop` Integration Test

Executed with top level `crux/Makefile` via `make itest`, using 
[`myriad`](https://github.com/erixzone/myriad) for notebook 'flea-circus' cluster.

[`crux/tests/integration/ripstop/`](https://github.com/erixzone/crux/blob/master/tests/ripstop/)  

Automated Test script `test.sh` is run by `make itest`  
Assumes docker image is built previously by `tests/integration/fulcrum2`  
Runs a `myriad` test with 10 nodes (`f1`-`f10`) running `ripstop flock`  
One node (`f99`) runs the beacon service with `ripstop watch`  
Splits nodes into 2 separate hordes, `sharks` and `jets`.   
Output logs for 10 nodes are named `f1-stdio, ..., f10-stdio`  
Beacon logs are named `f99-stdio`.  

Success tests examines logs for the following:

- startup of `reeve` on each node
- startup of `registry` on 1 leader
- statup of `steward` on 1 leader 
- all nodes: registration of `reeve` via `registry`
- all nodes: communication between `reeve` and `steward`
- all nodes: startup of `patstiche` 
- `steward` reciept of all endpoints forwarded from all `reeve` instances
- `steward` receipt of all clients forwarded from all `reeve` instances
- all nodes: `reeve` EndpointsUp result for `pastiche`
- all nodes: `reeve` Catalog result for `pastiche`.

##### Variations
Can force a docker image build by executing  
 `./test_sub.sh 1 10` (instead of `./test.sh`)  
Can add more nodes, e.g. 20 with  
`./test_sub.sh 0 20`.  
To run multiple tests (e.g. 100 on 16 nodes)   
and log failures into numbered directories use  
`./multi-run.sh 100 16` 

### `ripstop` - Key Components:

- TODO ... 
- everything from [`fulcrum`](https://github.com/erixzone/crux/tree/master/cmd/fulcrum)
- wait algorithm for flock stability, emergence of a leader, 
and registry node. [pkg/ruck/bootstrap.go](https://github.com/erixzone/crux/tree/master/pkg/ruck/bootstrap.go) `FlockWait()`
- `reeve` service [pkg/reeve](https://github.com/erixzone/crux/tree/master/pkg/reeve)
	- establishes/retrieves node local identity and checkpoint/restart data - [pkg/muck](https://github.com/erixzone/crux/tree/master/pkg/muck) 
	- Provides checkpoint restart of essential identity informaiton
	- Makes and maintains local key pairs (via `ssh-keygen`)
	- Registers local node service endpoints and clients
	- Provides client signing (via `ssh-agent` through a Unix socket) 
	- Recieves push updates from `steward` of cluster public keys, endpoint 
addresses, and gRPC service connectivity rules 
	- Maintains local public key whitelist and endpoint database. 
-  `register` service on leader node
	
- 	`steward` service on leader node
	- While `steward` maintains a service registry database - is not 
API query-able. `steward` pushes relevant data to each `reeve` so that 
`reeve` provides the catalog and endpoint query information to services 
on the same node.

### `ripstop` Command Line Help

	$ ripstop -h
	Does not use plugins.

	Usage:
	  ripstop [command]

	Available Commands:
	  flock       Run the flocking command with
	  help        Help about any command
	  key         Print a symmetric code
	  version     Print the version of ripstop code
	  watch       Watch the results of flocking
	
	Flags:
	  -h, --help        help for ripstop
          --xx string   placeholder
	
	Use "ripstop [command] --help" for more
	information about a command.
	
	
	$ ripstop flock --help
	nada
	
	Usage:
	  ripstop flock [flags]
	
	Flags:
          --beacon string   external coordination point (default "lodestar.org")
      -h, --help            help for flock
          --horde string    name of horde for endpoints (default "sharks")
          --ip string       specify node ip (default "127.0.0.1")
          --key string      specify secondary key
          --name string     specify node name
          --networks        comma-separated list of CIDR networks to probe
          --port int        use this port to listen for flocking traffic (default 23123)
	
	Global Flags:
      --xx string   placeholder


	$ ripstop watch --help
	nada
	
	Usage:
	  ripstop watch [flags]
	
	Flags:
          --beacon string   specify node ip/port (default "127.0.0.1:28351")
      -h, --help            help for watch
          --key string      specify secondary key
          --n int           exit with this count and stable
	
	Global Flags:
      --xx string   placeholder
