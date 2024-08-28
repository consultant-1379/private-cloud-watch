// Code generated from srv_dean.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Dean

type DeanServerStarter struct {
	xxx pb.DeanServer
}

// RegisterServer : rhubarb
func (d DeanServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterDeanServer(s, d.xxx)
}

// Name : rhubarb
func (d DeanServerStarter) Name() string {
	return "Dean"
}

// StartDeanServer starts and registers the Dean (grpc whitelist) server; exit on error for now
func StartDeanServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.DeanServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, DeanServerStarter{xxx}, quit, reeveapi)
}

// ConnectDean is how you connect to a Dean
func ConnectDean(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.DeanClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Dean")
	if err != nil {
		return nil, err
	}
	return pb.NewDeanClient(conn), nil
}
