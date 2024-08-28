# `fulcrum`
Executes the crux flocking protocol, a UDP based clustering mechanism with secured 
(Layer 4) encryption and leader election. 

Finds other nodes running `fulcrum` with the same shared secret, organizes them into 
a cluster with a leader. 
 
### `fulcrum` Functional Test

Executed with top level `crux/Makefile` via `make ftest`

[`crux/tests/functional/flock_test.go`](https://github.com/erixzone/crux/blob/master/tests/functional/flock_test.go)
 
Tests flocking algorithm without a cluster.  
See `TestBasic()` for variants of internal parameters used for debugging and testing. 


### `fulcrum` Integration Tests

Executed with top level `crux/Makefile` via `make itest`,  
using [`myriad`](https://github.com/erixzone/myriad) for notebook 'flea-circus' cluster.

[`crux/tests/integration/fulcrum1/test.sh`](https://github.com/erixzone/crux/blob/master/tests/fulcrum1/test.sh)  
Template - simple myriad execution test showing `flock version`, execution timestamp, `ls` 
of docker image `/usr/bin` contents, and test of a `myriad` file upload. Test does not 
build docker image.

[`crux/tests/integration/fulcrum2/test.sh`](https://github.com/erixzone/crux/blob/master/tests/fulcrum2/test.sh)  
Test builds docker image.  
Runs a `myriad` flocking test with 10 (`f1`-`f10`) nodes running `fulcrum flock`  
One node (`f99`) runs the beacon service with `fulcrum watch`  
Output logs for 10 nodes are named `f1-stdio, ..., f10-stdio`  
Beacon logs are named `f99-stdio`  
An awk script for analyzing beacon logs lives at `crux/tools/f99.awk`

### `fulcrum` Cloud Test

Requires: Digital Ocean account. Token goes in `crux/tests/cloud/basic/config.sh`. 

Executed with [`crux/tools/run-crux-build-and-cloud-test-with-docker-machine.sh`](https://github.com/erixzone/crux/blob/master/tools/run-crux-build-and-cloud-test-with-docker-machine.sh)  

[`crux/tests/cloud/basic/test.sh`](https://github.com/erixzone/crux/blob/master/tests/cloud/basic/test.sh)   
Uses `docker-machine` to initialize a docker swarm from 1-16 machines on Digital Ocean  
Creates a docker registry  
Pushes the test image to the registry  
Creates overlay network
Starts a beacon service with `fulcrum watch`  
Runs a flocking test with `fulcrum flock`   
Gets machine logs
Tears down the swarm cluster


### `fulcrum` Key Components:

- bootstrap code in [`pkg/ruck/bootstrap.go`](https://github.com/erixzone/crux/blob/master/pkg/ruck/bootstrap.go) via `BootstrapTest()`
- flocking protocol for leader election [`pkg/flock/flock/go`](https://github.com/erixzone/crux/blob/master/pkg/flock/flock.go)
- UDP based clustering, data exchange, node discovery [`pkg/flock/udp.go`](https://github.com/erixzone/crux/blob/master/pkg/flock/udp.go) 
- Layer 4 transport secured with encryption via initial shared symmetric key
- forward secrecy via Diffe-Hellman on ec25519 
- ongoing key rotation, leader monitoring [`pkg/flock/crypt.go`](https://github.com/erixzone/crux/blob/master/pkg/flock/crypt.go) 
- logging with `walrus` (a fork of logrus) [`pkg/walrus`](https://github.com/erixzone/crux/blob/master/pkg/walrus) 
and interface via [`pkg/clog`](https://github.com/erixzone/crux/blob/master/pkg/clog)
- enhanced Go errors with stack traces [`pkg/crux/error.go`](https://github.com/erixzone/crux/blob/master/pkg/crux/error.go) 
via `*crux.Err` return values


### `fulcrum` Command Line Help
````
	$ fulcrum -h
	Usage:
	  fulcrum [command]
	
	Available Commands:
	  flock       Run the flocking command with
	  help        Help about any command
	  key         Print a symmetric code
	  version     Print the version of fulcrum code
	  watch       Watch the results of flocking
	
	Flags:
	  -h, --help        help for fulcrum
	      --xx string   placeholder
	
	
	$ fulcrum flock --help
	nada
	
	Usage:
	  fulcrum flock [flags]
	
	Flags:
	      --beacon string   external coordination point (default "lodestar.org")
	  -h, --help            help for flock
	      --ip string       specify node ip (default "127.0.0.1")
	      --key string      specify secondary key
	      --name string     specify node name
	      --networks        comma-separated list of CIDR networks to probe	      --port int        use this port to listen for flocking traffic (default 23123)
	      --strew           just test strew initialisation


	Global Flags:
	      --xx string   placeholder

	
	
	$ fulcrum watch --help
	nada
	
	Usage:
	  fulcrum watch [flags]
	
	Flags:
	      --beacon string   specify node ip/port (default "127.0.0.1:28351")
	  -h, --help            help for watch
	      --key string      specify secondary key
	      --n int           exit with this count and stable
	
	Global Flags:
	      --xx string   placeholder
	
	
	Use "fulcrum [command] --help" for more information about a command.

	$ fulcrum key
	b1cbdcc38c1813afd9957547625a3b0dbe9ba6471a6d5d851e45d76567ce1794


	$ fulcrum version
	{
	  "Version": "",
	  "BuildDatetime": "UNKNOWN",
	  "CommitDatetime": "$Format:%cI$",
	  "CommitID": "$Format:%H$",
	  "ShortCommitID": "$Format:%h$",
	  "BranchName": "UNKNOWN",
	  "GitTreeIsClean": false,
	  "IsOfficial": false
	}
````
