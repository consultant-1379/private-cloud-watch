package ruck

/*
	The heartbeat is a node-centric server that collects heartbeats via a channel and
then redistributes them to the HealthCheck server. It also services requests for aliveness.

	The issue of how to handle expiring heartbeats is complicated in that the healthcheck
daemon might not exist for extended periods of time (especially at the start of time). As far
as possible, we want to send all heartbeats, expired or not, to healthcheck. But alas, that is
not always possible.

	So the output rules are
1) healthcheck gets every heartbeat collected, except that heartbeats older than reallyDead
	(or so) will be dropped.
2) the output of the Heartbeats method will be the latest heartbeat from every distinct UUID
	that has not expired.
*/

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	ruck "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
	rl "github.com/erixzone/crux/pkg/rucklib"
)

// name constants for this service
const (
	HeartbeatName = "Heartbeat"
	HeartbeatAPI  = "Heartbeat1"
	HeartbeatRev  = "Heartbeat1_0"

	outbound   = 5 * time.Second  // outbound frequescy for sending new heartbeats
	reallyDead = 60 * time.Second // heartbeats (approximately) this old never get sent to healthcheck
)

// HeartbeatServer - implement srv_heartbeat.proto
type HeartbeatServer struct {
	sync.Mutex
	alarm        *time.Ticker
	tsent        time.Time
	inbound      <-chan []pb.HeartbeatReq
	sent, unsent []*pb.HeartbeatReq
	doneq        chan bool
	log          clog.Logger
	out          chan crux.MonInfo
	nod          idutils.NodeIDT
}

// NewHeartbeatServer  - get one.
func NewHeartbeatServer(inb <-chan []pb.HeartbeatReq, lg clog.Logger, nod idutils.NodeIDT, reeveapi *reeve.StateT, out chan crux.MonInfo) *HeartbeatServer {
	hs := &HeartbeatServer{
		inbound: inb,
		sent:    make([]*pb.HeartbeatReq, 0),
		unsent:  make([]*pb.HeartbeatReq, 0),
		alarm:   time.NewTicker(HeartGap),
		tsent:   time.Now().UTC(),
		doneq:   make(chan bool, 2),
		log:     lg.With("focus", "heartbeatserver"),
		out:     out,
		nod:     nod,
	}
	go hs.draino(lg, out)
	go hs.sender(nod, reeveapi, lg, out)
	return hs
}

// Quit for gRPC
func (hcs *HeartbeatServer) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	hcs.log.Log(nil, "--->heartbeat quit %v\n", *in)
	hcs.doneq <- true // draino
	hcs.doneq <- true // sender
	hcs.doneq <- true // Afib
	return &pb.QuitReply{Message: ""}, nil
}

// draino absorbs heartbeats into the beats queue
func (hcs *HeartbeatServer) draino(lg clog.Logger, out chan crux.MonInfo) {
	logger := lg.With("focus", "draino")
	for {
		select {
		case hbl := <-hcs.inbound:
			hcs.Lock()
			for _, x := range hbl {
				if x.At != nil {
					hcs.unsent = append(hcs.unsent, &x)
				}
			}
			logger.Log(nil, "finished loop read (%d unsents)", len(hcs.unsent))
			hcs.Unlock()
		case <-hcs.doneq:
			logger.Log(nil, "draino exiting")
			return
		}
	}
}

