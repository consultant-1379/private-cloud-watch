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
	rl "github.com/erixzone/crux/pkg/rucklib"
)

// reeve-related specs
const (
	ProctorName = "Proctor"
	ProctorAPI  = "Proctor1"
	ProctorRev  = "Proctor1_0"
	// for now, import this from dean: execCycle   = 10 * time.Second
	// for now, import this from dean: inCycle     = 1 * time.Second
	proctorExpire = 5 * time.Second
)

// Proctor is our base type for describing the horde
type Proctor struct {
	sync.Mutex
	doneq    chan bool
	alive    chan<- []pb.HeartbeatReq
	network  **crux.Confab
	update   time.Time
	kv       kv.KV
	adm      horde.Administer
	act      horde.Action
	spec     string
	svcs     map[string]*horde.Service
	pending  []horde.Service
	log      clog.Logger
	me       string
	picket   map[string]*pb.PicketClient
	hc       *pb.HealthCheckClient
	reeveapi *reeve.StateT
	nod      idutils.NodeIDT
	nodes    []string
	tnodes   time.Time
}

// Quit for gRPC
func (p *Proctor) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	p.log.Log(nil, "--->proctor quit %v\n", *in)
	p.doneq <- true // Afib
	p.doneq <- true // inpulse
	p.doneq <- true // execpulse
	return nil, nil
}

// GetSpec for gRPC
func (p *Proctor) GetSpec(ctx context.Context, in *pb.Empty) (*pb.KhanSpec, error) {
	p.log.Log(nil, "--->proctor getspec %v\n", *in)
	return &pb.KhanSpec{Prog: p.spec, Err: crux.Err2Proto(nil)}, nil
}

// SetSpec for gRPC
func (p *Proctor) SetSpec(ctx context.Context, in *pb.KhanSpec) (*pb.KhanResponse, error) {
	p.log.Log(nil, "--->proctor setspec %v\n", *in)
	var ret pb.KhanResponse
	if in.Prog != p.spec {
		p.spec = in.Prog
		er := p.kv.Put(`khan/spec`, p.spec)
		// TBD deal with compile errors from new spec
		ret.Err = crux.Err2Proto(er)
	}
	return &ret, nil
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (p *Proctor) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	p.log.Log(nil, "--->proctor ping %v\n", *ping)
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Proctor1_0 is the low-level khan.
// this has to cover both restarting and initialisation.
// for now, we don't consider restart.
func Proctor1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	nod = ReNOD(nod, ProctorName, ProctorAPI)
	p := Proctor{
		doneq:    make(chan bool, 3), // coordinate this with activity in quit
		alive:    alive,
		network:  network,
		update:   time.Now().UTC(),
		kv:       kv.NewLocalKV(),
		log:      logger.With("focus", ProctorRev),
		reeveapi: reeveapi,
		nod:      nod,
		pending:  make([]horde.Service, 0),
		picket:   make(map[string]*pb.PicketClient, 0),
		tnodes:   time.Now().UTC().Add(-time.Second),
	}
	p.log.Log(nil, "proctor.nod = %s", p.nod.String())
	p.adm = &p
	p.act = &p
	p.me = (**network).GetNames().Node

	nid := ruck.StartProctorServer(&nod, ProctorRev, nod.NodeName, 0, &p, quit, reeveapi)
	go Afib(alive, p.doneq, UUID, "", nid)
	go p.inpulse()
	go p.execpulse()
	logger.Log(nil, "proctor starting: picketNID=%s  heartbeatNID=%s", picketNID.String(), heartbeatNID.String())

	p.openHealthCheck()

	logger.Log(nil, "proctor started: picketNID=%s  heartbeatNID=%s", picketNID.String(), heartbeatNID.String())

	return nid
}

