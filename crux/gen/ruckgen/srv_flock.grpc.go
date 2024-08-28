// Code generated from srv_flock.proto by tools/server/main.go; DO NOT EDIT.

package ruckgen

import (
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsvc"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/rucklib"
)

// Code block for service Flock

type FlockServerStarter struct {
	xxx pb.FlockServer
}

// RegisterServer : rhubarb
func (d FlockServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterFlockServer(s, d.xxx)
}

// Name : rhubarb
func (d FlockServerStarter) Name() string {
	return "Flock"
}

// StartFlockServer starts and registers the Flock (grpc whitelist) server; exit on error for now
func StartFlockServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.FlockServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, FlockServerStarter{xxx}, quit, reeveapi)
}

// ConnectFlock is how you connect to a Flock
func ConnectFlock(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.FlockClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Flock")
	if err != nil {
		return nil, err
	}
	return pb.NewFlockClient(conn), nil
}
