package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

func main() {
	mflag := flag.Int("m", 0, "type of metric")
	tflag := flag.Int("t", 20, "length of data (in seconds)")
	flag.Parse()
	if flag.NArg() < 1 {
		//		if (flag.NArg() < 1) || (*mflag == 0) {
		fmt.Fprintf(os.Stderr, "usage: %s [-t tlim] -m metric ...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(127)
	}

	switch *mflag {
	case 1:
		if (flag.NArg() != 1) && (flag.NArg() != 4) {
			fmt.Fprintf(os.Stderr, "usage: %s [-t tlim] -m 1 nnodes [down downmean downstddev]\n", os.Args[0])
			flag.PrintDefaults()
			os.Exit(127)
		}
		i, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "expected a numeric number of nodes, not '%s'\n", flag.Arg(0))
			os.Exit(127)
		}
		var down, downmean, downstddev float64
		if flag.NArg() == 4 {
			read1 := func(nn int, str string) float64 {
				xx, errx := strconv.ParseFloat(flag.Arg(nn), 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "expected a numeric %s '%s': %s\n", str, flag.Arg(nn), errx.Error())
					os.Exit(127)
				}
				return xx
			}
			down = read1(1, "down fraction")
			downmean = read1(2, "down mean")
			downstddev = read1(3, "down std dev")
		}
		metric1(time.Now().UTC(), *tflag, int(i), down, downmean, downstddev)
	case 2:
		if (flag.NArg() != 1) && (flag.NArg() != 4) {
			fmt.Fprintf(os.Stderr, "usage: %s [-t tlim] -m 2 samplefile [down downmean downstddev]\n", os.Args[0])
			flag.PrintDefaults()
			os.Exit(127)
		}
		samp, err := ioutil.ReadFile(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "expected a sample file in %s: %s\n", flag.Arg(0), err.Error())
			os.Exit(127)
		}
		var down, downmean, downstddev float64
		if flag.NArg() == 4 {
			read1 := func(nn int, str string) float64 {
				xx, errx := strconv.ParseFloat(flag.Arg(nn), 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "expected a numeric %s '%s': %s\n", str, flag.Arg(nn), errx.Error())
					os.Exit(127)
				}
				return xx
			}
			down = read1(1, "down fraction")
			downmean = read1(2, "down mean")
			downstddev = read1(3, "down std dev")
		}
		metric2(time.Now().UTC(), *tflag, samp, down, downmean, downstddev)
	}

	os.Exit(0)
}

type node1 struct {
	ipv4 string
	name string
	tick time.Time
	down int
}

func vlen() float64 {
	// hand built emulator: 40% = 0ish, 40% 2ish, 20% .2-1.8
	f := rand.Float32()
	if f <= .4 {
		return 0 + 0.1*rand.Float64()
	}
	if f <= .8 {
		return 1.9 + 0.1*rand.Float64()
	}
	return .2 + 1.6*rand.Float64()
}

