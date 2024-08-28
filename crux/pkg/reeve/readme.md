# Crux - The Reeve Service `pkg/reeve` 
Reeve is the main service manager running on every node.  

Reeve is the first package to start in `fulcrum` and `ripstop` after a flocking 
leader is elected. Reeve initializes/reloads local storage `pkg/muck` and manages 
keypairs and key generation via `ssh-keygen`.  It manages client-side public-key 
signing by `ssh-agent` for signed gRPC communcation via `pkg/grpcsig`, and provides 
the interface for endpoints of services to validate signed gRPC API calls against 
public-keys in the local whitelist database.

Reeve provides: 

1. Node Registration via `pkg/register` and its `registry` server running on the  leader.
2. Node Local assymetric key pair managment, client gRPC call signing (via 
`ssh-agent`) and service gRPC signature validation via `pkg/grpcsig`.
3. Centralized public key and service endpoint address distribution via the `steward` 
server, running on the leader.  
3. The communications agent (`ingest.go`) that recieves node local service client and 
endpoint updates and pushes them out to `steward`.  
4. The gRPC listening agent, recieving push-updates from `steward` of whitelisted
public keys and available service endpoints. 

## Code Contents

#### Reeve service and client interface protobuf file  
[`protos/srv_reeve.proto`](https://github.com/erixzone/crux/blob/master/protos/srv_reeve.proto)  

#### Generated code for gRPC client/server API  
[`gen/cruxgen/srv_reeve.pb.go`](https://github.com/erixzone/crux/blob/master/gen/cruxgen/srv_reeve.pb.go) 

#### Version numbers for `Reeve` and the compatible `Steward`  
`revision.go`  

#### Reeve Startup and ReeveAPI  
`agentstart.go`: Initializes or reloads node persistant information in `.muck`,  
starts `pkg/grpcsig` inter-process self-keys,  
checks `ssh-agent` is reachable for signing,  
starts the whitelist database,  
starts `ReeveAPI`,  
starts `reeve` gRPC endpoint,  
and provides the code to start the Communications Agent in `StartStewardIO()`.   

#### Key Management    
`keycycle.go`: Code to manage keys in `.muck` with `pkg/grpcsig` utilities. 

#### Reeve client and server code for Node Registration  
`cliutil.go`: Client used in `pkg/steward` for pushing updates to `reeve` and client  
used in `pkg/register` for 2nd channel `reeve` callback.  
`srvutil.go`: Server code providing reverse `http-signatures` callback endpoint.

#### Communications Agent (to `steward`)
`ingest.go`: Event loop that pushes updates to `steward`  
 `clientstore.go`: Checkpointed storage of local node client information, and update status  
 `endpointstore.go`: Checkpointed storage of local node endpoint information, and update status.  

#### Reeve Server (from local services; from `steward`)  
`usercalls.go`: gRPC server handlers for user calls: `RegisterEndpoint()`, `RegisterClient()`, 
`EndpointsUp()`, `Catalog()`  

`stewardcalls.go`: gRPC server handlers for `steward` calls: `WlUpdate()`, `EpUpdate()`, 
`WlState()`, `EpState()`, `UpdateCatalog()`  

#### Test  
`reeve_test.go`  
 
 <hr>

## Reeve APIs. Local and gRPC

ReeveAPI Reeve has a local API (function calls in the same process - see 
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

## Using Reeve

Reeve startup is currently shown in the example code in `pkg/ruck/bootstrap.go` as 
`BootstrapRipstop()`. Assume a flock is started, `reeve` is started on all nodes, and 
the `steward`, and `register` services are started on the leader. You can now start up 
a new `crux` service or client. 

A sample client `foo` and service `bar` is provided in `pkg/sample`

### A simple Client 

Goal - Make key pair, gRPC signer, and distribute public keys, get service Catalog 
and Endpoints available for client to connect to.

SO. You need a secured gRPC client `foo` for `crux` service `bar`, revision `bar_0_1`; 
and you need all the nodes running `bar` in the horde `foobar` to automatically whitelist 
your client. 

See `pkg/sample/fooclient.go`.  This demo can be hooked up in an integration test in `ripstop`
by exposing these commented lines in `pkg/ruck/bootstrap.go`.

      /*
      ...
      fooerr := sample.FooClientExercise(reeveapi, "flock", horde, me)
      if fooerr != nil {
            logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - FooClientExercise failed : %v", fooerr))
            os.Exit(1)
      }
      */


<hr>
A) Construct a `NodeIDT` with `pkg/idutils` `NewNodeID()` for your client `foo`:  

		fooName := "foo"
		fooAPI := "foo1"
		barRev := "bar_1_0"
		fooNodeID, _ := idutils.NewNodeID(blocName, hordeName, nodeName, fooName, fooAPI)

The `NodeID` packages the string variables `BlocName/HoardName/NodeName/ServiceName/ServiceAPI`.  
If NodeName is `f4` here `fooNodeID` in string form is `"/flock/foobar/f4/foo/foo_1"`  
This will be paired with a `KeyID` in (C) to uniquely identify your client in the cluster.  
  
B) Obtain a `ClientSignerT` interface for service `bar` via:  

`barSignerIf, _  := reeveapi.ClientSigner(barRev)`.  

The argument is the `ServiceRev` string of the service to connect to.  
Here we assume `reeveapi` is started up already. `ClientSigner()` makes the local 
keypair (or loads if already established), and returns a signer interface which 
includes the `KeyID`, the public key, and the access method to `ssh-agent` via 
UNIX pipe, which does the actual signing with the private key.
	
C) Get the `KeyID` and JSON public key from your `barSigner` made in (B):

