package self

// This tests the self-key system with a single process
// starting 2 servers and 2 clients.
// client signatures signed with a self-key can access any
// service this self-same process offers.
// Kind of like an internal GOD mode.
// but mind you real keys/signatures are applied and must pass
// validation.
// For non-self keys, the service name must match explictally
// A service name just bounces users calling the
// wrong servicename, before any signature lookup or matching.

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	pb "github.com/erixzone/crux/pkg/grpcsig/test/gen"
	"github.com/erixzone/crux/pkg/muck"
)

func TestSigtest(t *testing.T) { TestingT(t) }

type SigtestSuite struct {
	dir string
}

func init() {
	Suite(&SigtestSuite{})
}

const (
	port1    = ":50052"
	port2    = ":50053"
	dbname   = "pubkeys.db"
	service1 = "jettison"
	service2 = "phlogiston"
)

var testdb string

func (p *SigtestSuite) SetUpSuite(c *C) {
	p.dir = c.MkDir()
	fmt.Printf("Starting Muck")
	merr := muck.InitMuck(p.dir+"/"+".muck", "")
	fmt.Printf("IntiMuck() Errors?: [%v]\n", merr)
	c.Assert(merr, IsNil)
	fmt.Printf("\nStarting Self Keys\n")

	// Bootstrap some client keys as "self-keys" for this process,
	// with ssh-keygen, and start ssh-agent for signing
	kerr := grpcsig.InitSelfSSHKeys(true)
	fmt.Printf("InitSelfSSHKeys() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// Double check that we can read keys from ssh-agent
	_, kerr = grpcsig.ListKeysFromAgent(true) // Does ssh-agent have it?
	fmt.Printf("ListKeysFromAgent() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// Server uses one whitelist DB per process - for all servers
	// Provide our test public key lookup BoltDB (initally empty) as
	testdb = p.dir + "/" + dbname

	oerr := grpcsig.StartNewPubKeyDB(testdb)
	c.Assert(oerr, IsNil)
	logger := clog.Log.With("focus", "self_test")
	derr := grpcsig.InitPubKeyLookup(testdb, logger)
	c.Assert(derr, IsNil)

	// Add self public key to whitelist DB (i.e. this test talks to itself!)
	kerr = grpcsig.AddPubKeyToDB(grpcsig.GetSelfPubKey())
	fmt.Printf("AddPubKeyToDB() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	grpcsig.FiniPubKeyLookup()

	//------------------
	// START sigtest demo server 1 "jettison"

	fmt.Printf("\nStarting server %s\n", service1)
	var cerr *crux.Err

	httpsigService1, cerr := grpcsig.InitDefaultService(testdb, service1, nil, logger, 300, true)
	if cerr != nil {
		fmt.Fprintf(os.Stderr, "%s\nStack: %s\n", cerr.String(), cerr.Stack)
	}
	c.Assert(cerr, IsNil)

	s1 := httpsigService1.NewServer()
	pb.RegisterSigtestServer(s1, &server{})
	lis1, nerr1 := net.Listen("tcp", port1)
	if nerr1 != nil {
		fmt.Fprintf(os.Stderr, "Error -  net.Listen 1 failed: %v", nerr1)
	}
	c.Assert(nerr1, IsNil)

	// Fire up that service
	go s1.Serve(lis1)
	fmt.Printf("Serving '%s' on port%s\n", service1, port1)

	// Pause for handy MacOS X firewall dialogue to appear & clicky-clicky
	if runtime.GOOS == "darwin" {
		time.Sleep(4 * time.Second)
	} else {
		time.Sleep(4 * time.Millisecond)
	}

	//------------------
	// START sigtest server 2 "phlogiston"

	fmt.Printf("Starting server %s\n", service2)

	// Note here 2nd server gets same parameters (... os.Stdout, 300) as first when
	// passed (... nil, 0)
	httpsigService2, cerr := grpcsig.AddAnotherService(httpsigService1, service2, nil, 0)
	if cerr != nil {
		fmt.Fprintf(os.Stderr, "%s\nStack: %s\n", cerr.String(), cerr.Stack)
	}
	c.Assert(cerr, IsNil)

	s2 := httpsigService2.NewServer()
	pb.RegisterSigtestServer(s2, &server{})

	lis2, nerr2 := net.Listen("tcp", port2)
	if nerr2 != nil {
		fmt.Fprintf(os.Stderr, "Error -  net.Listen 2 failed: %v", nerr2)
	}
	c.Assert(nerr2, IsNil)

	go s2.Serve(lis2)
	fmt.Printf("Serving '%s' on port%s\n", service2, port2)
}

func (p *SigtestSuite) TearDownSuite(c *C) {
	fmt.Printf("\nTearing Down Suite\n")

	// Remove that key from "pubkeys.db", even though it is a transient /tmp dir..
	derr := grpcsig.RemoveSelfPubKeysFromDB()
	fmt.Printf("RemoveSelfPubKeysFromDB() %s Errors?: [%v]\n", testdb, derr)
	c.Assert(derr, IsNil)

	// Remove the private key from ssh-agent
	err := grpcsig.FiniSelfSSHKeys(true)
	fmt.Printf("FiniSelfSSHKeys() Errors?: [%v]\n", err)
	c.Assert(err, IsNil)

	// Stop the lookup service
	grpcsig.FiniDefaultService()

	fmt.Printf("Teardown done.\n")
}

func (p *SigtestSuite) TestTwoSelfClients(c *C) {

	// Start up client1 and client2
	fmt.Printf("\nStarting Client '%s' on port%s\n", service1, port1)

	selfSigner, err := grpcsig.SelfSigner(nil) // TEST SELF-SIGNATURES
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal - ssh-agent initialization failed: %v\n", err)
	}
	c.Assert(err, IsNil)
	conn1, err := selfSigner.Dial(port1)
	defer conn1.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Did not connect to server1: %v\n", err)
	}
	c.Assert(err, IsNil)

	// TEST SELF-SIGNATURES -- calling service2 on port2 with selfSigner
	fmt.Printf("Starting Client '%s' on port%s\n", service2, port2)
	conn2, err := selfSigner.Dial(port2)
	defer conn2.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Did not connect to server2: %v\n", err)
	}
	c.Assert(err, IsNil)

	// Try service 1, service 2
	fmt.Printf("\nCommunicate with Server1 '%s' on port%s\n", service1, port1)
	Communicate(conn1)

	fmt.Printf("Communicate with Server2 '%s' on port%s\n", service2, port2)
	Communicate(conn2)

	fmt.Printf("Communicate with Server1 '%s' on port%s\n", service1, port1)
	Communicate(conn1)

	fmt.Printf("Communicate with Server2 '%s' on port%s\n", service2, port2)
	Communicate(conn2)

	fmt.Printf("Communicate with Server1 '%s' on port%s\n", service1, port1)
	Communicate(conn1)

	fmt.Printf("Communicate with Server2 '%s' on port%s\n", service2, port2)
	Communicate(conn2)

	fmt.Printf("Completed Self-Communication.\n")
}

func Communicate(conn *grpc.ClientConn) {
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

// -----------------
// Below is not really relevant to the test, just necessary to communicate

// server - implements sigtest.SigtestServer, saves inbound sigtests
type server struct {
	savedSigtests []*pb.SigtestRequest
}

// CreateSigtest - appends an input sigtest to save list, responds via unary
func (s *server) CreateSigtest(ctx context.Context, in *pb.SigtestRequest) (*pb.SigtestResponse, error) {
	s.savedSigtests = append(s.savedSigtests, in)
	return &pb.SigtestResponse{Id: in.Id, Success: true}, nil
}

// GetSigtest - returns all saved sigtests via stream
func (s *server) GetSigtests(all *pb.SigtestAll, stream pb.Sigtest_GetSigtestsServer) error {
	if all.IsAll != true {
		return nil
	}
	for _, sigtest := range s.savedSigtests {
		if err := stream.Send(sigtest); err != nil {
			return err
		}
	}
	return nil
}

// Two CLIENT gRPC methods are used to test the Unary and Stream Interceptors

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
