package main

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/net/context"

	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	pb "github.com/erixzone/crux/pkg/grpcsig/test/gen"
)

const (
	port = ":50052"
)

// server - implements sigtest.SigtestServer, saves inbound sigtests
type server struct {
	savedSigtests []*pb.SigtestRequest
}

// CreateSigtest - appends an input sigtest to save list, responds via unary
func (s *server) CreateSigtest(ctx context.Context, in *pb.SigtestRequest) (*pb.SigtestResponse, error) {
	s.savedSigtests = append(s.savedSigtests, in)
	return &pb.SigtestResponse{Id: in.Id, Success: true}, nil
}

// GetSigtest - returns all saved igtests via stream
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

func main() {

	// -----------------
	// gRPC http-signatures demo

	// Initialize default grpcsig system service with provided database and 300s clockskew

	dbname := "pubkeys_test.db"
	// service := "phlogiston"
	service := "jettison"
	var cerr *c.Err
	logmain := clog.Log.With("focus", "test-server")
	httpsigService, cerr := grpcsig.InitDefaultService(dbname, service, nil, logmain, 300, true)
	if cerr != nil {
		fmt.Fprintf(os.Stderr, "%s\nStack: %s\n", cerr.String(), cerr.Stack)
		os.Exit(1)
	}
	defer grpcsig.FiniDefaultService()

	// Start  gRPC server with Interceptors for http-signatures inbound
	s := httpsigService.NewServer()
	//------------------

	pb.RegisterSigtestServer(s, &server{})

	lis, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error -  net.Listen failed: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Serving '%s' on port%s\n", service, port)
	s.Serve(lis)
}