`fooKeyID, fooKeyJSON := reeveapi.PubKeysFromSigner(barSignerIf)`


This extracts the database lookup `KeyID` and the JSON form of the public key from 
this signer.  
These are sent to `reeve` in the next steps...
	
D) Make a `pb.ClientInfo{}` gRPC struct from the results from (A) and (C).

	fooCI := pb.ClientInfo{
		Nodeid:  fooNodeID.String(),
		Keyid:   fooKeyID,
		Keyjson: fooKeyJSON,
		Status:  pb.KeyStatus_CURRENT,
	}
This packages up the `NodeID`, `KeyID` and JSON form of the public key for 
sending to `reeve` via gRPC.
	 
E) Obtain a local signer for contacting `reeve` via gRPC.

`selfsign := reeveapi.SelfSigner()` 

This gets an inter-process gRPC signer so you can communicate to your local node's 
own `http-signatures` secured `reeve` without needing any public key distribution. 
	
F) Dial up a gRPC client to your own node's `reeve` 

		logfoo := clog.Log.With("focus", "foo_client", "node", nodeName)
		_, reevenetid, _, _, _ := reeveapi.ReeveCallBackInfo() 
		reeveNID, _ := idutils.NetIDParse(reevenetid)
		reeveclient, rerr := reeve.OpenGrpcReeveClient(reeveNID, selfsign, logfoo)

This starts a `foo` client logger,  
then gets your `reeve` NetID (containing its network address and port),  
then uses the signer from (D) to Dial the gRPC client to `reeve`. 

G) Register the `foo` client across the cluster by sending the `fooCI` struct 
from (D) to `reeve` with `RegisterClient()`:

	ack, cerr := reeveclient.RegisterClient(context.Background(), &fooCI)
	if cerr != nil {
		msg := fmt.Sprintf("error - RegisterClient failed for foo: %v", cerr)
		logfoo.Log("error", msg)
		return crux.ErrS(msg)
	}
	logfoo.Log("info", fmt.Sprintf("foo client is registered with reeve: %v", ack))
	

This sends the `fooCI` to `reeve`.  
In turn `reeve` logs this client information in a local store, then forwards 
the information to `steward`.  
`Steward` stores this information in its tracking database.  
`Steward` then applies the rules table, and distributes 
your `KeyID` and JSON public key data to nodes in horde `foobar`, 
running service `bar`, to which your `foo` client is allowed to connect.

  
  
