package ruck

import (
	"fmt"
	"plugin"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	ruck "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// PicketName is the current versioned name
const (
	PicketName = "Picket"
	PicketAPI  = "Picket1"
	PicketRev  = "Picket1_0"
)

// Mind holds the per-world data.
type Mind struct {
	quit    <-chan bool
	alive   chan<- []pb.HeartbeatReq
	network **crux.Confab
	fns     map[string]*crux.Fservice
	rstate  *reeve.StateT
	horde   string
	ipname  string
	UUID    string
	doneq   chan bool
	log     clog.Logger
	nod     idutils.NodeIDT
	eRev    string
	pmap    map[string]*plugin.Plugin
}

// NewMind returns you THE mind for this run. It takes a global shutdown channel and a network.
func NewMind(quit <-chan bool, network **crux.Confab, UUID string, log clog.Logger, nod idutils.NodeIDT, reeveapi *reeve.StateT) *Mind {
	mind := Mind{
		quit:    quit,
		alive:   heartChan,
		network: network,
		fns:     make(map[string]*crux.Fservice),
		rstate:  reeveapi,
		horde:   nod.HordeName,
		ipname:  nod.NodeName,
		doneq:   make(chan bool),
		log:     log,
		nod:     nod,
		pmap:    make(map[string]*plugin.Plugin),
	}
	// it is a little dangerous not to terminate the other end of alive, but it will be connected soon.
	return &mind
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (m *Mind) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Quit for gRPC
func (m *Mind) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	m.log.Log(nil, "--->picket quit %v", *in)
	m.doneq <- true
	return &pb.QuitReply{Message: ""}, nil
}

// StartFiles for gRPC
func (m *Mind) StartFiles(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	m.log.Log(nil, "--->startfiles %v", *in)
	seenErr := make(map[int]*crux.Err)
	funky := make(map[int]func(<-chan bool, chan<- []pb.HeartbeatReq, **crux.Confab, string, clog.Logger, idutils.NodeIDT, string, *reeve.StateT) idutils.NetIDT)

	m.log.Log(nil, "len=%d funcn=>%s<", len(in.Reqs), in.Reqs[0].Funcname)
	/*if (len(in.Reqs) == 1) && (in.Reqs[0].Funcname[0:8] == "Pastiche") {
		funky[0] = Pastiche0_1
	} else */{
		// this is fairly straight forward; accumulate the function calls, checking for errors.
		for rqi, rq := range in.Reqs {
			var err error
			image := rq.Filename
			if (image == "") || (!muster.init && image == "@") {
				image = DefaultExecutable
			}
			if image == "@" { // hopefully, this will be the common case
				if file, ok := muster.smap[rq.Funcname]; ok {
					image = file
				}
			}
			if image[0] != '/' {
				image = "/tmp/cache/" + image // call to pastiche here
			}
			if true {
				// so bogus; this avoids a plugin already loaded bug
				image = DefaultExecutable
			}
			m.log.Log(nil, "reading plugin from %s", image)
			p, ok := m.pmap[image]
			if !ok {
				p, err = plugin.Open(image)
				if err != nil {
					if seenErr[int(rq.Seq)] == nil {
						seenErr[int(rq.Seq)] = crux.ErrE(err)
					}
					continue
				}
				m.pmap[image] = p
			}
			f, err := p.Lookup(rq.Funcname)
			if err != nil {
				if seenErr[int(rq.Seq)] == nil {
					seenErr[int(rq.Seq)] = crux.ErrE(err)
				}
				continue
			}
			fn, ok := f.(func(<-chan bool, chan<- []pb.HeartbeatReq, **crux.Confab, string, clog.Logger, idutils.NodeIDT, string, *reeve.StateT) idutils.NetIDT)
			if ok {
				funky[rqi] = fn
			} else {
				if seenErr[int(rq.Seq)] == nil {
					seenErr[int(rq.Seq)] = crux.ErrF("function %s has the wrong type", rq.Funcname)
				}
				continue
			}
		}
	}
	m.log.Log(nil, "seenErr = %+v  in.Reqs=[[%+v]] hc=%+v alive=%+v", seenErr, in.Reqs, heartChan, m.alive)

	// done error analysis; now execute any that can be executed
	var reply []*pb.StartRep
	for rqi, rq := range in.Reqs {
		if seenErr[int(rq.Seq)] == nil {
			fr := crux.Fservice{FuncName: rq.Funcname, FileName: rq.Filename}
			m.log.Log(nil, "wtf fr=%+v  nod=%+v  reeveapi=%p", fr, m.nod, m.rstate)
			m.SetMember(&fr, m.nod.NodeName) // record it
			m.log.Log(nil, "fired off fr=%+v", fr)
			go funky[rqi](fr.Quit, fr.Alive, m.network, fr.UUID, m.log.With("focus", rq.Funcname), m.nod, m.eRev, m.rstate)
			reply = append(reply, &pb.StartRep{Seq: int32(rq.Seq), UUID: fr.UUID, Filename: fr.FileName, Funcname: fr.FuncName, Err: crux.Err2Proto(nil)})
		} else {
			reply = append(reply, &pb.StartRep{Seq: int32(rq.Seq), Err: crux.Err2Proto(seenErr[int(rq.Seq)])})
		}
		m.log.Log(nil, "reply is now: %+v", reply)
	}
	m.log.Log(nil, "leaving")

	return &pb.StartReply{Reqs: reply}, nil
}

// StopFiles for gRPC
func (m *Mind) StopFiles(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	m.log.Log(nil, "--->stopfiles %v", *in)
	reply := pb.StopReply{UUID: in.UUID}
	if fr, ok := m.fns[in.UUID]; ok {
		m.log.Log(nil, "sending quit to %+v", fr)
		fr.Quit <- true
	} else {
		reply.Err = crux.Err2Proto(crux.ErrF("no process with UUID=%s", in.UUID))
	}
	return &reply, nil
}

// AllFiles lists all running processes
func (m *Mind) AllFiles(ctx context.Context, xxx *pb.Empty) (*pb.StartReply, error) {
	m.log.Log(nil, "-->allfiles") // debugging; eliminate after a while TBD
	var reply []*pb.StartRep
	for _, fn := range m.fns {
		reply = append(reply, &pb.StartRep{UUID: fn.UUID, Funcname: fn.FuncName, Filename: fn.FileName, Start: crux.Time2Timestamp(fn.T)})
	}
	return &pb.StartReply{Reqs: reply}, nil
}

// SetMember does admin work for a service
func (m *Mind) SetMember(fr *crux.Fservice, nodeName string) *crux.Fservice {
	fr.UUID = crux.SmallID()
	fr.Alive = m.alive
	fr.Quit = make(chan bool, 2)
	fr.T = time.Now()
	m.fns[fr.UUID] = fr
	m.alive <- []pb.HeartbeatReq{{State: pb.ServiceState_NMAP, UUID: fr.UUID, Various: fmt.Sprintf("%s %s %s", nodeName, fr.FileName, fr.FuncName), At: crux.Time2Timestamp(fr.T)}}
	return fr
}

// Picket1_0 is this release.
func Picket1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	logger.Log(nil, "starting %s", PicketName)

	nod = ReNOD(nod, PicketName, PicketAPI)
	mind := NewMind(nil, network, UUID, logger, nod, reeveapi)
	// start up a picket server on us
	fr := crux.Fservice{FuncName: PicketName, FileName: ""}
	mind.SetMember(&fr, nod.NodeName)
	quit = fr.Quit // ignore this quit; use the one from fr
	mind.log.Log(nil, "picket started: %+v", fr)
	nid := ruck.StartPicketServer(&nod, PicketRev, nod.NodeName, 0, mind, quit, reeveapi)
	go Afib(alive, mind.doneq, fr.UUID, "", nid)
	picketNID = nid
	mind.log.Log(nil, "ending %s", PicketName)
	return nid
}
