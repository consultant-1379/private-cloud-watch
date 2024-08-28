// Code generated from srv_reeve.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Reeve

type ReeveServerStarter struct {
	xxx pb.ReeveServer
}

// RegisterServer : rhubarb
func (d ReeveServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterReeveServer(s, d.xxx)
}

// Name : rhubarb
func (d ReeveServerStarter) Name() string {
	return "Reeve"
}

// StartReeveServer starts and registers the Reeve (grpc whitelist) server; exit on error for now
func StartReeveServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.ReeveServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, ReeveServerStarter{xxx}, quit, reeveapi)
}

// ConnectReeve is how you connect to a Reeve
func ConnectReeve(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.ReeveClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Reeve")
	if err != nil {
		return nil, err
	}
	return pb.NewReeveClient(conn), nil
}