<hr>
### Client gRPC Resources - `Catalog()` and `EndpointsUp()`

#### `Catalog()`

Once your client is registered, and the `steward` has distributed your public keys, 
you can request a list of services your client can access.  

To call the `Catalog()` gRPC function, you need from above:
`fooNodeID`, `fooKeyID`, the logger `logfoo`, and the gRPC client `reeveclient`.

First, make a `pb.CatalogRequest` 

	catrequest := pb.CatalogRequest{
		Nodeid: fooNodeID.String(),
		Keyid:   fooKeyID}
		

Then you can use `reeveclient`. Here we are printing the catalog to `logfoo`.

	fooCatalog, aerr := reeveclient.Catalog(context.Background(), &catrequest)
	if aerr != nil {
		msg :=  fmt.Sprintf("error - Catalog failed for foo: %v", aerr)
		logfoo.Log("node", nodeName, "ERROR", msg)
		return crux.ErrS(msg)
	}
	if fooCatalog != nil {
		logfoo.Log("node", nodeName, "INFO", fmt.Sprintf("Catalog result: %v", fooCatalog))
	}


Note that you may need to wait for the entire catalog to emerge as `steward` will 
need some time to distribute its information.  

The essentials of the `Catalog()` gRPC return type `pb.Catalog` reply are as follows. 

A `List` is a slice of pointer to `CatalogInfo` which contain the `NodeID` and `NetID` 
pairs for 1 of each kind of service you can connect to. You can assume the `NetID` is an 
active endpoint for the service named in `FockID`. To see all of the endpoints in the 
cluster, use `EndpointsUp()`.

````
type CatalogReply struct {
        List                 []*CatalogInfo 
        Error                string         
}

type CatalogInfo struct {
        Nodeid              string   
        Netid                string   
        Filename             string   
}

````


#### `EndpointsUp()`

Once your client is registered, and the `steward` has distributed your public keys, 
you can request a list of endpoints your client can access.  

To call the `reeve` EndpointsUp() gRPC function, you need from above:
`fooNodeID`, `fooKeyID`, the logger `logfoo`, and the gRPC client `reeveclient`.


First, make a `pb.EndpointRequest` 

	eprequest := pb.CatalogRequest{
		Nodeid: fooNodeID.String(),
		Keyid:   fooKeyID,
		Limit: 0}

Here the `Limit` argument set to `0` means no limit to the number of endpoints 
returned. `Limit` set `> 0` caps the number of endpoints returned to the integer 
value provided.

Now you can use `reeveclient` to call `EndpointsUp`.

		fooEndpoints, uerr := reeveclient.EndpointsUp(context.Background(), &eprequest)
        if uerr != nil {
                msg := fmt.Sprintf("error - EndpointsUp failed for foo: %v", uerr)
                logfoo.Log("node", nodeName, "ERROR", msg)
                return crux.ErrS(msg)
        }

