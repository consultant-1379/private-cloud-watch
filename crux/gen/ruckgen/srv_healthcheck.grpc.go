// Code generated from srv_healthcheck.proto by tools/server/main.go; DO NOT EDIT.

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

// Code block for service HealthCheck

type HealthCheckServerStarter struct {
	xxx pb.HealthCheckServer
}

// RegisterServer : rhubarb
func (d HealthCheckServerStarter) RegisterServer(s *grpc.Server) {
	pb.RegisterHealthCheckServer(s, d.xxx)
}

// Name : rhubarb
func (d HealthCheckServerStarter) Name() string {
	return "HealthCheck"
}

// StartHealthCheckServer starts and registers the HealthCheck (grpc whitelist) server; exit on error for now
func StartHealthCheckServer(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.HealthCheckServer, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, HealthCheckServerStarter{xxx}, quit, reeveapi)
}

// ConnectHealthCheck is how you connect to a HealthCheck
func ConnectHealthCheck(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.HealthCheckClient, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "HealthCheck")
	if err != nil {
		return nil, err
	}
	return pb.NewHealthCheckClient(conn), nil
}