func (p *Proctor) openHealthCheck() {
	hbNOD, err := idutils.NewNodeID(p.nod.BlocName, p.nod.HordeName, p.nod.NodeName, HealthCheckName, HealthCheckAPI)
	if err != nil {
		p.log.Log(nil, "openHealthCheck: hbNOD failed: %v", err)
		return
	}
	hcNID, hcsign, err := rl.Get1Endpoint(hbNOD, HealthCheckRev, p.reeveapi)
	if err != nil {
		p.log.Log(nil, "openHealthCheck: Get1Endpoint failed: %v", err)
		return
	}

	x, err := ruck.ConnectHealthCheck(hcNID, hcsign, p.log)
	if err != nil {
		p.log.Log(nil, "openHealthCheck: ConnectHealthCheck failed: %v", err)
		return
	}
	p.hc = &x
	p.log.Log(nil, "openHealthCheck succeeded!")
}

// stroke the khan engine
func (p *Proctor) execpulse() {
	beat := time.NewTicker(execCycle)
	for {
		p.log.Log("execpulse")
		select {
		case <-beat.C:
			p.prune()
			p.log.Log(nil, "proctor about to khan (pending=%v)", p.pending)
			active, who, err := khan.Khan(p.adm, p.kv, p.act, p.pending)
			p.log.Log(nil, "khan out: active=%v who=%s err=%v", active, who, err)
		case <-p.doneq:
			// we're exiting!!
			beat.Stop()
			return
		}
	}
}

// read new stuff
func (p *Proctor) inpulse() {
	beat := time.NewTicker(inCycle)
	for {
		p.log.Log("inpulse")
		select {
		case <-beat.C:
			var x int
			if p.hc != nil {
				if hbr, err := (*p.hc).Heartbeats(context.Background(), &pb.Empty{}); err == nil {
					x = len(hbr.List)
					p.absorb(hbr.List)
				}
			}
			p.log.Log(nil, "proctor(%p) inpulse %d items", p.hc, x)
		case <-p.doneq:
			// we're exiting!!
			beat.Stop()
			return
		}
	}
}

