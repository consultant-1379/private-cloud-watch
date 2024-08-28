package ruck

/*
	gather some random statistics!
*/

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// name constants for this service
const (
	MetricName = "Metric"
	MetricAPI  = "Metric1"
	MetricRev  = "Metric1_0"
)

var (
	cpuTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_temperature_celsius",
		Help: "Current temperature of the CPU.",
	})
	hdFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hd_errors_total",
			Help: "Number of hard-disk errors.",
		},
		[]string{"device"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(cpuTemp)
	prometheus.MustRegister(hdFailures)
}

// MetricServer - implement srv_heartbeat.proto
type MetricServer struct {
	sync.Mutex
	inb   <-chan []pb.HeartbeatReq
	alarm *time.Ticker
	tsent time.Time
	doneq chan bool
	log   clog.Logger
	out   chan crux.MonInfo
	nod   idutils.NodeIDT
}

// NewMetricServer  - get one.
func NewMetricServer(inb <-chan []pb.HeartbeatReq, lg clog.Logger, nod idutils.NodeIDT, reeveapi *reeve.StateT, out chan crux.MonInfo) *MetricServer {
	hs := &MetricServer{
		inb:   inb,
		alarm: time.NewTicker(HeartGap),
		tsent: time.Now().UTC(),
		doneq: make(chan bool, 2),
		log:   lg.With("focus", "heartbeatserver"),
		out:   out,
		nod:   nod,
	}
	return hs
}

// Quit for gRPC
func (hcs *MetricServer) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	hcs.log.Log(nil, "--->heartbeat quit %v\n", *in)
	hcs.doneq <- true // sendo
	hcs.doneq <- true // Afib
	return &pb.QuitReply{Message: ""}, nil
}

func (hcs *MetricServer) sendo() {
	time.Sleep(1 * time.Second)
	hdFailures.With(prometheus.Labels{"device": "/dev/sda"}).Inc()
	epoch := time.Now().UTC()
	timer := time.NewTicker(100 * time.Millisecond)
	for {
		hcs.log.Log(nil, "---firingx")
		select {
		case <-hcs.doneq:
			timer.Stop()
			return
		case <-timer.C:
			t := time.Now().UTC().Sub(epoch)
			// constants below designed to give us a nice stream of measurements
			x := 60.0 + 20*math.Sin(float64(t)) + 5*rand.Float64()
			hcs.log.Log(nil, "cpuTemp(%.2f)", x)
			cpuTemp.Set(float64(x))
		}
	}
}

// Metric1_0 is this release.
func Metric1_0(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, log clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	nod = ReNOD(nod, MetricName, MetricAPI)
	log.Log(nil, "starting %s nod=%+v", MetricName, nod)
	hb := NewMetricServer(heartChan, log, nod, reeveapi, (**network).Monitor())
	nid := idutils.NetIDT{}
	//nid := ruck.StartMetricServer(&nod, MetricRev, nod.NodeName, 0, hb, quit, reeveapi)
	go Afib(alive, hb.doneq, UUID, "", nid)
	go hb.sendo()
	log.Log(nil, "ending %s", MetricName)
	return nid
}
