package main

import (
	"fmt"
	"math/rand"
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

func ExamplePlugin(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	logger.Log(nil, "example starting")
	heart := time.Tick(2 * time.Second)
	bye := time.NewTimer(time.Duration(30+rand.Intn(40)) * time.Second)
	node := (**network).GetNames().Node
	alive <- []pb.HeartbeatReq{{State: pb.ServiceState_NMAP, UUID: UUID, Various: fmt.Sprintf("%s %s %s", node, "file", "ExamplePlugin"), At: crux.Time2Timestamp(time.Now().UTC())}}
loop:
	for {
		select {
		case <-heart:
			alive <- []pb.HeartbeatReq{pb.HeartbeatReq{
				UUID:    UUID,
				State:   pb.ServiceState_UP,
				At:      crux.Time2Timestamp(time.Now().UTC()),
				Expires: crux.Time2Timestamp(time.Now().UTC().Add(5 * time.Second)),
			}}

			logger.Log(nil, "example heartbeat")
		case <-quit:
			break loop
		case <-bye.C:
			break loop
		}
	}
	logger.Log(nil, "example ending")
	return idutils.NetIDT{}
}

func LogPlugin(quit <-chan bool, adv chan<- []crux.Fservice, network **crux.Confab) {
	xx := 1 * time.Second
	heart := time.Tick(xx)
	dummy := make([]crux.Fservice, 1)
	cfgLog := crux.GetLogger()
	cfgLog.Log("shinyInteger", 5, "We-Feel", "really Super")
loop:
	for {
		select {
		case <-heart:
			adv <- dummy
		case <-quit:
			break loop
		}
	}
	adv <- nil
}

func main() {
	// nothing to see or do here
}
