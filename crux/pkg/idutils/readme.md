# `pkg/idutils` 
## Tools for `client` and `endpoint` name tuples 

Within `crux` naming conventions for services are set up to provide 
for the automated distribution of public keys and endpoint addresses.  

This package provides helper functions and parsers for the conversion 
of string forms to struct forms.

## The tuples.

There are 3 tuples used to identify service components (`client` and `endpoint`) 
in a crux cluster:  
`NodeID`, `KeyID` and `NetID`.

## Crux Endpoints

Endpoints are identified with a `NodeID` (sometimes abbreviated `fid` 
in the source code) and a `NetID` (often `nid` in source code) as a pair.  

Each is a tuple with a struct form and a string form. 

Each `endpoint` is uniquely identified through this pair of tuples,  
i.e. a `NodeID` and a `NetID`.  
Think of this pair as being a row of multiple columns in a relational database table.

The Go types are:

`NodeIDT`  the T is for "NodeID Type"

`````
type NodeIDT struct {
	BlocName   string `json:"blocname"`
	HordeName   string `json:"hordename"`
	NodeName    string `json:"nodename"`
	ServiceName string `json:"servicename"`
	ServiceAPI  string `json:"serviceapi"`
}
`````


`NetIDT` - the T is for "NetID Type"

`````
type NetIDT struct {
	ServiceRev string `json:"servicerev"`
	Principal  string `json:"principal"`
	Host       string `json:"host"`  // IPV4 or IPV6 host portion of address - see grpc.Dial()
	Port       string `json:"port"`  // port part of the address
}

`````

## Crux Clients

Clients are identified with a `NodeID` (sometimes abbreviated `fid` in the source code)  
and a `KeyID` (often `kid` in source code) as a pair.  

Each is a tuple with a struct form and a string form. 

Each `client` is identified with a pair of tuples, i.e. a `NodeID` and a `KeyID`.  
 
Think of this pair as being a row in a relational database table.

The Go types for these are:

`NodeIDT` the T is for "NodeID Type"

`````
type NodeIDT struct {
	BlocName   string `json:"blocname"`   // The bloc name
	HordeName   string `json:"hordename"`   // The Horde name
	NodeName    string `json:"nodename"`    // The DNS resolvable node Hostname
	ServiceName string `json:"servicename"` // See /crux/naming.md
	ServiceAPI  string `json:"serviceapi"`  // See /crux/naming.md
}
`````

`KeyIDT`  the T is for "KeyID Type"

`````
type KeyIDT struct {
	ServiceRev  string `json:"servicerev"`  // See /crux/naming.md
	Principal   string `json:"principal"`   // The unique identifier of the local reeve
	Fingerprint string `json:"fingerprint"` // The : delimited md5 fingerprint of your public key
}
`````

##  Tuples and String Forms

You can see that NodeIDs are used to capture common components 
of both `client` and `endpoint` information. 

### Making `endpoint` NodeID + NetID tuples:

For an `endpoint` A NodeID is obtained like so:

`````
// blocName, hordeName, and nodeName are passed to you from the service startup arguments.

barName := "bar"
barAPI := "bar_1"
barNodeID, _ := idutils.NewNodeID(blocName, hordeName, nodeName, barName, barAPI)
`````

The `endpoint` NetID is obtained like so for port `51010`:

`````
barRev := "bar_1_0"
principal, _ := muck.Principal()
barNetID, _ := idutils.NewNetID(barRev, principal, nodeName, 51010)
`````
#### Samples from Crux Logs

On crux gRPC endpoint (server-side) startup - with `http-signatures` 
enabled - we use this convention in the logs to show the `NodeID` 
and `NetID` contents in their string forms on a single line:

`level=info INFO="/flock/jets/f9/bar/bar_1 Serving /bar_1_0/3OLU3ZM5FH1IJXYYZ476H7/net/f9:51010" focus="bar_1_0" mode=grpc-signatures `


### Making `client` NodeID + KeyID tuples:

For a `client` NodeID, same as above but with the client names.

`````
fooName := "foo"
fooAPI := "foo1"
fooNodeID, _ := idutils.NewNodeID(blocName, hordeName, nodeName, fooName, fooAPI)
`````

Your `client` KeyID, is provided by `pkg/reeve` which generates the key material.

`````
barRev := "bar_1_0"  // foo is a client of bar_1_0

barSignerIf, _ := reeveapi.ClientSigner(barRev)
fooKeyID, fooKeyJSON := reeveapi.PubKeysFromSigner(barSignerIf)
`````


For a `client` the NodeID carries the `ServiceName` of the `client`, 
the KeyID carries the `ServiceName` of the intended `endpoint`.  
So this pair of tuples encodes the from -> to information of the 
connecting pair `client` to `endpoint`. 

#### Samples From Crux Logs

On server-side logging, we use `NodeID` and `KeyID` in their string 
forms to identify a client on `reeve` and `steward` operations, e.g. 

