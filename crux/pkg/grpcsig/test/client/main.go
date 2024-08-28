// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/pborman/uuid"
	"golang.org/x/net/context"

	"github.com/erixzone/crux/pkg/grpcsig"
	pb "github.com/erixzone/crux/pkg/grpcsig/test/gen"
)

const (
	address = "localhost:50052"
)

/* To get this to work, see grpcsig/readme.md

Uses environment variables GRPCSIG_FINGERPRINT and GRPCSIG_USER

check SSH_AUTH_SOCK with:
$ ssh-add -l
The agent has no identities.

if so, do this:
$ ssh-add -K ~/.ssh/id_rsa
Identity added: /Users/yourname/.ssh/id_rsa ({~}/.ssh/id_rsa)

then check again
$ ssh-add -l
*/

func main() {

	addy := os.Getenv("GRPCSIGCLI_DIAL")
	if addy == "" {
		addy = address
	}

	//-----------------
	// GRPC http-signatures - typical client-side

	// Initialize the ssh-agent signer from the environment variables
	// SSH_AUTH_SOCK, GRPCSIG_FINGERPRINT and GRPCSIG_USER
	// service := "phlogiston"
	service := "jettison" // Name of gRPC service we will call - must match server side
	agentSigner, err := grpcsig.DefaultEnvSigner(service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal - ssh-agent initialization failed: %v\n", err)
		os.Exit(1)
	}

	// Set up a connection to the gRPC server, chaining in the grpcsig Interceptors
	// using the grpc_middleware methods. note that WithInsecure() refers to https
	// transport security (i.e. TLS is off for this demo)
	conn, err := agentSigner.Dial(addy)
	defer conn.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Did not connect to server: %v\n", err)
		os.Exit(1)
	}
	//-----------------

	// Communicate to server with Unary and Stream gRPC.

	// Creates a new SigtestClient to put some data on the server (Stream)
	client := pb.NewSigtestClient(conn)
	sigtest := &pb.SigtestRequest{
		Id:   1,
		Name: "Important Data 1",
		Sigstreams: []*pb.SigtestRequest_SigStream{
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
		},
	}

	// Create a new sigtest - put some more stuff on the server (Stream)
	createSigtest(client, sigtest)
	sigtest = &pb.SigtestRequest{
		Id:   2,
		Name: "Important Data 2",
		Sigstreams: []*pb.SigtestRequest_SigStream{
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
			&pb.SigtestRequest_SigStream{
				Uid: uuid.NewUUID().String(),
			},
		},
	}

	// Create a new sigtest - query the server for all its data (Unary)
	createSigtest(client, sigtest)
	// Get All from the server
	all := &pb.SigtestAll{IsAll: true}
	getSigtests(client, all)
}

// Two gRPC methods are used to test the Unary and Stream Interceptors

// createSigtest - calls the RPC method CreateSigtest of SigtestServer
func createSigtest(client pb.SigtestClient, sigtest *pb.SigtestRequest) {
	resp, err := client.CreateSigtest(context.Background(), sigtest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create Sigtest on Server: %v\n", err)
		os.Exit(1)
	}
	if resp.Success {
		fmt.Fprintf(os.Stdout, "New Sigtest %d added to Server.\n", resp.Id)
	}
}

// getSigtests - calls the RPC method GetSigtests of SigtestServer
func getSigtests(client pb.SigtestClient, all *pb.SigtestAll) {
	// streaming
	stream, err := client.GetSigtests(context.Background(), all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot get Sigtest List from Server: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Server Sigtest List:\n")
	for {
		sigtest, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Bad Sigtest Stream\n client:%v\n err:%v\n", client, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "%v\n", sigtest)
	}
}
