package main

import (
	"fmt"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/ruck"
)

func main() {
	fmt.Printf("Not intended to run this func.")
	nod, _ := idutils.NewNodeID("flock", "horde", "node", "cName", "cAPI")
	HealthCheck1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
	Heartbeat1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
	Picket1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
	Dean1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
	Pastiche0_1(nil, nil, nil, "uuid", nil, nod, "", nil)
	Proctor1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
	Genghis1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
	Yurt1_0(nil, nil, nil, "uuid", nil, nod, "", nil)
}

func Heartbeat1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Heartbeat1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func HealthCheck1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.HealthCheck1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Picket1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Picket1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Dean1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Dean1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Proctor1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Proctor1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Genghis1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Genghis1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Yurt1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Yurt1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Pastiche0_1(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Pastiche0_1(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}

func Metric1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	return ruck.Metric1_0(quit, alive, network, UUID, logger, nod, eRev, reeveapi)
}