`level=info msg="Catalog sent to /flock/jets/f9/foo/foo1 /bar_1_0/3OLU3ZM5FH1IJXYYZ476H7/keys/af:80:33:c5:6d:bd:6f:99:d2:17:47:a4:76:01:29:69" PID=9 TS="2019-01-16T19:12:35.93Z" focus=reeve0.1 mode=grpc-signatures `

Or just the `KeyID` in string form. (The client node can be found 
by looking for the principal - which is the unique identifier for its reeve)

On the `bar` server side, the gRPC listeners (via `pkg/grpcsig`) 
automatically log the `KeyID` on `http-signatures` authentication like so:

`level=info msg="grpcsig client authenticated: /bar_1_0/B6TKD80RURJRP9MXKAV7G0/keys/b8:16:e2:f5:5e:dd:6d:8c:21:aa:d2:7a:63:35:a2:c2" PID=9 TS="2019-01-16T19:12:35.46Z" focus="bar_1_0" mode=grpc-signatures `

A failed authentication may look like this on the server side, 
in this case when the calling gRPC client does not appear on the whitelist:

`level=warning msg="GetPublicKeyString failed: unauthenticated - server cannot find public key with provided http-signatures header keyId '/reeve0.1/PWUAC54NMA3262KCD7WQFL/keys/d0:8c:01:f7:46:71:07:13:49:17:ec:56:2b:52:ec:3b' : PubKeysDBLookup grpcsig BoltDB item not found : no match for KeyID /reeve0.1/PWUAC54NMA3262KCD7WQFL/keys/d0:8c:01:f7:46:71:07:13:49:17:ec:56:2b:52:ec:3b" PID=9 TS="2019-01-16T19:11:59.63Z" focus=reeve0.1 mode=grpc-signatures`


## NodeID String Form

The string form is used to pass the NodeID in gRPC function calls 
to `pkg/reeve` when registering clients and endpoints.

The string form of a NodeID is obtained like so:

`fmt.Printf("%s\n", barNodeID.String())`

which might print this on the node `f3`:

`/flock/sharks/f3/bar/bar_1`
 
The string form is delimited with "/" and follows directory naming rules.

The string form of a NodeID is parsed with:

`fidstring, err := idutils.NodeIDParse(fid)`

## NetID String Form

The string form is used to pass the NetID in gRPC function calls 
to `pkg/reeve` when registering endpoints.

The string form of a NetID is obtained like so:

`fmt.Printf("%s\n", barNetID.String())`

which might print this on node `f4`:

`/bar_1_0/3OLU3ZM5FH1IJXYYZ476H7/net/f4:51010`

The string form is delimited with `/` and follows directory naming rules.  
The substring `/net/` is a required delimiter between the `principal` and `address`.

The string form of a NetID is parsed with:

`nidstring, err := idutils.NetIDParse(nid)`

The address substring can be obtained with:

`nid.Address()`

The port (as an `int`) can be obtained with:

`port := idutils.SplitPort(nid.Address())`

The host name can be obtained with:
`host := idutils.SplitHost(nid.Address())` 

## KeyID String Form

As stated above, KeyIDs are issued by `reeve` when client key pairs are 
generated. 

In addition, the `KeyID` string form is used as the key in the 
whitlelist database, which is a key-value store holding the JSON 
form of the public key of a client.

The string form is used to pass the KeyID in gRPC function calls 
to `pkg/reeve` when registering clients. Reeve also needs the 
`fooKeyJSON` - the JSON form of the public key, as provided above in:

`fooKeyID, fooKeyJSON := reeveapi.PubKeysFromSigner(barSignerIf)`

The key pair issuing `reeve` is the `principal` - a identifiable
unique, nonhuman, crux service which makes and maintains key pairs. 

The string form of a KeyID is obtained like so:

`fmt.Printf("%s\n", fooKeyID.String())`

which might look like this this on the node where `reeve` 
is assigned the `principal` as `"VR7OS8Q34OHUMVYR3OS152"`:

`/bar_1_0/VR7OS8Q34OHUMVYR3OS152/keys/2c:5e:af:01:87:3f:2e:fe:76:7a:04:6b:0e:c1:08:6b`
 
The principal is an NUID from the folks at NATS.io, is generated 
on first run on a node, and persists across node restarts.
 
The string form is delimited with `/` and follows directory naming rules.  
The substring `/keys/` is a required delimiter between the 
`principal` and `fingerprint`.

The string form of a KeyID is parsed with:

`fidstring, err := idutils.KeyIDParse(fid)`


## uuids for `client` and `endpoint` 

An appropriate sized hash of the concatenated string form of 
a `NodeID` + `NetID` will be unique within the cluster for each 
endpoint, and can be used to generate a `uuid` for the endpoint.

	endpointuuid = uuid.NewMD5(uuid.NIL, []byte(fid.String() + nid.String())).String()
	
Likewise the hash of concatenated `NodeID` + `KeyID` string forms 
can be used to generate a `uuid` for a client.	
	
	clientuuid = uuid.NewMD5(uuid.NIL, []bytes(fid.String() + kid.String())).String()