The example `foo` client code finishes by pinging each `bar` server on the list of endpoints:

        if fooEndpoints != nil {
                logfoo.Log("node", nodeName, "INFO", fmt.Sprintf("EndpointsUp result: %v", fooEndpoints))
                for _, fooEp := range fooEndpoints.List {
                        fooEpNetID, _ := idutils.NetIDParse(fooEp.Netid)
                        // Ping Each Bar server - returns on timeout,  error or success
                        werr := WakeUpBar(barAgentSigner, fooEpNetID)
                        if werr != nil {
                                msg := fmt.Sprintf("error - foo WakeUpBar failed on endpoint %s: %v", fooEpNetID.St$
                                logfoo.Log("node", nodeName, "ERROR", msg)
                        }
                        logfoo.Log("node", nodeName, "INFO", fmt.Sprintf("foo endpoint %s is awake :)", fooEpNetID.$
                }
        }


The essentials of the `EndpointsUp()` gRPC return type `pb.Endpoints` reply are as follows.
  
A `List` is a slice of pointer to `EpInfo` which carry the `NodeID` and 
`NetID` pairs for 1 of each kind of service you can connect to. 

````
type Endpoints struct {
        List                 []*EpInfo
        Error                string
}

type EpInfo struct {
        Nodeid              string
        Netid                string
        Priority             string // Unused - not set
        Rank                 int32  // Unused - not set
}
````


### A simple Server Endpoint 
Goal - Enable `crux` http-signatures validation with the crux whitelist of 
allowed public keys; and distribute this gRPC endpoint information to eligible clients.

SO. You will run the gRPC service `bar`,  api `bar_0`, revision `bar_0_1` on 
port `51010` and you need all the nodes running `bar` in the horde `foobar` to 
automatically apply `http-signatures` validation to all inbound `gRPC` clients, 
with automatic updates of the whitelist of allowed clients.

See `pkg/sample/barserver.go`.  This demo can be hooked up in an integration test in `ripstop`
by exposing these commented lines in `pkg/ruck/bootstrap.go`.

		/*
       barerr := sample.BarServiceStart(reeveapi, "flock", horde, me)
       if barerr != nil {
               logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - BarServiceStart failed : %v", barerr))
               os.Exit(1)
       }
		*/

<hr>
A) Construct a `NodeIDT` with `pkg/idutils` `NewNodeID()` for your server endpoint `bar`:  

        barName := "bar"
        barAPI  := "bar_1"
        barRev  := "bar_1_0"
        barNodeID, _ := idutils.NewNodeID(blocName, hordeName, nodeName, barName, barAPI)

The `NodeID` packages the string variables `BlocName/HoardName/NodeName/ServiceName/ServiceAPI`.  
If NodeName is `f4` here `barNodeID` in string form is `"/flock/foobar/f4/bar/bar_1"`  
This will be paired with a `NetID` in (B) to uniquely identify your client in the cluster. 

B) Construct a `NetIDT` with `pkg/idutils` `NewNetID()` for your endpoint `bar_1_0`. 
The `principal` is a unique ID assigned to the node and stored in `.muck`. `NodeName` is 
the DNS lookup string or IP address, and `51010` is the host.

        principal, _ := muck.Principal()
        barNetID, _ := idutils.NewNetID(barRev, principal, nodeName, 51010)
        
C) Obtain a `grpcsig` `ImplementationT` interface for service bar via:

        logbar := clog.Log.With("focus", "bar_service", "node", nodeName)
        barImp := reeveapi.SecureService(barRev)
        if barImp == nil {
                msg := "failed reeveapi.SecureService for bar"
                logbar.Log("node", nodeName, "error", msg)
                return crux.ErrS(msg)
        }


The argument of `reeveapi.SecureService` is the `ServiceRev` string of the service 
to connect to. Here we assume `reeveapi` is started up already. `SecureService()` 
provides a interface with a handle to the whitelist database and its lookup methods 
for finding public keys by client `KeyID`. 


D) You need a startup routine like this `SimpleStart` for the gRPC service `bar` 
in package `bar`, which uses the required gRPC interceptors defined in `pkg/grpcsig`.


````
// SimpleStart - starts server Bar
func SimpleStart(nod idutils.NodeIDT, nid idutils.NetIDT, impif interface{}) *crux.Err {
        // Cast the impif interface
        ptrimp, ok := impif.(**grpcsig.ImplementationT)
        if !ok {
                return crux.ErrF("bad interface{} passed to EasyStart, not a **grpcsig.ImplementationT")
        }
        imp := *ptrimp

        // Start gRPC server with Interceptors for http-signatures inbound
        s := grpc.NewServer(grpc.UnaryInterceptor(grpcsig.UnaryServerInterceptor(*imp)),
                grpc.StreamInterceptor(grpcsig.StreamServerInterceptor(*imp)))

        // Use your Protobuf Register function for server bar
        pb.RegisterBarServer(s, &server{})

        // Listen on the specified port
        lis, err := net.Listen("tcp", nid.Port)
        msg := ""
        if err != nil {
                msg = fmt.Sprintf("error - bar net.Listen failed: %v", err)
                return crux.ErrS(msg)
        }

       // Print and/or log the serving message with the full NodeID and NetID
       msg = fmt.Sprintf("%s Serving %s", nod.String(), nid.String())
       fmt.Printf("%s\n", msg)
       imp.Logger.Log("INFO", msg) // imp carries a Logger, so use it.
       go s.Serve(lis)
       return nil
}

