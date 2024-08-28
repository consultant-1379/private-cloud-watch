// Code generated from srv_steward.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Steward

type StewardServerStarter struct {
	xxx pb.StewardServer
}

// RegisterServer : rhubarb
func (d StewardServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterStewardServer(s, d.xxx)
}

// Name : rhubarb
func (d StewardServerStarter) Name() string {
	return "Steward"
}

// StartStewardServer starts and registers the Steward (grpc whitelist) server; exit on error for now
func StartStewardServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.StewardServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, StewardServerStarter{xxx}, quit, reeveapi)
}

// ConnectSteward is how you connect to a Steward
func ConnectSteward(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.StewardClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Steward")
	if err != nil {
		return nil, err
	}
	return pb.NewStewardClient(conn), nil
}
