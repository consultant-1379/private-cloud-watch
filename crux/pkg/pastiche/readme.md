#Pastiche Architecture
Unlike many grpc based services, pastiche has a **Client** type that wraps the generated grpc client, hiding its rpc nature and handling the message generation and response decoding.

The **Blobstore** type handles most of the local storage and caching logic, while the **Server** type wraps the Blobstore and uses other pastiche servers on the network for operations requiring  peer communications

#Running the pastiche demo 

In the crux dir, do a 'make build' to get pastiche demo binaries built and installed.

In pkg/pastiche,  run multi-server-demo.sh
Check client.log for results