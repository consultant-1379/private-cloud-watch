// Code generated from srv_yurt.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Yurt

type YurtServerStarter struct {
	xxx pb.YurtServer
}

// RegisterServer : rhubarb
func (d YurtServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterYurtServer(s, d.xxx)
}

// Name : rhubarb
func (d YurtServerStarter) Name() string {
	return "Yurt"
}

// StartYurtServer starts and registers the Yurt (grpc whitelist) server; exit on error for now
func StartYurtServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.YurtServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, YurtServerStarter{xxx}, quit, reeveapi)
}

// ConnectYurt is how you connect to a Yurt
func ConnectYurt(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.YurtClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Yurt")
	if err != nil {
		return nil, err
	}
	return pb.NewYurtClient(conn), nil
}