// sender sends the heartbeats off
func (hcs *HeartbeatServer) sender(nod idutils.NodeIDT, reeveapi *reeve.StateT, lg clog.Logger, out chan crux.MonInfo) {
	var active bool
	var hcheck pb.HealthCheckClient

	logger := lg.With("focus", "sender")
	ensure := func() bool {
		if !active {
			hcnid, hcsigner, err := rl.Get1Endpoint(nod, HealthCheckRev, reeveapi)
			if err != nil {
				hcs.log.Log(nil, "sender Get1Endpoint(HealthCheck) failed: %v; continuing", err)
				return false
			}
			hcheck, err = ruck.ConnectHealthCheck(hcnid, hcsigner, hcs.log)
			if err != nil {
				hcs.log.Log(nil, "sender: ConnectHealthCheck failed: %v; continuing", err)
				return false
			}
			active = true
			logger.Log(nil, "ensure returns true")
		}
		return active
	}

	sclock := time.NewTicker(outbound)
	logger.Log(nil, "senderclock started")
	for {
		logger.Log(nil, "sender loop")
		select {
		case <-sclock.C:
			logger.Log(nil, "senderclock ticked %d", len(hcs.unsent))
			if ensure() {
				hcs.Lock()
				hcs.log.Log(nil, "sending %d heartbeats %+v", len(hcs.unsent), out)
				erb, err := hcheck.AbsorbBeats(context.Background(), &pb.HeartbeatsResponse{List: hcs.unsent})
				logger.Log(nil, "absorbbeats return erb=%+v err=%v", erb, err)
				hcs.sent = append(hcs.sent, hcs.unsent...)
				hcs.unsent = make([]*pb.HeartbeatReq, 0)
				hcs.Unlock()
			}
			hcs.obliterate()
		case <-hcs.doneq:
			logger.Log(nil, "sender exiting")
			sclock.Stop()
			if ensure() {
				hcs.Lock()
				hcs.log.Log(nil, "sending %d heartbeats", len(hcs.unsent))
				erb, err := hcheck.AbsorbBeats(context.Background(), &pb.HeartbeatsResponse{List: hcs.unsent})
				logger.Log(nil, "absorbbeats return erb=%+v err=%v", erb, err)
				hcs.sent = append(hcs.sent, hcs.unsent...)
				hcs.unsent = make([]*pb.HeartbeatReq, 0)
				hcs.Unlock()
			}
			logger.Log(nil, "--sender exited")
			return
		}
	}
}

// eliminate reallyDead heartbeats
func (hcs *HeartbeatServer) obliterate() {
	hcs.Lock()
	defer hcs.Unlock()
	cutoff := time.Now().UTC().Add(-reallyDead)
	nsent := len(hcs.sent)
	nunsent := len(hcs.unsent)
	var xs, xus []*pb.HeartbeatReq
	for _, h := range hcs.unsent {
		if !crux.Timestamp2Time(h.At).Before(cutoff) {
			xus = append(xus, h)
		}
	}
	hcs.unsent = xus
	for _, h := range hcs.sent {
		if !crux.Timestamp2Time(h.At).Before(cutoff) {
			xs = append(xs, h)
		}
	}
	hcs.sent = xs
	var worked string
	if (nsent != len(hcs.sent)) || (nunsent != len(hcs.unsent)) {
		worked = " fired"
	}
	hcs.log.Log(nil, "obliterate%s: sent(%d -> %d), unsent(%d -> %d)", worked, nsent, len(hcs.sent), nunsent, len(hcs.unsent))
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (hcs *HeartbeatServer) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Heartbeats returns the heartbeats
func (hcs *HeartbeatServer) Heartbeats(ctx context.Context, in *pb.Empty) (*pb.HeartbeatsResponse, error) {
	togo := make(map[string]*pb.HeartbeatReq)

	// get the latest heartbeat for each uuid
	work := func(list []*pb.HeartbeatReq) {
		for _, hb := range list {
			if x, ok := togo[hb.UUID]; !ok || crux.Timestamp2Time(hb.At).After(crux.Timestamp2Time(x.At)) {
				togo[hb.UUID] = hb
			}
		}
	}
	hcs.Lock()
	defer hcs.Unlock()

	// figure out what to send
	work(hcs.sent)
	work(hcs.unsent)
	// now send them
	var ret []*pb.HeartbeatReq
	for _, hb := range togo {
		ret = append(ret, hb)
	}
	return &pb.HeartbeatsResponse{List: ret}, nil
}

// Heartbeat1_0 is this release.
func Heartbeat1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, log clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	nod = ReNOD(nod, HeartbeatName, HeartbeatAPI)
	log.Log(nil, "starting %s nod=%+v", HeartbeatName, nod)
	hb := NewHeartbeatServer(heartChan, log, nod, reeveapi, (**network).Monitor())
	nid := ruck.StartHeartbeatServer(&nod, HeartbeatRev, nod.NodeName, 0, hb, quit, reeveapi)
	go Afib(alive, hb.doneq, UUID, "", nid)
	heartbeatNID = nid
	log.Log(nil, "ending %s", HeartbeatName)
	return nid
}