func (n node1) addset() []prompb.TimeSeries {
	var ret []prompb.TimeSeries
	stime := n.tick.UnixNano() / 1000000 // milliseconds
	stimei := fmt.Sprintf("%d", stime)
	stimes := time.Unix(stime/1000, (stime%1000)*1000000).Format(time.UnixDate)
	// echoing structure in observed file
	ts := prompb.TimeSeries{Labels: make([]prompb.Label, 0), Samples: make([]prompb.Sample, 0)}
	ts.Labels = append(ts.Labels, prompb.Label{Name: "IPv4", Value: n.ipv4})
	up := "up"
	if n.down > 0 {
		up = "down"
	}
	ts.Labels = append(ts.Labels, prompb.Label{Name: "__name__", Value: up})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_date", Value: stimes})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_sec", Value: stimei})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "instance", Value: n.name})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "job", Value: "fulcrum2"})
	ts.Samples = append(ts.Samples, prompb.Sample{Timestamp: 1564519908141})
	ret = append(ret, ts)

	ts = prompb.TimeSeries{Labels: make([]prompb.Label, 0), Samples: make([]prompb.Sample, 0)}
	ts.Labels = append(ts.Labels, prompb.Label{Name: "IPv4", Value: n.ipv4})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "__name__", Value: "scrape_duration_seconds"})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_date", Value: stimes})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_sec", Value: stimei})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "instance", Value: n.name})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "job", Value: "fulcrum2"})
	ts.Samples = append(ts.Samples, prompb.Sample{Value: vlen(), Timestamp: 1564519908141})
	ret = append(ret, ts)

	ts = prompb.TimeSeries{Labels: make([]prompb.Label, 0), Samples: make([]prompb.Sample, 0)}
	ts.Labels = append(ts.Labels, prompb.Label{Name: "IPv4", Value: n.ipv4})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "__name__", Value: "scrape_samples_scraped"})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_date", Value: stimes})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_sec", Value: stimei})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "instance", Value: n.name})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "job", Value: "fulcrum2"})
	ts.Samples = append(ts.Samples, prompb.Sample{Timestamp: 1564519908141})
	ret = append(ret, ts)

	ts = prompb.TimeSeries{Labels: make([]prompb.Label, 0), Samples: make([]prompb.Sample, 0)}
	ts.Labels = append(ts.Labels, prompb.Label{Name: "IPv4", Value: n.ipv4})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "__name__", Value: "scrape_samples_post_metric_relabeling"})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_date", Value: stimes})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "config_sec", Value: stimei})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "instance", Value: n.name})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "job", Value: "fulcrum2"})
	ts.Samples = append(ts.Samples, prompb.Sample{Timestamp: 1564519908141})
	ret = append(ret, ts)

	return ret
}

