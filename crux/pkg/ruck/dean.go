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
	"github.com/erixzone/crux/pkg/khan"
	"github.com/erixzone/crux/pkg/kv"
	"github.com/erixzone/crux/pkg/reeve"
)

// reeve-related specs
const (
	DeanName  = "Dean"
	DeanAPI   = "Dean1"
	DeanRev   = "Dean1_0"
	execCycle = 10 * time.Second
	inCycle   = 1 * time.Second
)

// Dean is our base type for describing the horde
type Dean struct {
	sync.Mutex
	doneq     chan bool
	alive     chan<- []pb.HeartbeatReq
	network   **crux.Confab
	update    time.Time
	kv        kv.KV
	adm       horde.Administer
	act       horde.Action
	spec      string
	svcs      map[string]*horde.Service
	log       clog.Logger
	me        string
	picket    *pb.PicketClient
	heartbeat *pb.HeartbeatClient
	reeveapi  *reeve.StateT
	nod       idutils.NodeIDT
}

// Quit for gRPC
func (d *Dean) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	d.log.Log(nil, "--->quit %v\n", *in)
	d.doneq <- true // Afib
	d.doneq <- true // inpulse
	d.doneq <- true // execpulse
	return nil, nil
}

// GetSpec for gRPC
func (d *Dean) GetSpec(ctx context.Context, in *pb.Empty) (*pb.KhanSpec, error) {
	d.log.Log(nil, "--->getspec %v\n", *in)
	return &pb.KhanSpec{Prog: d.spec, Err: crux.Err2Proto(nil)}, nil
}

// SetSpec for gRPC
func (d *Dean) SetSpec(ctx context.Context, in *pb.KhanSpec) (*pb.KhanResponse, error) {
	d.log.Log(nil, "--->setspec %v\n", *in)
	var ret pb.KhanResponse
	if in.Prog != d.spec {
		d.spec = in.Prog
		er := d.kv.Put(`khan/spec`, d.spec)
		// TBD deal with compile errors from new spec
		ret.Err = crux.Err2Proto(er)
	}
	return &ret, nil
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (d *Dean) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	d.log.Log(nil, "--->ping %v\n", *ping)
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Dean1_0 is the low-level khan.
// this has to cover both restarting and initialisation.
// for now, we don't consider restart.
func Dean1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	nod = ReNOD(nod, DeanName, DeanAPI)
	d := Dean{
		doneq:    make(chan bool, 3), // coordinate this with activity in quit
		alive:    alive,
		network:  network,
		update:   time.Now().UTC(),
		kv:       kv.NewLocalKV(),
		log:      logger.With("focus", DeanRev),
		reeveapi: reeveapi,
		nod:      nod,
	}
	d.adm = &d
	d.act = &d
	d.me = (**network).GetNames().Node

	nid := ruck.StartDeanServer(&nod, DeanRev, nod.NodeName, 0, &d, quit, reeveapi)
	go Afib(alive, d.doneq, UUID, "", nid)
	go d.inpulse()
	go d.execpulse()
	logger.Log(nil, "dean starting: picketNID=%s  heartbeatNID=%s", picketNID.String(), heartbeatNID.String())

	d.openPicket()
	d.openHeartbeat()

	logger.Log(nil, "dean started: picketNID=%s  heartbeatNID=%s", picketNID.String(), heartbeatNID.String())

	return nid
}

func (d *Dean) openPicket() {
	psign := d.reeveapi.SelfSigner()
	x, err := ruck.ConnectPicket(picketNID, psign, d.log)
	d.picket = &x
	if err != nil {
		d.log.Log(nil, "dean ConnectPicket failed: %v", err)
		return
	}
	d.log.Log(nil, "dean openPicket succeeded!")
}

func (d *Dean) openHeartbeat() {
	hbsign := d.reeveapi.SelfSigner()
	x, err := ruck.ConnectHeartbeat(heartbeatNID, hbsign, d.log)
	d.heartbeat = &x
	if err != nil {
		d.log.Log(nil, "dean ConnectHeartbeat failed: %v", err)
		return
	}
	d.log.Log(nil, "dean openHeartbeat succeeded!")
}

