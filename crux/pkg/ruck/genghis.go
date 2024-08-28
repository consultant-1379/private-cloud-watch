// Copyright 2019 Us
//
// Package Genghis is a bloc-wide horde administrator accessed through a gRPC interface:
//
//	service Genghis {
//		rpc ClientHorde(ClientHordeReq) returns (HordeReply) {}
//		rpc AllocHorde(AllocHordeReq) returns (AllocHordeReply) {}
//		rpc UnAllocHorde(ClientHordeReq) returns (AllocHordeReply) {}
//		rpc Navail(Empty) returns (NavailReply) {}
//		rpc Flock(FlockPost) returns (Empty) {}
//		rpc PingTest (Ping) returns (Ping) {}
//	 	rpc Quit(QuitReq) returns (QuitReply) {}
//	}
//
// The PingTest and Quit methods implement the normal ping test and quit methods.
//
// The set of nodes Genghis manages are updated solely by the Flock method. This simply documets what nodes
// are available, regardless of how they are assigned to hordes. This is normally done through the flocking code.
//
//	message FlockPost {
//		string name = 1;
//		repeated string nodes = 2;
//	}
//
// Hordes are allocated by the AllocHorde method. It takes an owner, the horde name and the number of nodes.
// The number of nodes allocated must be at least 1; you can determine the actual number by the ClientHorde method.
// Hordes can be deallocated by the UnAllocHorde method.
//
// Nodes are allocated to a horde indirectly; at some point, they simply know (via the Confab.GetNames() method) that
// they belong to the given horde. It is up to them to organise after that.
//
//	message AllocHordeReq {
//		string who = 1;
//		string name = 2;
//		int32 n = 3;
//		Err err = 4;
//	}
//	message AllocHordeReply {
//		Err err = 1;
//	}
//	message ClientHordeReq {
//		string horde = 1;
//	}
//
// You can find out about a specific horde by the ClientHorde method. It returns the owner, the actual nodes assigned,
// the number of nodes desired, and when it was allocated. Note that this is not a static assessment of a horde;
// for example, if we ask for a 10 node horde, and get only 6, it may well be that a little later, 4 more nodes are
// available and they will be added to the horde in the manner described above.
//
//	message HordeReply {
//		string name = 1;
//		repeated string nodes = 2;
//		int32 want = 3;
//		Timestamp start = 4;
//		string req = 5;
//		Err err = 6;
//	}
//
// You can find out how many nodes are available for allocation through the Navail method. Note that there is no interlocking
// between the Navail method and horde allocation and so even if Navail says that 50 (say) nodes are available, there may be much
// less than that when a subsequent AllocHorde request is serviced.
//
//	message NavailReply {
//		int32 total = 1;
//		int32 avail = 2;
//	}

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
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// reeve-related specs
const (
	GenghisName = "Genghis"
	GenghisAPI  = "Genghis1"
	GenghisRev  = "Genghis1_0"

	NodeTimeout = 10 * time.Minute
)

// Hordelet is a real horde (avoiding nameing clashes)
type Hordelet struct {
	Name  string
	Nodes []string
	Nwant int
	Nhave int
	ReqT  time.Time
	ReqN  string
}

// Lazaretto is how we partition nodes
type Lazaretto struct {
	sync.Mutex
	doneq chan bool
	all   map[string]time.Time
	list  []Hordelet
	log   clog.Logger
	used  int
}

// Quit for gRPC
func (l *Lazaretto) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	l.log.Log(nil, "--->quit %v\n", *in)
	l.doneq <- true // Afib
	l.doneq <- true // expire
	return nil, nil
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
func (l *Lazaretto) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	l.log.Log(nil, "--->ping %v\n", *ping)
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error") // why this error? TBD
}

// Navail returns number of nodes being managed
func (l *Lazaretto) Navail(ctx context.Context, in *pb.Empty) (*pb.NavailReply, error) {
	var ret pb.NavailReply
	l.Lock()
	l.recount()
	ret.Total = int32(len(l.all))
	ret.Avail = ret.Total - int32(l.used)
	l.Unlock()
	l.log.Log(nil, "--->navail returns %+v", ret)
	return &ret, nil
}

// Flock is how you register nodes
func (l *Lazaretto) Flock(ctx context.Context, in *pb.FlockPost) (*pb.Empty, error) {
	l.Lock()
	defer l.Unlock()
	t := time.Now().UTC()
	for _, n := range in.Nodes {
		l.all[n] = t
	}
	return &pb.Empty{}, nil
}

