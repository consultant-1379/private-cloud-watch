package ruck

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	ruck "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/horde"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// reeve-related specs
const (
	YurtName  = "Yurt"
	YurtAPI   = "Yurt1"
	YurtRev   = "Yurt1_0"
	yurtCycle = 1 * time.Second
)

// Yurt is our base type for describing the horde
type Yurt struct {
	sync.Mutex
	doneq     chan bool
	alive     chan<- []pb.HeartbeatReq
	network   **crux.Confab
	update    time.Time
	adm       horde.Administer
	act       horde.Action
	spec      string
	svcs      map[string]*pb.HeartbeatReq
	log       clog.Logger
	me        string
	picket    *pb.PicketClient
	heartbeat *pb.HeartbeatClient
	reeveapi  *reeve.StateT
	nod       idutils.NodeIDT
}

// Quit for gRPC
func (d *Yurt) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	d.log.Log(nil, "--->quit %v\n", *in)
	d.doneq <- true // Afib
	d.doneq <- true // inpulse
	return nil, nil
}

// Who returns who is up
func (d *Yurt) Who(ctx context.Context, in *pb.Empty) (*pb.HeartbeatsResponse, error) {
	d.Lock()
	defer d.Unlock()

	var ret []*pb.HeartbeatReq
	for _, x := range d.svcs {
		ret = append(ret, x)
	}
	d.log.Log(nil, "yurt who returns >%+v<  (from svcs=%s)", ret, d.svcs)
	return &pb.HeartbeatsResponse{List: ret}, nil
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (d *Yurt) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	d.log.Log(nil, "--->ping %v\n", *ping)
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Yurt1_0 is the low-level khan.
// this has to cover both restarting and initialisation.
// for now, we don't consider restart.
func Yurt1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	d := Yurt{
		doneq:    make(chan bool, 2), // coordinate this with activity in quit
		alive:    alive,
		network:  network,
		log:      logger.With("focus", YurtRev),
		reeveapi: reeveapi,
		nod:      nod,
	}
	d.me = (**network).GetNames().Node
	nod = ReNOD(nod, YurtName, YurtAPI)
	nid := ruck.StartYurtServer(&nod, YurtRev, nod.NodeName, 0, &d, quit, reeveapi)
	go Afib(alive, d.doneq, UUID, "", nid)
	go d.inpulse()

	d.openHeartbeat()

	logger.Log(nil, "yurt started: picketNID=%s  heartbeatNID=%s", picketNID.String(), heartbeatNID.String())

	return nid
}

func (d *Yurt) openHeartbeat() {
	hbsign := d.reeveapi.SelfSigner()
	x, err := ruck.ConnectHeartbeat(heartbeatNID, hbsign, d.log)
	d.heartbeat = &x
	if err != nil {
		d.log.Log(nil, "yurt ConnectHeartbeat failed: %v", err)
		return
	}
	d.log.Log(nil, "yurt openHeartbeat succeeded!")
}

// read new stuff
func (d *Yurt) inpulse() {
	beat := time.NewTicker(yurtCycle)
	for {
		d.log.Log("inpulse")
		select {
		case <-beat.C:
			if d.heartbeat != nil {
				if hbr, err := (*d.heartbeat).Heartbeats(context.Background(), &pb.Empty{}); err == nil {
					d.absorb(hbr.List)
				}
			}
			d.log.Log(nil, "yurt inpulse")
		case <-d.doneq:
			// we're exiting!!
			beat.Stop()
			return
		}
	}
}

func (d *Yurt) absorb(list []*pb.HeartbeatReq) {
	nmap := make(map[string]*pb.HeartbeatReq)
	for _, x := range list {
		if x.NID == "" {
			continue
		}
		nmap[x.UUID] = x
	}
	if true {
		var x string
		for _, s := range nmap {
			x += fmt.Sprintf(" %s(%s)", s.UUID, s.NID)
		}
		d.log.Log(nil, "yurt absorbed %d services:%s", len(nmap), x)
	} else {
		d.log.Log(nil, "yurt absorbed %d services: %+v", len(nmap), nmap)
	}
	d.Lock()
	d.svcs = nmap
	d.Unlock()
}
