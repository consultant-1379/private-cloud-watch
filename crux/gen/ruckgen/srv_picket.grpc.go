// Code generated from srv_picket.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Picket

type PicketServerStarter struct {
	xxx pb.PicketServer
}

// RegisterServer : rhubarb
func (d PicketServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterPicketServer(s, d.xxx)
}

// Name : rhubarb
func (d PicketServerStarter) Name() string {
	return "Picket"
}

// StartPicketServer starts and registers the Picket (grpc whitelist) server; exit on error for now
func StartPicketServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.PicketServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, PicketServerStarter{xxx}, quit, reeveapi)
}

// ConnectPicket is how you connect to a Picket
func ConnectPicket(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.PicketClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Picket")
	if err != nil {
		return nil, err
	}
	return pb.NewPicketClient(conn), nil
}