// ClientHorde returns information about my node
func (l *Lazaretto) ClientHorde(ctx context.Context, in *pb.ClientHordeReq) (*pb.HordeReply, error) {
	var ret pb.HordeReply
	for _, h := range l.list {
		for _, n := range h.Nodes {
			if n == in.Horde {
				ret.Name = h.Name
				for _, k := range h.Nodes {
					ret.Nodes = append(ret.Nodes, k)
				}
				ret.Want = int32(h.Nwant)
				ret.Start = crux.Time2Timestamp(h.ReqT)
				ret.Req = h.ReqN
				l.log.Log(nil, "--->myhorde(%s) returns %+v", in.Horde, ret)
				return &ret, nil
			}
		}
	}
	crux.Err2Proto(crux.ErrF("node %s not assigned to any horde", in.Horde))
	l.log.Log(nil, "--->myhorde(%s) returns %+v", in.Horde, ret)
	return &ret, nil
}

// AllocHorde is how you register hordes
func (l *Lazaretto) AllocHorde(ctx context.Context, in *pb.AllocHordeReq) (*pb.AllocHordeReply, error) {
	l.Lock()
	defer l.Unlock()
	if int(in.N) < (len(l.all) - l.used) {
		str := fmt.Sprintf("%s: wanted %d nodes, but only %d available", in.Name, in.N, (len(l.all) - l.used))
		l.log.Log(str)
		return &pb.AllocHordeReply{Err: crux.Err2Proto(crux.ErrS(str))}, nil
	}
	h := Hordelet{Name: in.Name, Nwant: int(in.N), Nhave: int(in.N), ReqT: time.Now().UTC(), ReqN: in.Who}
	// this next part is grisly, but how often will we do it?
	used := make(map[string]bool)
	for _, hh := range l.list {
		for _, n := range hh.Nodes {
			used[n] = true
		}
	}
	// set subtraction almost
	var i int
	for k := range l.all {
		if _, ok := used[k]; !ok {
			h.Nodes = append(h.Nodes, k)
			i++
			if i == h.Nwant {
				break
			}
		}
	}
	l.list = append(l.list, h)
	l.log.Log(nil, "allochorde returns %+v", h)
	return &pb.AllocHordeReply{}, nil
}

// UnAllocHorde returns information about my node
func (l *Lazaretto) UnAllocHorde(ctx context.Context, in *pb.ClientHordeReq) (*pb.AllocHordeReply, error) {
	var ret pb.AllocHordeReply
	//	l.log.Log(nil, "--->myhorde(%s) returns %+v", in.Node, ret)
	return &ret, nil
}

// Genghis1_0 is how we manage the hordes.
// FIXME: reeveapi should be a rucklib.ReeveAPI interface, but we can't do that until the code generator for StartGenghisServer(...) uses the same iface the function's reeveapi  arg.
func Genghis1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	l := Lazaretto{
		doneq: make(chan bool, 2), // coordinate this with activity in quit
		all:   make(map[string]time.Time),
		log:   logger.With("focus", GenghisRev),
	}

	node := (**network).GetNames().Node
	nod = ReNOD(nod, GenghisName, GenghisAPI)
	nid := ruck.StartGenghisServer(&nod, GenghisRev, node, 0, &l, quit, reeveapi)
	go Afib(alive, l.doneq, UUID, "", nid)
	go l.expire()
	logger.Log(nil, "genghis started")

	return nid
}

func (l *Lazaretto) expire() {
	clock := time.NewTimer(NodeTimeout / 2)
	for {
		select {
		case <-clock.C:
			l.Lock()
			expire := time.Now().UTC().Add(-NodeTimeout)
			for k, t := range l.all {
				if t.Before(expire) {
					delete(l.all, k)
				}
			}
			l.recount()
			l.Unlock()
		case <-l.doneq:
			clock.Stop()
			return
		}
	}
}

/*
	this is a mess.
	go thru and recount everything.
	but what happens when a horde drops below its count?

	WARNING: this routine must be protected by Lock/Unlock
*/
func (l *Lazaretto) recount() {
	l.used = 0
	for _, h := range l.list {
		var nn int
		for _, n := range h.Nodes {
			if _, ok := l.all[n]; ok {
				nn++
			}
		}
		h.Nhave = nn
		l.used += nn
		if nn != h.Nwant {
			l.log.Log(nil, "horde %s: want(%d) != have(%d)", h.Name, h.Nwant, h.Nhave)
			// presumably, do something here TBD
		}
	}
}
