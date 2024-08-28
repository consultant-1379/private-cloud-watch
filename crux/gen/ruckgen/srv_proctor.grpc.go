// Code generated from srv_proctor.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Proctor

type ProctorServerStarter struct {
	xxx pb.ProctorServer
}

// RegisterServer : rhubarb
func (d ProctorServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterProctorServer(s, d.xxx)
}

// Name : rhubarb
func (d ProctorServerStarter) Name() string {
	return "Proctor"
}

// StartProctorServer starts and registers the Proctor (grpc whitelist) server; exit on error for now
func StartProctorServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.ProctorServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, ProctorServerStarter{xxx}, quit, reeveapi)
}

// ConnectProctor is how you connect to a Proctor
func ConnectProctor(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.ProctorClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Proctor")
	if err != nil {
		return nil, err
	}
	return pb.NewProctorClient(conn), nil
}
