package khan

import (
	"fmt"
	"sort"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/horde"
)

type KhanKhan struct {
	h *horde.Horde
}

func init() {
	Suite(&KhanKhan{})
}

type act struct {
	service string
	node    string
	count   int
}

type fakeAct struct {
	start []act
	stop  []act
}

var acts fakeAct

func (k *KhanKhan) SetUpSuite(c *C) {
	var err error
	k.h, err = horde.GetHorde("", "crux,local", "khan unit test")
	acts = fakeAct{}
	acts.Reset()
	k.h.Act = &acts
	c.Assert(err, IsNil)
	genCluster(c, k.h.Adm)
	makesvc(c, k.h.Act, "segp", "kv0", "a1")
	makesvc(c, k.h.Act, "segp", "kv1", "a2")
	makesvc(c, k.h.Act, "segp", "kv2", "a3")
	makesvc(c, k.h.Act, "segp", "kv2", "a4")
	makesvc(c, k.h.Act, "s3", "node1", "a5")
	makesvc(c, k.h.Act, "s3", "node2", "a6")
	fmt.Printf("+=+= %+v\n", k.h.Act.What())
}

func (k *KhanKhan) TearDownSuite(c *C) {
	// nothing
}

func (k *KhanKhan) TestKahn(c *C) {
	er := k.h.KV.Put("khan/spec", `sp := pick(segp, SIZE(ALL)-2, ALL)
	s3 := pick(s3, 3, !LABEL(KV))
	pick(poot, SIZE(s3&ALL)/2, s3&sp)
	`)
	c.Assert(er, IsNil)
	//	k.kv.Put("/khan/hent", "2")

	active, who, err := Khan(k.h.Adm, k.h.KV, k.h.Act, nil)
	c.Assert(err, IsNil)
	c.Assert(active, Equals, true)
	whoShould := []string{"s3", "poot", "segp"}
	sort.Strings(whoShould)
	sort.Strings(who)
	assertSameSlice(c, whoShould, who)
	dumpKV(c, k.h.KV)
	// finally verify correct things got started and stopped
	// should stop 1 segp on kv2
	xx := []act{{"segp", "kv2", 1}}
	c.Assert(acts.stop, DeepEquals, xx)
	// get a bunch of info about started
	ns3, s3Noded, nsegp, segpNoded := whack(acts.start)
	// verify 1 segp started on node*
	c.Assert(ns3, Equals, 1)
	c.Assert(s3Noded, Equals, true)
	// verify 2 segps started on node*
	c.Assert(nsegp, Equals, 3)
	c.Assert(segpNoded, Equals, true)
	// dumpKV(c, k.h)
}

func whack(start []act) (ns3 int, s3Noded bool, nsegp int, segpNoded bool) {
	s3Noded = true
	segpNoded = true
	for _, a := range start {
		if a.service == "s3" {
			ns3++
			s3Noded = s3Noded && (len(a.node) > 4) && (a.node[:4] == "node")
		}
		if a.service == "segp" {
			nsegp++
			segpNoded = segpNoded && (len(a.node) > 4) && (a.node[:4] == "node")
		}
	}
	s3Noded = s3Noded && (ns3 > 0)
	segpNoded = segpNoded && (nsegp > 0)
	fmt.Printf("ns3=%d s3n=%v nsegp=%d segpn=%v\n", ns3, s3Noded, nsegp, segpNoded)
	return
}

func (f *fakeAct) Start(node, service string, count int) {
	fmt.Printf("start %s %s %d\n", node, service, count)
	f.start = append(f.start, act{service, node, count})
}
func (f *fakeAct) Start1(node, service, addr string) {
	fmt.Printf("start1 %s %s %s\n", node, service, addr)
	f.start = append(f.start, act{service, node, 1})
}
func (f *fakeAct) Stop(node, service string, count int) {
	fmt.Printf("stop %s %s %d\n", node, service, count)
	f.stop = append(f.stop, act{service, node, count})
}
func (f *fakeAct) What() []horde.Service {
	there := make(map[string]*act)
	for _, a := range f.start {
		srv := a.node + "+" + a.service
		if t, ok := there[srv]; ok {
			t.count += a.count
		} else {
			there[srv] = &act{node: a.node, service: a.service, count: a.count}
		}
	}
	for _, a := range f.stop {
		srv := a.node + "+" + a.service
		if t, ok := there[srv]; ok {
			t.count -= a.count
		}
	}
	var ret []horde.Service
	for _, a := range there {
		for c := 0; c < a.count; c++ {
			ret = append(ret, horde.Service{Node: a.node, Name: a.service})
		}
	}
	fmt.Printf("what -> %+v\n", ret)
	return ret
}
func (f *fakeAct) Reset() {
	f.start = make([]act, 0)
	f.stop = make([]act, 0)
}
