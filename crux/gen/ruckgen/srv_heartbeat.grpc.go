// Code generated from srv_heartbeat.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service Heartbeat

type HeartbeatServerStarter struct {
	xxx pb.HeartbeatServer
}

// RegisterServer : rhubarb
func (d HeartbeatServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterHeartbeatServer(s, d.xxx)
}

// Name : rhubarb
func (d HeartbeatServerStarter) Name() string {
	return "Heartbeat"
}

// StartHeartbeatServer starts and registers the Heartbeat (grpc whitelist) server; exit on error for now
func StartHeartbeatServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.HeartbeatServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, HeartbeatServerStarter{xxx}, quit, reeveapi)
}

// ConnectHeartbeat is how you connect to a Heartbeat
func ConnectHeartbeat(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.HeartbeatClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "Heartbeat")
	if err != nil {
		return nil, err
	}
	return pb.NewHeartbeatClient(conn), nil
}
