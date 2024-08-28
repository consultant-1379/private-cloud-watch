// Code generated from srv_pastiche.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service PasticheSrv

type PasticheSrvServerStarter struct {
	xxx pb.PasticheSrvServer
}

// RegisterServer : rhubarb
func (d PasticheSrvServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterPasticheSrvServer(s, d.xxx)
}

// Name : rhubarb
func (d PasticheSrvServerStarter) Name() string {
	return "PasticheSrv"
}

// StartPasticheSrvServer starts and registers the PasticheSrv (grpc whitelist) server; exit on error for now
func StartPasticheSrvServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.PasticheSrvServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, PasticheSrvServerStarter{xxx}, quit, reeveapi)
}

// ConnectPasticheSrv is how you connect to a PasticheSrv
func ConnectPasticheSrv(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.PasticheSrvClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "PasticheSrv")
	if err != nil {
		return nil, err
	}
	return pb.NewPasticheSrvClient(conn), nil
}