func (p *Proctor) absorb(list []*pb.HeartbeatReq) {
	p.Lock()
	defer p.Unlock()
	nmap := make(map[string]*horde.Service)
	for _, x := range list {
		if x.NID == "" {
			continue
		}
		et := crux.Timestamp2Time(x.Expires)
		if s, ok := nmap[x.UUID]; (!ok) || et.After(s.Expire) {
			nid, err := idutils.NetIDParse(x.NID)
			if err != nil {
				p.log.Log(nil, "nid(%s) parse error: %v", x.NID, err)
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
		p.prune1(x.UUID)
	}
	if true {
		var x string
		for _, s := range nmap {
			x += fmt.Sprintf(" %s(%s %s)", s.UniqueID, s.Name, s.Addr)
		}
		p.log.Log(nil, "proctor absorbed %d services:%s", len(nmap), x)
	} else {
		p.log.Log(nil, "proctor absorbed %d services: %+v", len(nmap), nmap)
	}
	p.svcs = nmap
}

func (p *Proctor) prune() {
	p.Lock()
	defer p.Unlock()
	now := time.Now().UTC()
	var i, j int
	// i is where we copy to
	// j is where we copy from
	// find first expire
	for i = range p.pending {
		if !p.pending[i].Expire.After(now) {
			j = i
			break
		}
	}
	for j++; j < len(p.pending); j++ {
		if p.pending[j].Expire.After(now) {
			p.pending[i] = p.pending[j]
			i++
		}
	}
	p.pending = p.pending[:i]
}

func (p *Proctor) prune1(uuid string) {
	p.log.Log(nil, "prune1(%s)", uuid)
	for i := range p.pending {
		if p.pending[i].UniqueID == uuid {
			if i < len(p.pending)-1 {
				copy(p.pending[i:], p.pending[i+1:])
			}
			p.pending = p.pending[:len(p.pending)-1]
			p.log.Log(nil, "prune1.pending: %+v", p.pending)
			return
		}
	}
	p.log.Log("prune1(%s) failed", uuid)
}

// What reports the services in our horde
func (p *Proctor) What() []horde.Service {
	var ret []horde.Service
	p.Lock()
	defer p.Unlock()
	for _, v := range p.svcs {
		ret = append(ret, *v)
	}
	return ret
}

func (p *Proctor) openPicket(node string) *pb.PicketClient {
	p.log.Log(nil, "proctor.openPicket.nod = %s", p.nod.String())
	epl, err := rl.AllEndpoints(p.nod, ProctorRev, p.reeveapi)
	if err != nil {
		p.log.Log(nil, "allendpoints failed: %s", err.String())
		return nil
	}
	p.log.Log(nil, "openPicket.allendpoints = %+v", epl)
	for _, e := range epl {
		nid, err := idutils.NetIDParse(e.Netid)
		if err != nil {
			p.log.Log(nil, "bad netid '%s': %s", e.Netid, err.String())
			continue
		}
		if nid.Host == node {
			psign, err := p.reeveapi.ClientSigner(PicketRev) // Signer for the Client side (does the signing on client calls out)
			if err != nil {
				p.log.Log(nil, "proctor.picket signer failed: %v", err)
				return nil
			}
			pc, err := ruck.ConnectPicket(nid, psign, p.log)
			if err != nil {
				p.log.Log(nil, "proctor.picket client failed: %v", err)
				return nil
			}
			p.log.Log(nil, "openPicket returns picket %p at %s", &pc, nid.String())
			return &pc
		}
	}
	p.log.Log(nil, "couldn't find picket on node %s", node)
	return nil
}

func (p *Proctor) getPicket(node string) *pb.PicketClient {
	if pp, ok := p.picket[node]; !ok || (pp == nil) {
		p.picket[node] = p.openPicket(node)
	}
	return p.picket[node]
}

// Start something on a node
func (p *Proctor) Start(node, service string, count int) {
	p.log.Log(nil, "proctor_startxxx=%s %s %d", node, service, count)
	pc := p.getPicket(node)
	if pc == nil {
		p.log.Log(nil, "getPicket=%p", pc)
	} else {
		p.log.Log(nil, "getPicket=%+v", *pc)
	}
	if pc == nil {
		return
	}
	var fr []*pb.StartReq
	// call picket on my node
	for i := 0; i < count; i++ {
		// call picket to start service
		p.log.Log(nil, "@@start %s on %s", service, node)
		fr = append(fr, &pb.StartReq{Filename: "@", Funcname: service, Seq: 99})
	}
	freq := pb.StartRequest{Reqs: fr}
	frep, errr := (*pc).StartFiles(context.Background(), &freq)
	p.log.Log(nil, "proctor.start(%+v)=%+v  err=%v", fr, frep, errr)
	if errr != nil {
		p.log.Log(nil, "proctor.start failed: %v", errr.Error())
		return
	}
	// now process return, setting up pending
	et := time.Now().UTC().Add(proctorExpire)
	for _, rp := range frep.Reqs {
		e := crux.Proto2Err(rp.Err)
		if e != nil {
			p.log.Log(nil, "picket(%s,%s) failed: %s", service, node, e.String())
			continue
		}
		// do we need to do fails by seq? TBD
		p.pending = append(p.pending, horde.Service{
			Name:     service,
			Node:     node,
			UniqueID: rp.UUID,
			Expire:   et,
		})
	}
	p.log.Log(nil, "pending now %+v", p.pending)
}

// Start1 something on a node
func (p *Proctor) Start1(node, service, addr string) {
	p.log.Log(nil, "@@start1 %s on %s at %s\n", service, node, addr)
	p.Start(node, service, 1)
}

// Stop something on a node
func (p *Proctor) Stop(node, service string, count int) {
	// call steward to find picket on node node
	for i := 0; i < count; i++ {
		// call picket to stop service
		p.log.Log(nil, "@@stop %s on %s\n", service, node)
	}
}

// Reset resets the action
func (p *Proctor) Reset() {
	// nada
}

// these routines implement the Administer interface for our horde

// UniqueID is our "horde" name
func (p *Proctor) UniqueID() string {
	return "H1_" + p.me
}

// Description for our horde
func (p *Proctor) Description() string {
	return "horde including " + p.me
}

// RegisterNode registers our node
func (p *Proctor) RegisterNode(name string, tags []string) *crux.Err {
	// no need to implement this
	return nil
}

// Nodes lists the nodes in a horde
func (p *Proctor) Nodes() ([]horde.Node, *crux.Err) {
	return []horde.Node{{Name: p.me, Tags: nil}}, nil
}
