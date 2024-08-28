package ruck

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	pb "github.com/erixzone/crux/gen/cruxgen"
	ruck "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/flock"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/x509ca"
)

// naming constats for flock as a service
const (
	FlockName = "Flock"
	FlockAPI  = "Flock1"
	FlockRev  = "Flock1_0"
)

func newFlock(port int, key, name, ip, beacon, networks, certdir string, visitor bool) *flock.Flock {
	kmon := make(chan crux.MonInfo, 1000)
	fmt.Printf("ruck.newFlock called with: port=%d name=%s ip=%s key=%s beacon=%s networks=%s certdir=%s\n", port, name, ip, key, beacon, networks, certdir)

	vOpts, priv, cert, err1 := x509ca.ReadCerts(certdir)
	crux.FatalIfErr(nil, crux.ErrE(err1))
	if name == "" {
		name = cert.Subject.CommonName
		fmt.Printf("ruck.newFlock sets name=%s from cert\n", name)
	}
	if ip == "" {
		ip = name
		fmt.Printf("ruck.newFlock sets ip=%s from name\n", ip)
	}

	uflock, unode, err := flock.NewUDP(port, name, ip, networks, kmon)
	crux.FatalIfErr(nil, err)

	err1 = unode.KeyInit(vOpts, priv, cert)
	crux.FatalIfErr(nil, crux.ErrE(err1))
	unode.GetCertificate().PromStartTLS()

	kk, _ := flock.String2Key(key)
	go ksend(beacon, kk, kmon) // send results outbound
	return flock.NewFlockNode(flock.NodeID{Moniker: name}, unode, uflock, &kk, beacon, visitor)
}

func ksend(beacon string, key flock.Key, m chan crux.MonInfo) {
	beaconip, port1x, err1 := net.SplitHostPort(beacon)
	crux.FatalIfErr(nil, crux.ErrE(err1))
	fmt.Printf("bip=%s port=%s ch=%v\n", beaconip, port1x, m)
	port, _ := strconv.Atoi(port1x)
	k, err := flock.NewUDPX(port, beaconip, key, false)
	crux.FatalIfErr(nil, err)

	for {
		mi := <-m
		//fmt.Printf("ksend(%+v)\n", mi)
		k.Send(&mi, &key)
	}
}

// Pod returns support for a gRPC interface
type Pod struct {
	sync.Mutex
	f     *flock.Flock
	doneq chan bool
}

// Flock1_0 is this release.
func Flock1_0(f *flock.Flock, quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, log clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	log.Log(nil, "starting %s nod=%+v", FlockName, nod)
	p := NewFlockServer(f)
	nod = ReNOD(nod, FlockName, FlockAPI)
	nid := ruck.StartFlockServer(&nod, FlockRev, nod.NodeName, 0, p, quit, reeveapi)
	go Afib(alive, p.doneq, UUID, "", nid)
	log.Log(nil, "started %s", FlockName)
	return nid
}

// NewFlockServer returns a pod
func NewFlockServer(f *flock.Flock) *Pod {
	p := Pod{f: f, doneq: make(chan bool)}
	return &p
}

// Nodes returns the heartbeats
func (p *Pod) Nodes(ctx context.Context, in *pb.Empty) (*pb.NodeReply, error) {
	p.Lock()
	mems := p.f.Mem()
	p.Unlock()
	var ret []string
	for _, n := range mems {
		ret = append(ret, n.Moniker)
	}
	return &pb.NodeReply{Nodes: ret}, nil
}