func metric1(start time.Time, tspan int, nnodes int, down, downmean, downstddev float64) {
	nodes := make([]node1, nnodes)
	for i := 0; i < nnodes; i++ {
		nodes[i] = node1{
			ipv4: fmt.Sprintf("172.18.0.%d", i),
			name: fmt.Sprintf("f%d:8090", nnodes-i),
			tick: start.Add(time.Duration(rand.Intn(100)) * time.Millisecond),
		}
	}

	// the grind begins now. step through time, spawning entries for each node
	stepv := 2
	for step := 0; step < tspan/stepv; step++ {
		wr := prompb.WriteRequest{}
		for i := 0; i < nnodes; i++ {
			// first do down stuff
			if (nodes[i].down == 0) && (rand.Float64() < down) {
				nodes[i].down = int(math.Ceil((rand.NormFloat64()*downstddev + downmean) / float64(stepv)))
			}
			wr.Timeseries = append(wr.Timeseries, nodes[i].addset()...)
			nodes[i].tick.Add(time.Duration(stepv) * time.Second)
			if nodes[i].down > 0 {
				nodes[i].down--
			}
		}
		b, err := json.Marshal(wr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "json encode error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("%s\n", b)
	}
}

type node2 struct {
	tenant string
	href   string
	name   string
	tick   time.Time
	down   int
	nnodes float64
}

func samp1x(samp []byte) prompb.WriteRequest {
	var ret prompb.WriteRequest
	i := 0
	n := len(samp)
	//	fmt.Printf("sampling %d .. %d\n", i, n)
	for i < n {
		// determine start of this timeseries
		k := strings.IndexByte(string(samp[i:n]), '{')
		if k == -1 {
			break
		}
		i += k
		//		fmt.Printf("new i = %d\n", i)
		// now find end of string
		p := i
		depth := 0
		for p < n {
			if samp[p] == '{' {
				depth++
			}
			if samp[p] == '}' {
				depth--
			}
			p++
			if depth == 0 {
				break
			}
		}
		//		fmt.Printf("looped: depth=%d p=%d n=%d\n", depth, p, n)
		// did we get a fully enclosed group?
		if depth == 0 {
			var x prompb.WriteRequest
			if err := json.Unmarshal(samp[i:p], &x); err != nil {
				fmt.Printf("unmarshal(%d..%d) = %c..%c\n", i, p, samp[i], samp[p])
				fmt.Fprintf(os.Stderr, "unmarshal error: %s\n", err.Error())
				os.Exit(1)
			}
			ret.Timeseries = append(ret.Timeseries, x.Timeseries...)
		}
		i = p
	}
	return ret
}

func samp1(samp []byte) []node2 {
	req := samp1x(samp)
	fmt.Fprintf(os.Stderr, "%d timeseries\n", len(req.Timeseries))
	// build two maps: one for tenants and one for vm's
	tmap := make(map[string]*node2, 0)
	vmap := make(map[string]*node2, 0)

	for i := range req.Timeseries {
		// convert to node2
		nod := new(node2)
		m := make(map[string]string, 0)
		for _, l := range req.Timeseries[i].Labels {
			m[l.Name] = l.Value
		}
		nod.href = m["instance"]
		nod.tenant = m["tenant"]
		if x, ok := m["name"]; ok {
			nod.name = x
			vmap[nod.tenant+"|"+nod.name] = nod
		} else {
			nod.name = nod.tenant
			tmap[nod.tenant] = nod
		}
		// ignore samples for now
	}
	// we have the tenants and vm's; build our list
	nnodes := make([]node2, 0)
	for ten, tnode := range tmap {
		// count the vm's
		for _, n := range vmap {
			if n.tenant == ten {
				tnode.nnodes++
			}
		}
		nnodes = append(nnodes, *tnode)
		for _, n := range vmap {
			if n.tenant == ten {
				nnodes = append(nnodes, *n)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "%d tenants, %d nodes\n", len(tmap), len(nnodes))
	return nnodes
}

func (n node2) addset() prompb.TimeSeries {
	var ret []prompb.TimeSeries
	stime := n.tick.UnixNano() / 1000000 // milliseconds

	// first do entry for the tenant
	if n.tenant == n.name {
		ts := prompb.TimeSeries{Labels: make([]prompb.Label, 0), Samples: make([]prompb.Sample, 0)}
		ts.Labels = append(ts.Labels, prompb.Label{Name: "__name__", Value: "consul"})
		ts.Labels = append(ts.Labels, prompb.Label{Name: "instance", Value: n.href})
		ts.Labels = append(ts.Labels, prompb.Label{Name: "job", Value: "consul"})
		ts.Labels = append(ts.Labels, prompb.Label{Name: "tenant", Value: n.tenant})
		ts.Samples = append(ts.Samples, prompb.Sample{Value: n.nnodes, Timestamp: stime})
		ret = append(ret, ts)
		return ts
	}
	// echoing structure in observed file
	ts := prompb.TimeSeries{Labels: make([]prompb.Label, 0), Samples: make([]prompb.Sample, 0)}
	up := float64(1)
	if n.down > 0 {
		up = 0
	}
	ts.Labels = append(ts.Labels, prompb.Label{Name: "__name__", Value: "status"})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "instance", Value: n.href})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "job", Value: "consul"})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "name", Value: n.name})
	ts.Labels = append(ts.Labels, prompb.Label{Name: "tenant", Value: n.tenant})
	ts.Samples = append(ts.Samples, prompb.Sample{Value: up, Timestamp: 1564519908141})
	return ts
}

func metric2(start time.Time, tspan int, samp []byte, down, downmean, downstddev float64) {
	nodes := samp1(samp)
	for i := 0; i < len(nodes); i++ {
		nodes[i].tick = start.Add(time.Duration(rand.Intn(100)) * time.Millisecond)
	}

	// the grind begins now. step through time, spawning entries for each node
	stepv := 120
	for step := 0; step < tspan/stepv; step++ {
		wr := prompb.WriteRequest{}
		for i := 0; i < len(nodes); i++ {
			// first do down stuff
			if (nodes[i].down == 0) && (rand.Float64() < down) {
				nodes[i].down = int(math.Ceil((rand.NormFloat64()*downstddev + downmean) / float64(stepv)))
			}
			wr.Timeseries = append(wr.Timeseries, nodes[i].addset())
			nodes[i].tick.Add(time.Duration(stepv) * time.Second)
			if nodes[i].down > 0 {
				nodes[i].down--
			}
		}
		b, err := json.Marshal(wr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "json encode error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("%s\n", b)
	}
}
