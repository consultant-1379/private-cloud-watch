// Code generated from srv_genghis.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Genghis

type GenghisServerStarter struct {
	xxx pb.GenghisServer
}

// RegisterServer : rhubarb
func (d GenghisServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterGenghisServer(s, d.xxx)
}

// Name : rhubarb
func (d GenghisServerStarter) Name() string {
	return "Genghis"
}

// StartGenghisServer starts and registers the Genghis (grpc whitelist) server; exit on error for now
func StartGenghisServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.GenghisServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, GenghisServerStarter{xxx}, quit, reeveapi)
}

// ConnectGenghis is how you connect to a Genghis
func ConnectGenghis(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.GenghisClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Genghis")
	if err != nil {
		return nil, err
	}
	return pb.NewGenghisClient(conn), nil
}
