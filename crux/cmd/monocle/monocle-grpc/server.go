package main

import (
	//"context"

	"fmt"
	"net"

	context "golang.org/x/net/context" // built-in context not enough for grpc. (Until golang 1.9)
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
)

// PingTestResp = value returned from a GetDump of the "ping"
// subsystem.  For liveness testing.
var PingTestResp = "PING RESPONSE"

// PingSubsys - A dummy subsystem label for use in liveness tests.
var PingSubsys = "ping"

// Server - implement gRPC interface for monocle library.
type Server struct {
	validSubsystems []string // The legal subsystems that we can get
	// dumps for.  Could be more than one
	// subsystem per microservice.
}

// NewServer - Return a grpc server
func NewServer() (*Server, error) {
	// TODO: initialize subsystem strings

	// TODO: Determine how we'll map from subsystem names to grpc servers
	//  For instance, if we have multiple pastiche servers

	// We could just hand wire for every new function, but it
	// seems like we could do something more intelligent using
	// reeve/steweard and/or code generation.
	return &Server{}, nil
}

// GetDump -
func (*Server) GetDump(ctx context.Context, pReq *pb.GetDumpRequest) (*pb.GetDumpResponse, error) {

	// TODO: Allow subsys argument to have multiple subsystems.

	resp := pb.GetDumpResponse{}
	resp.Success = true
	fmt.Printf("Subsystem: %s   Level:%d\n", pReq.Subsystems, pReq.Level)
	switch pReq.Subsystems {
	case PingSubsys:
		resp.Data = PingTestResp
	case "pastiche":
		// TODO: Grpc to pastiche server(s) for info.  Need to
		// figure out how we decide which pastiche server(s)
		// to contact after Steward lookup.  All of them? A
		// single one? We need the ability to collect info for
		// both a single node, and all nodes.

		resp.Data = "UNIMPLEMENTED"
	default:
		resp.Success = false
		resp.Data = "unknown subsystem <" + pReq.Subsystems + ">"
	}

	return &resp, nil
}

// Start - The grpc server will start listening for requests.
func (ps *Server) Start(port string) error {
	port = ":" + port
	clog.Log.Log(nil, "Starting Monocle server on localhost, addr: 127.0.0.1%s\n", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		clog.Log.Fatal(nil, "failed to listen: %v", err)
		return err
	}
	s := grpc.NewServer()
	pb.RegisterMonocleServer(s, ps)
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		clog.Log.Fatal(nil, "failed to serve: %v", err)
		return err
	}

	return nil
}