// stroke the khan engine
func (d *Dean) execpulse() {
	beat := time.NewTicker(execCycle)
	for {
		d.log.Log("execpulse")
		select {
		case <-beat.C:
			d.log.Log(nil, "dean about to khan")
			active, who, err := khan.Khan(d.adm, d.kv, d.act, nil)
			d.log.Log(nil, "khan out: active=%v who=%s err=%v", active, who, err)
		case <-d.doneq:
			// we're exiting!!
			beat.Stop()
			return
		}
	}
}

// read new stuff
func (d *Dean) inpulse() {
	beat := time.NewTicker(inCycle)
	for {
		d.log.Log("inpulse")
		select {
		case <-beat.C:
			if d.heartbeat != nil {
				if hbr, err := (*d.heartbeat).Heartbeats(context.Background(), &pb.Empty{}); err == nil {
					d.absorb(hbr.List)
				}
			}
			d.log.Log(nil, "dean inpulse")
		case <-d.doneq:
			// we're exiting!!
			beat.Stop()
			return
		}
	}
}

func (d *Dean) absorb(list []*pb.HeartbeatReq) {
	nmap := make(map[string]*horde.Service)
	for _, x := range list {
		if x.NID == "" {
			continue
		}
		et := crux.Timestamp2Time(x.Expires)
		if s, ok := nmap[x.UUID]; (!ok) || et.After(s.Expire) {
			nid, err := idutils.NetIDParse(x.NID)
			if err != nil {
				d.log.Log(nil, "nid(%s) parse error: %v", x.NID, err)
				continue
			}
			nmap[x.UUID] = &horde.Service{
				UniqueID: x.UUID,
				Name:     nid.ServiceRev,
				Node:     nid.Host,
				Addr:     nid.Host + ":" + nid.Port,
				Expire:   et,
				Stage:    horde.StageReady, // this should be computed from Stage TBD
			}
		}
	}
	if true {
		var x string
		for _, s := range nmap {
			x += fmt.Sprintf(" %s(%s %s)", s.UniqueID, s.Name, s.Addr)
		}
		d.log.Log(nil, "dean absorbed %d services:%s", len(nmap), x)
	} else {
		d.log.Log(nil, "dean absorbed %d services: %+v", len(nmap), nmap)
	}
	d.Lock()
	d.svcs = nmap
	d.Unlock()
}

// What reports the services in our horde
func (d *Dean) What() []horde.Service {
	var ret []horde.Service
	d.Lock()
	defer d.Unlock()
	for _, v := range d.svcs {
		ret = append(ret, *v)
	}
	return ret
}

// Start something on a node
func (d *Dean) Start(node, service string, count int) {
	var fr []*pb.StartReq
	// call picket on my node
	for i := 0; i < count; i++ {
		// call picket to start service
		d.log.Log(nil, "@@start %s on %s", service, node)
		fr = append(fr, &pb.StartReq{Filename: "@", Funcname: service, Seq: 99})
	}
	freq := pb.StartRequest{Reqs: fr}
	frep, errr := (*d.picket).StartFiles(context.Background(), &freq)
	d.log.Log(nil, "dean.picket_start(%+v)=%+v  err=%v", fr, frep, errr)
	// error TBD
}

// Start1 something on a node
func (d *Dean) Start1(node, service, addr string) {
	d.log.Log(nil, "@@start1 %s on %s\n", service, node)
}

// Stop something on a node
func (d *Dean) Stop(node, service string, count int) {
	// call steward to find picket on node node
	for i := 0; i < count; i++ {
		// call picket to stop service
		d.log.Log(nil, "@@stop %s on %s\n", service, node)
	}
}

// Reset resets the action
func (d *Dean) Reset() {
	// nada
}

// these routines implement the Administer interface for our horde of one node

// UniqueID is our "horde" name
func (d *Dean) UniqueID() string {
	return "H1_" + d.me
}

// Description for our horde
func (d *Dean) Description() string {
	return "horde of one " + d.me
}

// RegisterNode registers our node
func (d *Dean) RegisterNode(name string, tags []string) *crux.Err {
	// no need to implement this
	return nil
}

// Nodes lists the nodes in a horde
func (d *Dean) Nodes() ([]horde.Node, *crux.Err) {
	return []horde.Node{{Name: d.me, Tags: nil}}, nil
}
