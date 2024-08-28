package ruck

/*
**Healthcheck**  is a plugin for fulcrum. It is fed with the other plugins' "alive" data which it reports to Reeve
and also updates a grpc health server (running in go routine) so status of any service can be queried remotely.

The google *GRPC Health Checking Protocol* defined in health.proto has only a status int, to describe a server's status.
The local crux aliveness information can be much richer.

**HealthcheckServer** is the implementation of the grpc health.v1 server with a few extensions like Ping and Heartbeat

Similar to
   https://godoc.org/google.golang.org/grpc/health

We are using  a copy of the official health.proto
https://github.com/grpc/grpc/blob/v1.15.0/src/proto/grpc/health/v1/health.proto
with a different package name than "grpc.health.v1".
*/

import (
	"context"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/erixzone/crux/gen/cruxgen"
	ruck "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// name constants for this service
const (
	HealthCheckName = "HealthCheck"
	HealthCheckAPI  = "HealthCheck1"
	HealthCheckRev  = "HealthCheck1_0"

	Window = 12 * time.Hour
)

// HealthCheckServer - implement grpc api defined by .grpc.health.v1
// and used in srv_healthcheck.proto
type HealthCheckServer struct {
	sync.Mutex
	// Serving status's
	// UNKNOWN = 0
	// SERVING = 1
	// NOT_SERVING = 2

	beats   []pb.HeartbeatReq
	doneq   chan bool
	log     clog.Logger
	network **crux.Confab
}

// NewHealthCheckServer  - get one.
func NewHealthCheckServer(log clog.Logger, network **crux.Confab) *HealthCheckServer {
	return &HealthCheckServer{
		doneq:   make(chan bool),
		beats:   make([]pb.HeartbeatReq, 1),
		log:     log,
		network: network,
	}
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (hcs *HealthCheckServer) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Quit for gRPC
func (hcs *HealthCheckServer) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	hcs.log.Log(nil, "--->healthcheck quit %v", *in)
	hcs.doneq <- true
	return &pb.QuitReply{Message: ""}, nil
}

// Check - This is the only function grpc.health.v1 has.  Return
// status for the service specified in the grpc request. Or the status
// of this healthserver if the service is empty.
func (hcs *HealthCheckServer) Check(ctx context.Context, in *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	hcs.Lock()
	defer hcs.Unlock()
	// No service specified.  Just let them know we're up (at least healthChecking is).
	if in.Srvice == "" {
		// check the server overall health status.
		return &pb.HealthCheckResponse{
			Status: pb.ServingStatus_SERVING,
		}, nil
	}
	// TODO: Support getting status for multiple services at once
	// with a synthetic service  like "all", "basic", etc.
	// The healthcheck spec allows this, but behavior is user defined.

	// Return status for a specific service. except this broken for now, because we don't know how to name things. TBD
	if status, ok := hcs.beats[0], true; ok {
		var st pb.ServingStatus
		if time.Now().UTC().After(crux.Timestamp2Time(status.Expires)) {
			st = pb.ServingStatus_NOT_SERVING
		} else {
			st = pb.ServingStatus_SERVING
		}
		return &pb.HealthCheckResponse{
			Status: st,
		}, nil
	}
	return nil, status.Error(codes.NotFound, "unknown service")
}

// Heartbeats returns heartbeats for Crux work
func (hcs *HealthCheckServer) Heartbeats(ctx context.Context, in *pb.Empty) (*pb.HeartbeatsResponse, error) {
	var ret []*pb.HeartbeatReq
	for _, b := range hcs.beats {
		ret = append(ret, &b)
	}
	return &pb.HeartbeatsResponse{List: ret}, nil
}

// Len (sorting)
func (hcs *HealthCheckServer) Len() int {
	return len(hcs.beats)
}

// Less (sorting)
func (hcs *HealthCheckServer) Less(i, j int) bool {
	return crux.Timestamp2Time(hcs.beats[i].At).Before(crux.Timestamp2Time(hcs.beats[j].At))
}

// Swap (sorting)
func (hcs *HealthCheckServer) Swap(i, j int) {
	hcs.beats[i], hcs.beats[j] = hcs.beats[j], hcs.beats[i]
}

// AbsorbBeats absorbs heartbeats
func (hcs *HealthCheckServer) AbsorbBeats(ctx context.Context, in *pb.HeartbeatsResponse) (*pb.AbsorbResponse, error) {
	hcs.Lock()
	defer hcs.Unlock()
	hcs.log.Log(nil, "absorb given %d beats", len(in.List))
	for _, hb := range in.List {
		hcs.beats = append(hcs.beats, *hb)
		hcs.beacon(hb)
	}
	hcs.log.Log(nil, "sorting %+v", hcs.beats[:10])
	sort.Sort(hcs)
	cutoff := time.Now().UTC().Add(-Window)
	var i int
	for i = range hcs.beats {
		if !crux.Timestamp2Time(hcs.beats[i].At).Before(cutoff) {
			break
		}
	}
	hcs.beats = hcs.beats[i:]
	hcs.log.Log(nil, "absorb read %d beats; after date trim |beats|=%d", len(in.List), len(hcs.beats))
	return &pb.AbsorbResponse{Err: crux.Err2Proto(nil)}, nil
}

var hbuuid = 1001

func (hcs *HealthCheckServer) beacon(hb *pb.HeartbeatReq) {
	if ch := (**hcs.network).Monitor(); ch != nil {
		hcs.log.Log(nil, "beacon-> %+v", *hb)
		ch <- crux.MonInfo{Op: crux.HeartBeatOp, Moniker: hb.UUID, N: int(hb.State), T: crux.Timestamp2Time(hb.At), Oflock: hb.Various}
		hbuuid++
	}
}

// HealthCheck1_0 is this release.
func HealthCheck1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, log clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	log.Log(nil, "starting %s (ch=%v || %v) nod=%+v eRev=%s reeveapi=%p", HealthCheckName, alive, (**network).Monitor(), nod, eRev, reeveapi)
	hc := NewHealthCheckServer(log, network) //nil, network, horde, ipname, reeveapi)
	nod = ReNOD(nod, HealthCheckName, HealthCheckAPI)
	nid := ruck.StartHealthCheckServer(&nod, HealthCheckRev, nod.NodeName, 0, hc, quit, reeveapi)
	go Afib(alive, hc.doneq, UUID, "", nid)
	log.Log(nil, "ending %s nod=%s nid=%s", HealthCheckName, nod, nid)
	return nid
}