````
E) Call your gRPC service startup with `barNodeID`, `barNetID`, and `barImp` from (A-C)


        err := SimpleStart(barNodeID, barNetID, barImp)
        if err != nil {
                msg := fmt.Sprintf("error - SimpleStart failed for bar: %v", err)
                logbar.Log("error", msg)
                return crux.ErrS(msg)
        }

You should see the Serving message.


F) Make a `pb.EndpointInfo{}` gRPC struct from the results from (A) and (B).

        ts := time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
        barEI := pb.EndpointInfo{
                Tscreated: ts,
                Tsmessage: ts,
                Status:    pb.ServiceState_UP,
                Nodeid:    barNodeID.String(),
                Netid:     barNetID.String()}
    
    
This packages up the NodeID, and NetID for sending to reeve via gRPC.

G) Obtain a local signer for contacting `reeve` via gRPC.

`selfsign := reeveapi.SelfSigner()` 
	
H) Dial up a gRPC client to your own node's `reeve` 

		 _, reevenetid, _, _, _ := reeveapi.ReeveCallBackInfo()
        reeveNID, _ := idutils.NetIDParse(reevenetid)
        reeveclient, gerr := reeve.OpenGrpcReeveClient(reeveNID, selfsign, logbar)

This gets your `reeve` NetID (containing its network address and port),  
then uses the signer from (G) to Dial the gRPC client to `reeve`. 

I) Register the `bar` endpoint across the cluster by sending the `barEI` struct 
from (F) to `reeve` with `RegisterEndpoint()`:

        ack, rerr := reeveclient.RegisterEndpoint(context.Background(), &barEI)
        if rerr != nil {
                msg := fmt.Sprintf("error - RegisterEndpoint failed for bar: %v", rerr)
                logbar.Log("error", msg)
                return crux.ErrS(msg)
        }
        logbar.Log("info", fmt.Sprintf("bar server is registered with reeve: %v", ack))
	

This sends the `barEI` to `reeve`.  
In turn `reeve` logs this endpoint information in a local store, then forwards 
the information to `steward`.  
`Steward` stores this information in its tracking database.  
`Steward` then applies the rules table, and distributes 
your `bar` service identifier pair (`barNodeID`,`barNetID`) data to nodes in 
horde `foobar`, running client `foo`, which are allowed to connect to your `bar` server endpoint.


<hr>

### Endpoint Resources - gRPC Signatures Validation

This is automatically provided. `reeve` maintains the whitelist, and provides the interface 
to do the whitelist database lookups via `reeveapi.SecureService()`.  All the gRPC Server 
startup code needs to pass in the Interceptors.


<hr>


# TODO

# reeve_test.go - reverse-http-signature public key exchange


The reverse-http-signature mechanism is tested here.

Tests Factor 2 of authentication mechanism for registering nodes (fulcrums with reeve servers) where:

Factor 1: flock encryption key (symmetric key, shared secret over flocking protocol) 

Factor 2: reeve() reverse-http-signature service (asymmetric key, non-shared secret)

The test here uses the grpcsig self-key so that the test reeve client 
(i.e. that would go in register() service part of Steward()) can operate in the 
same runtime as the test reeve() server.

The handshake model for reeve{} - register() is:

A fulcrum's calls register() with its flock key encrypted (Factor 1) - 
public key and netId of its reeve() service.
The register() service decrypts the node info, then does a gRPC callback (Factor 2) to 
get the rest of the node's service and key information. 

That register() instance will use the reeve client code `reeve/pkg/cliutils.go` 
(under test here) to receive an update bundle of the fulcrum's services and public keys.

For the sake of the test for the reverse-http-signature mechanism, we bypass the 
steward database bits for now.



