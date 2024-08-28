# Crux Service Naming Conventions

Within `crux` naming conventions for services are set up to provide 
for the automated distribution of public keys and endpoint addresses.  

To facilitate this automation, naming conventions also extend to rules 
specified to control the distribution of client and endpoint 
information around the cluster. 

Conventions stated here provide placeholder and minimal support for  
versioning schemes, which may be expanded later.

These naming conventions and distribution rules are addressed in this 
document.  More details about service naming can be found 
in [`pkg/idutils`](https://github.com/erixzone/crux/tree/master/pkg/idutils/readme.md). 

## Definitions and Terms

A `client` is a component that initiates gRPC calls to an `endpoint` 
via `grpc.Dial`.

An `endpoint` is a component that listens for gRPC communications 
via `grpc.NewServer` on a port started with `net.Listen`. 

A "service" is a general term, understood in terms of pairs of 
`clients` and `endpoints`.  Your service may be a `client` of more 
than one type of `endpoint`. Your service may be one or more `endpoint` 
listeners on different ports, and have no client. Your service may 
just involve a `client`, and do some other task, like display logs 
to a console.

In the `http-signatures` layer 7 security scheme used in crux, 
a `client` must sign gRPC calls with a private key.  So a client 
needs a key-pair, a signer and a `KeyID` which is a tuple-based 
key-value store identifier. The `endpoint` finds the `client` public 
key in its whitelist database via this `KeyID`.  Of course, the `endpoint` 
being called must have a copy of the client's public key in its whitelist.

Crux provides the key pairs, signers and whitelisting facilities for 
the cluster through `pkg/reeve` and `pkg/steward`.  

When implementing a service, you will access these facilities through `pkg/reeve`.  

The crux gRPC system provides "interceptor" code for adding "http-signatures" 
whitelisting to endpoints. 

For gRPC clients, this covers tools to make key pairs and signers for 
gRPC clients, "interceptors" to insert the signatures into gRPC requests, 
and systems to to automatically distribute public keys across the cluster. 

For gRPC endpoints, this includes the validation interceptors, a 
whitelist database of public keys on every node, and systems to 
automatically distribute the network address of the endpoint to 
eligible clients that arise across the cluster.

These features are accessed through the reeve service, and the documentation 
in [`pkg/reeve/readme.md`](https://github.com/erixzone/crux/tree/master/pkg/reeve/readme.md) 
outlines the steps involved for setting up gRCP clients and endpoints 
with code examples in `pkg/sample`.

Finally, services in `crux` are restricted to access by whitelist.  
Whitelists are automatically populated by `pkg/steward`, and managed on 
each node by `pkg/reeve`.  
"Distribution Rules" specify what `client` and `endpoint` pairs 
are allowed to communicate over gRPC, and control what information
is distributed across the cluster. 


## Naming and Versioning

A crux `endpoint` is named with a convention that attempts to provide 3 slots to 
accomodate your choices for version information. Let's work through some examples 
and see how these choices affect the rules for allowed connections between clients 
and endpoints, and understand how signers are specified.

NOTE: This is a minimalist approach, and will likely be expanded in the future.

There are 3 slots provided for your version information. 
There is no style enforcement as to how you use these 3 slots. 
They are unparsed strings, note that `"/"` characters are not allowed.
 
+ `ServiceName`   Used to match Distribution Rules (see below).
+ `ServiceAPI`    Unused placeholder.
+ `ServiceRev`    Used to match client public key & signer to endpoint.

An `endpoint` is specified with all 3 identifiers.
A bare `client` is specified with `ServiceName` and `ServiceAPI`.  
A client has no `ServiceRev`.  

No attempt is made at parsing these strings.  It is important that you 
understand how these 3 slots are used by the Distribution Rules 
and which to specify for a `client` or an `endpoint`.

E.G. 1. A version of `endpoint` - "bar 1.3.2", the following naming 
choices are made:

+ `"bar_1"` is the `ServiceName` (least specific string) 
+ `"bar_1_3"` is the `ServiceAPI` (API compatibility string)
+ `"bar_1_3_2"` is the `ServiceRev` (most specific string)  

For a client `foo` 

+ `"foo"`  is the `ServiceName`
+ `"foo_1"` is the `ServiceAPI`
There is no `ServiceRev` for `foo`. 

Crux distributes `client` public keys and `endpoint` addresses.  
The system automatically pushes this information out to all eligible 
communicating pairs on nodes via `pkg/steward`. 
The "Distribution Rules" specify what pairs of `client` and `endpoint` 
are allowed to communicate. These rules limit 
what `pkg/steward` will ship as whitelist entries to `pkg/reeve` on 
each node. Distribution Rules are defined in the database portion of `steward`, 
`pkg/registrydb/allowed.go`, and are outlined in the next section.

To specify the distribution rules, the `ServiceName` for `client` and `endpoint` is used.
This line of distribution rule JSON permits all `"foo"` clients to connect with all `"bar_1"` 
endpoints (i.e. `steward` is allowed to distribute data about these pairs to each 
node in a `crux` cluster)  within the `horde` named `"sharks"`:

`{"rule":"5", "horde":"sharks", "from":"foo", "to":"bar_1", "owner":"cruxadmin"}`

When you specify a distribution rule, the `"from"` and `"to"` fields must exactly 
match a client `ServiceName` and an endpoint `ServiceName`.  

If there are multiple versions of "bar" running (bar 1.3.2, bar 1.6.0, bar 1.0.1), 
and they share `ServiceName` `"bar_1"` then any `client` named `"foo"` is eligible
by distribution rule to have its public keys distributed to any `endpoint` running `"bar_1"`,
and if the signature passes, connect to them.

E.G. 2. To further restrict access to "bar 1.3.2" permitting only "bar 1.3" 
services to connect to "foo", these are the choices you make:

+ `"bar_1_3"` is the `ServiceName` (least specific string) 
+ `"bar_1_3"` is the `ServiceAPI` (API compatibility string)
+ `"bar_1_3_2"` is the `ServiceRev` (most specific string)  

For a client `foo` 

+ `"foo"`  is the `ServiceName`
+ `"foo_1"` is the `ServiceAPI`  
There is no `ServiceRev` for `foo`. 
 
When you specify a distribution rule here, again the `"from"` and `"to"` fields must 
exactly match a client `ServiceName` and an endpoint `ServiceName`:
  
`{"rule":"5", "horde":"sharks", "from":"foo", "to":"bar_1_3", "owner":"cruxadmin"}`

So this will allow any client "foo" to have its whitelisting information distributed 
to any endpoint with `ServiceName` `"bar_1_3``.

Again, there is no parsing of these strings, so version expressions 
(like > or <) are not employed. At this stage you cannot specify some range of 
version numbering with > and < and delimited digits. You must add in a 
line into the distribution rules for each version `ServiceRev` used by 
endpoints in the system.

Finally, "allowing" a connection via a distribution rule does not make the connection work. 
You have to use `pkg/reeve` calls to get gRPC keypairs, signers, and to 
distribute your keys through the cluster. The distribution rules limit what `pkg/steward` 
is allowed to distribute back to each node's `pkg/reeve` in the cluster 
- i.e. the whitelisted public keys and the endpoints available to your client.  


## Distribution Rules

The information distributed by crux are `client` public keys and `endpoint` 
network addresses.  
Distribution rules are the rules that the system uses to define eligible communicating 
pairs, and distribute their information accordingly.  

These rules affect whether or not an `endpoint` node recieves a whitelist entry for a `client`;
and whether or not a `client` node recieves a network address for that `endpoint`. 
Distribution rules are not a validation mechanism, instead they regulate cluster-wide dissemination 
of the validatation data. On a whitelist-only system like crux the rules effectively 
specify the pairs of types of `client` and `endpoint` that are allowed to 
communicate. Again, distribution rules do not, by themselves, do any identity validation.

### TLDR; The Important Bits:

+ Distribution rules limit the distribution of public keys and endpoint network addresses across 
the cluster.
+ Distribution rules limit whitelists to eligible gRPC connection pairs in eligible hordes.
+ Distribution rule naming is with `ServiceName` level.
+ You need to specify distribution rules for new services you add to `crux`.
+ Distribution rules are less specific than gRPC signers:
	+ Client gRPC signers are obtained by passing  
	the Endpoint `ServiceRev` to `reeve.ClientSigner()`
	+ Endpoint gRPC validation hooks are obtained by passing   
	the Endpoint `ServiceRev` to  `reeve.SecureService()`
+ You need to specify distribution rules for new hordes you define in a flock.
+ Multi-line rules are used to span multiple hordes.

Distribution rules are found in the database portion of `steward` : 
[`pkg/registrydb/allowed.go`](https://github.com/erixzone/crux/tree/master/pkg/registrydb/)

The distribution rules limit what `pkg/steward` will ship as whitelist entries to `pkg/reeve` 
on each node. Everything not on the whitelist is blocked by default. 
Only the listed distribution rule pairs get distributed on the elgibile horde(s).

The absence of a distribution rule, or failure to string match a rule named pair of 
`client` and `endpoint`, means no information is distributed.  
So - any `client` trying to gRPC call an `endpoint` that is not listed 
in an rule is blocked by default. It never makes it on to the whitelist 
of that endpoint. By default `crux` gRPC services use the `http-signatures` 
in `pkg/grpcsig` , and this requires the public key data for signature 
validation. As a whitelisted system only the ones with the proper private 
key signature, and public key signature validation are accepted.

When you specify a distribution rule, the `"from"` and `"to"` fields must exactly 
match a `client ServiceName` and an `endpoint ServiceName`. 

For example:  

`{"rule":"1", "horde":"sharks", "from":"foo", "to":"bar", "owner":"cruxadmin"}`

This will consider any `client` with `ServiceName="foo"` as eligible to connect 
to any `endpoint` with `ServiceName="bar`, and distribute any information arising 
(again, the public keys and addresses) out to the applicable nodes. 

This process of distribution is triggered when the reeve gRPC `RegisterEndpoint()` 
and `RegisterClient()` calls are made.  The reeve forwards these to the central 
steward service, which distributes them to other nodes. This rule will only 
apply to clients and endpoints running on nodes within the horde named `"sharks"`.  

To allow a client and endpoint to span more than one `horde`, the rule 
is repeated for each `hordename`.  This is how system-wide endpoints like 
steward get their information distributed across the entire flock:

`{"rule":"6", "horde":"sharks", "from":"foo", "to":"baz", "owner":"cruxadmin"}`
`{"rule":"6", "horde":"jets", "from":"foo", "to":"baz", "owner":"cruxadmin"}`

Multi-line rules (with the same rule number) are OR'ed together. 

Each node's reeve also gets has a copy of the distribution rule table, which is 
updated and refreshed from the central steward service. It uses this 
to establish what a client can see in the reeve gRPC `Catalog()` function.  
A client is restricted by reeve to see only the `endpoint` data 
(network addresses, in this case) matching the distribution rule pairs in which 
it is listed (again by `ServiceName`).

So the reeve `Catalog()` gRPC call returns a list of endpoints 
matching the distribution rules that are allowed to connect to the provided client whose 
`ServiceName` matches up.

However - just because an endpoint appears in a `Catalog` listing, does not mean 
you can access it - you need to have the right key material and signer, which 
are specified as `ServiceRev` when the `client` is made through `reeve.ClientSigner()`.

For example, the client `"foo"` above may see both `"bar"` 
and `"baz"` endpoint entries in the reeve `Catalog` call.  

But its signer will specify only one endpoint `ServiceRev` (e.g. `"bar_1_3"`).  
This is the string passed to `reeve.SecureService()` when the client obtains 
the key material and signer.  The client signer provided by this call will 
only work with endpoints of type `"bar"` offering `ServiceRev="bar_1_3"`. 

The reeve `EndpointsUp` gRPC call returns a list of endpoint data 
(network addresses) that the provided client can access with the key material 
it has. These will be nodes in the rule eligible `horde`s running the endpoint 
`ServiceRev="bar_1_3"`.  If the endpoint is of type `bar` running any `ServiceRev`
string not matching `"bar_1_3"`, this client will fail to validate on gRPC connections.
  