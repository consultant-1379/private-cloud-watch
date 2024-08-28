package ruck

import (
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
)

// Afib is a standardised heartbeat generator
func Afib(alive chan<- []pb.HeartbeatReq, quit <-chan bool, UUID, v string, nid idutils.NetIDT) {
	send1 := func() {
		alive <- []pb.HeartbeatReq{pb.HeartbeatReq{
			UUID:    UUID,
			State:   pb.ServiceState_UP,
			NID:     nid.String(),
			At:      crux.Time2Timestamp(time.Now().UTC()),
			Expires: crux.Time2Timestamp(time.Now().UTC().Add(2 * HeartGap)),
			Various: v,
		}}
		clog.Log.Log(nil, "afib hb for %s", nid.String())
	}
	sclock := time.NewTicker(HeartGap)
	clog.Log.Log(nil, "heartclock %s starting hc=%+v gap=%s", UUID, heartChan, HeartGap.String())
	send1()
	for {
		select {
		case <-sclock.C:
			send1()
		case <-quit:
			sclock.Stop()
			clog.Log.Log(nil, "heartclock %s exiting", UUID)
			return
		}
	}
}

// ReNOD does a quickie on an existing NodeID
// note - this bypasses allowed name, api character vaidations in idutils.
func ReNOD(nod idutils.NodeIDT, name, api string) idutils.NodeIDT {
	ret := nod
	ret.ServiceName = name
	ret.ServiceAPI = api
	return ret
}
