# CRUX
A golang toolkit for making reproducible distributed software systems. 

Provides reproducible build artefacts, secure initial cluster formation, 
leader election (`pkg/flock`), secured gRPC service endpoints (`pkg/grpcsig`), 
and rules-based public key and endpoint address distribution (`pkg/reeve`, `pkg/steward`).

##### N.B.
Crux is a work in progress.   
Repo documentation (this `readme.md` and others) represent crux "as-is".  
It does not reflect ongoing work areas or outline future changes/plans.

### Top Level Executables
Executable links below provide documentation about:   
- executabe function   
- high-level cluster testing (integration and/or cloud)  
- links to key `/crux/pkg` and source components  
- command line utilization


[`crux/cmd/crux`](https://github.com/erixzone/crux/tree/master/cmd/crux)  
Simple template/demo command executable with embedded build annotation.

[`crux/cmd/fulcrum`](https://github.com/erixzone/crux/tree/master/cmd/fulcrum)
Cluster Formation via Flocking Protocol: Continuous UDP based clustering with 
Layer 4 secured packet encryption and leader election over the local network 
or a list of CIDR specified networks.

[`crux/cmd/ripstop`](https://github.com/erixzone/crux/tree/master/cmd/ripstop)
Cluster Fabric Bootstrap with Services: Flocking protocol (as above) with 
bootstrap and registration of Layer 7 `http-signatures` secured `gRPC` services. 
Automated distribution of `gRPC` client public keys and endpoint addresses to nodes, 
according to service access rules and the principle of least priviledge. An example 
service,  `Pastiche`, transfers files across the test cluster.
 
## BUILDING CRUX

### UBUNTU LINUX SETUP
On a fresh Ubuntu Linux system you will need the following packages to build crux:

	sudo apt-get install autoconf build-essential \
	ca-certificates curl gawk libtool pkg-config \
	unzip wget yasm
	
You also need to install [Go](https://golang.org/doc/install), with the target directory for build executables in the `PATH`.  
 Ensure your `~/.bash_profile` has the following lines,  
 substituting the path to your Golang working directory for `/build/go`:

	export GOROOT=/usr/local/go
	export GOPATH=${HOME}/build/go
	export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

Then from the command prompt source the file, and ensure Go is responding. 

	source ~/.bash_profile
	go --version

Next, install [Docker CE](https://docs.docker.com/v17.09/engine/installation/linux/docker-ce/ubuntu/#upgrade-docker-ce) following the instructions on the link.

Then, proceed to Clone the repo.

### MAC OS X SETUP

todo. 



### Clone the repo

Set up the path to the `crux` repo,  
following the Go convention, where you will clone the repo.  
(i.e. into `${GOPATH}/src/github.com/erixzone/crux`)

	mkdir -p ${HOME}/build/go/src/github.com/erixzone
	cd ~/source/go/src/github.com/erixzone
	

From here you have 2 options to clone and build the repo.  
Both of these require some setup of ssh keys and accounts.  

First, the basic commands to clone the repos, then some links to setup ssh-keys and accounts.
  
#### GitHub - build only

		git clone git@github.com:erixzone/crux.git

#### Erixzone Gerrit - build and contribute

Substitute your Erixzone Gerrit account username for `username`:

		gerrit clone ssh://username@review.eng.erixzone.net:29418/crux


#### To set up `git clone` on [GitHub](https://github.com). 

`crux` is a private GitHub repo.  
You will need

1. Team membership in Erixzone on GitHub.
2. [ssh public key set with GitHub](https://help.github.com/articles/connecting-to-github-with-ssh/) for command line access.

#### To set up `gerrit clone` on [gerrit.erixzone.net](https://gerrit.erixzone.net)

1. Talk to a team member to get an account set up on [gerrit.erixzone.net](https://gerrit.erixzone.net) 
2. Your ssh public key must be installed on the Erixzone Gerrit server via its web interface.  
(Click on your name on the right, then Settings, then SSH Public Keys on the left menu.)
3. You need the `gerrit` shell script installed in `/usr/local/bin/` to clone  
the repository and set the git hooks for contributing.

## Build and Test Crux

	make
	make test

On completion you should find executables:

	ls $GOPATH/bin
	crux	    golex    goyacc	         monocle-grpc    pastiche             protoc-gen-go              registercli-test  server
	enumerator  golint   grpcsigcli-test monocle-swagger pastiche-demo        protoc-gen-grpc-gateway    registersrv-test  stringer
	fulcrum     govendor grpcsigsrv-test myriad          pastiche-testserver  pubkeydb                   ripstop           testgen

	

### Integration tests

Docker based integration tests are run with:

	make itest

To bootstrap a "flea-circus" `crux` cluster on a development notebook, 
we use [`myriad`](https://github.com/erixzone/myriad) (which is provided 
in the vendor section of the `crux` repo) and [Docker] Myriad's `readme.md` 
may be helpful to understand the myriad file which launches cluster jobs. 

`Myriad` can provide a cluster from 2-50 nodes on a Linux or MacOS notebook 
with 16GB of system memory.

`Myriad` launches multiple Docker containers with provided commands, arguments, 
and specified input files (imported into the running docker container). 
For output, `myriad` exports each node's `stdout` (i.e. logs). Any 
user-specified files can also be exported from the docker container for 
each cluster node instance.

The files required to test `ripstop` with `myriad` are in 
[`crux/tests/integration/ripstop/`](https://github.com/erixzone/crux/tree/master/tests/integration/ripstop), 
and these are executed with the top-level `crux/Makefile` invoked with `make itest` 
after a successful `make test`. The `ripstop` logs can be seen in this directory 
after a `crux` bootstrapping run. So far, there is no mechanism for interacting 
with a running `myriad` cluster at this stage.

See the command documentation for [`fulcrum`](https://github.com/erixzone/crux/tree/master/cmd/fulcrum) 
and  [`ripstop`](https://github.com/erixzone/crux/tree/master/cmd/ripstop) for more information about 
their respective Integration Tests. 


### Cloud Testing

Initial testing of the flocking protocol in the cloud is limited to a `docker-machine` 
invoked swarm on Digital Ocean. See `crux/tests/cloud` and notes in the command 
documentation for [`fulcrum`](https://github.com/erixzone/crux/tree/master/cmd/fulcrum). 

### We are not alone

For later integration testing, you need more than just crux; you need myriad and prometheus as well.
The following steps use the `project` command (from somewhere) and assume you have no code.
(If you have code, omit the 'clone' parameter.)

- make myriad
. project -p myriad clone
make install
cp myriad /xxx  # we need for binary for the next step; /xxx needs to be your normal bin dir

- make prometheus container
. project -p crux clone
cd ../magneto/prometheus/prometheus
rm -f prom*.gz    # get rid of old detritus
make container	# this builds from committed source (so do a git commit before the make)

- do your integration test. E.g.
cd ../../../crux/tests/integration/fulcrum
./test1.sh

## CRUX Internals

### CRUX Service Packages


#### [`pkg/reeve`](https://github.com/erixzone/crux/tree/master/pkg/reeve)  
Reeve manages node registration and service security information. It runs.  
It is used for:  
- Registering a node for centralized service managment and public key distribution.  
- Managing assymetric key pairs, client gRPC call signing and service gRPC 
signature validation, and  
- The communications agent acting that pushes service client and endpoint updates 
to the centralized node registry `steward`  
- The listening agent, recieving push-updates from `steward` of whitelisted public 
keys and available service endpoints. 

Service interface protobuf file is [`protos/srv_reeve.proto`](https://github.com/erixzone/crux/blob/master/protos/srv_reeve.proto).  
Generated code for gRPC client/server API is [`gen/cruxgen/srv_reeve.pb.go`](https://github.com/erixzone/crux/blob/master/gen/cruxgen/srv_reeve.pb.go) 

Reeve is the first package to start in `fulcrum` and `ripstop` after a flocking 
leader is elected. Reeve initializes/reloads local storage `pkg/muck` and manages 
keypairs and key generation via `ssh-keygen`.  It manages client-side public-key 
signing by `ssh-agent` for signed gRPC communcation via `pkg/grpcsig`, and provides 
the interface for endpoints of services to validate signed gRPC API calls against 
public-keys in the local whitelist database. 

Reeve has a local API (function calls in the same process - see 
[`pkg/crux/reeveapi.go`](https://github.com/erixzone/crux/blob/master/pkg/crux/reeveapi.go) - 
for the signing and validation functions. 

````
type ReeveAPI interface {
	SetEndPtsHorde(string) string
	ReeveCallBackInfo() (string, string, string, string, interface{})
	LocalPort() int
	StartStewardIO(time.Duration) *Err
	SecureService(string) interface{}
	ClientSigner(string) (interface{}, *Err)
	SelfSigner() interface{}
	PubKeysFromSigner(interface{}) (string, string)
}
````

Reeve has a `gRPC` API that exposes functions for local node endpoint and 
client advertisement to the cluster.  Reeve can provide a catalog of services 
that a client can connect to - following the principle of least priviledge. 
A client must have a rule allowing connectivity before it can see that service 
listed in its catalog listing. Reeve provides a list of service endpoints 
available for signed gRPC communication.

````
service Reeve {
	rpc PingTest (Ping) returns (Ping) {}
	rpc Heartbeat(HeartbeatReq) returns (HeartbeatReply) {}

	rpc RegisterEndpoint(EndpointInfo) returns (Acknowledgement) {}
	rpc RegisterClient(ClientInfo) returns (Acknowledgement) {}
	rpc EndpointsUp(EndpointRequest) returns (Endpoints) {}
	rpc Catalog(CatalogRequest) returns (CatalogReply) {}
````

Reeve also has gRPC API funcitons that communicate with `steward` and 
`register` for managing rules-based public key whitelists and lists of available 
endpoints, which are pushed to `reeve` by `steward` as changes to cluster emerge.

````
	rpc UpdateCatalog(CatalogList) returns (Acknowledgement) {}
	rpc UpdatePubkeys(SignWith) returns (PubKeysUpdate) {}
	rpc WlState(StateId) returns (Acknowledgement) {}
	rpc EpState(StateId) returns (Acknowledgement) {}
	rpc WlUpdate(WlList) returns (Acknowledgement) {}
	rpc EpUpdate(EpList) returns (Acknowledgement) {}

}
````


#### [`pkg/pastiche`](https://github.com/erixzone/crux/tree/master/pkg/pastiche)  
Service providing gRPC based file transfer (with cluster file lookup by MD5 hash) 
between cluster nodes.  
Protobuf declaration in [`protos/srv_pastiche.proto`](https://github.com/erixzone/crux/blob/master/protos/srv_pastiche.proto).  
Generated code for gRPC client/server API is [`gen/cruxgen/srv_pastiche.pb.go`](https://github.com/erixzone/crux/blob/master/gen/cruxgen/srv_pastiche.pb.go)  

````
service PasticheSrv {
	rpc PingTest (Ping) returns (Ping) {}
	rpc AddDataStream(stream AddRequest) returns (AddResponse) {}
	rpc GetDataStream(GetRequest) returns (stream GetResponse) {}
	rpc GetPath( PathRequest ) returns (PathResponse) {}
	rpc RegisterFile(RegisterFileRequest) returns (RegisterFileResponse) {}
	rpc Reserve(ReserveRequest) returns (ReserveResponse) {}
	rpc Delete(DeleteRequest) returns (DeleteResponse) {}
	rpc DeleteAll(DeleteAllRequest) returns (DeleteAllResponse) {}
	rpc AddTar(AddTarRequest) returns (AddTarResponse) {}
	rpc AddTarFromRemote(AddTarFromRemoteRequest) returns (AddTarFromRemoteResponse) {}
	rpc AddDirToCache(AddDirToCacheRequest) returns (AddDirToCacheResponse) {}
	rpc AddFilesFromDir(AddFilesFromDirRequest) returns (AddFilesFromDirResponse) {}
	rpc Heartbeat(HeartbeatReq) returns (HeartbeatReply) {}
}
````

#### [`pkg/register`](https://github.com/erixzone/crux/tree/master/pkg/register)  
The centralized node registry service providing the 2-factor, 2-channel 
authentication mechanism for `reeve` to gain access to `steward`.  
Protobuf declaration in [`protos/register.proto`](https://github.com/erixzone/crux/blob/master/protos/register.proto)  
Generated code for gRPC client/server API is [`gen/cruxgen/srv_steward.pb.go`](https://github.com/erixzone/crux/blob/master/gen/cruxgen/srv_steward.pb.go)  

````
service Registry {
  rpc PingTest (Ping) returns (Ping) {}
  rpc Register (CallBackEnc) returns (stream RegisterInfo) {}
}
````


#### [`pkg/steward`](https://github.com/erixzone/crux/tree/master/pkg/steward)  
Centralized agent that recieves updates of endpoints and clients from `reeve`, 
distributes public keys and endpoints to cluster nodes via push to `reeve`.  
Protobuf declaration in [`protos/srv_steward.proto`](https://github.com/erixzone/crux/blob/master/protos/srv_steward.proto)
Generated code for gRPC client/server API is [`gen/cruxgen/srv_steward.pb.go`](https://github.com/erixzone/crux/blob/master/gen/cruxgen/srv_steward.pb.go)  

Steward maintains a datatabase 
(see [`pkg/registrydb`](https://github.com/erixzone/crux/tree/master/pkg/registrydb)) 
of nodes and update status, applies rules (declared currently in 
[`pkg/registrydb/allowed.go`](https://github.com/erixzone/crux/blob/master/pkg/registrydb/allowed.go)) 
to select which client-to-endpoint connections are allowed, and pushes the appropriate public 
keys and endpoint lists out to the nodes. Steward key and endpoint distribution operates on 
the principle of least priviledge. Steward has no direct query API, it listens to authorized 
`reeve` for service updates, then acts accordingly.

````
service Steward {
	rpc PingTest (Ping) returns (Ping) {}
	rpc Heartbeat(HeartbeatReq) returns (HeartbeatReply) {}
	rpc EndpointUpdate(EndpointData) returns (Acknowledgement) {}
	rpc ClientUpdate(ClientData) returns (Acknowledgement) {}
}
````


### CRUX Infrastructure Packages

[`pkg/crux`](https://github.com/erixzone/crux/tree/master/pkg/crux)  Provides 
error handling with call stack, the confab interface to the flocking protocol, 
and the local service interfaces for `reeve` signing and node registry. 

[`pkg/flock`](https://github.com/erixzone/crux/tree/master/pkg/flock) Provides 
the UDP based flocking protocol with encryption and key rotation facilities.

[`pkg/grpcsig`](https://github.com/erixzone/crux/tree/master/pkg/grpcsig) Provides 
the implementation of [`http-signatues`](https://tools.ietf.org/html/draft-cavage-http-signatures-10#section-3.1) 
for gRPC client/server calls, as well as the database (BoltDB) handling local 
public keys and endpoints.

[`pkg/idutils`](https://github.com/erixzone/crux/tree/master/pkg/idutils) Provides 
the definitions and parsers for the `NodeID`, `NetID` and `KeyID`, which are tuples 
of identifiers used by services connecting to `reeve` and by `steward`.

[`pkg/muck`](https://github.com/erixzone/crux/tree/master/pkg/muck) Provides 
local data storage area for `crux` services and for persistent data, as well as key storage. 

[`pkg/ruck`](https://github.com/erixzone/crux/tree/master/pkg/ruck) Package 
providing the bootstrap code for top-level executables `fulcrum` and `ripstop`. 

[`pkg/walrus`](https://github.com/erixzone/crux/tree/master/pkg/walrus) and 
[`pkg/clog`](https://github.com/erixzone/crux/tree/master/pkg/clog)  A fork of Logrus 
adding tagged logging, a ring buffer and a go-kit style loggin api.  
`pkg/clog` provides the interface to `pgk/walrus`.


