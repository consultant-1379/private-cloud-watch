package khan

import (
	"sort"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/horde"
)

type KhanDiff struct {
	h *horde.Horde
}

func init() {
	Suite(&KhanDiff{})
}

func (k *KhanDiff) SetUpSuite(c *C) {
	var err error
	k.h, err = horde.GetHorde("", "crux,local", "diff unit test")
	c.Assert(err, IsNil)
	makesvc(c, k.h.Act, "segp", "kv0", "a1")
	makesvc(c, k.h.Act, "segp", "kv1", "a2")
	makesvc(c, k.h.Act, "segp", "kv2", "a3")
	makesvc(c, k.h.Act, "segp", "kv2", "a4")
	makesvc(c, k.h.Act, "s3", "node1", "a5")
	makesvc(c, k.h.Act, "s3", "node2", "a6")
	genCluster(c, k.h.Adm)
}

func (k *KhanDiff) TearDownSuite(c *C) {
	// nothing
}

const prog6 = `sp := pick(segp, 5, ALL)
	s3 := pick(s3, 3, !LABEL(KV))
	pick(poot, 2, s3&sp)
	`

func (k *KhanDiff) TestDiffPlain(c *C) {
	correct := []string{
		"segp[-1@kv2]",
		"s3[1@node0]",
		"segp[1@node0]",
		"poot[1@node1]",
		"segp[1@node1]",
		"poot[1@node2]",
	}
	explanation := []string{
		"start 1 instances of 'segp' on node0;  no constraints",
		"start 1 instances of 'segp' on node1;  no constraints",
		"start 1 instances of 's3' on node0;  no constraints",
		"start 1 instances of 'poot' on node2;  no constraints",
		"stop 1 instances of 'segp' on kv2; no constraints",
		"start 1 instances of 'poot' on node1;  no constraints",
	}
	k.testDiff(c, prog6, correct, explanation)
}

func (k *KhanDiff) TestDiffSeqNokay(c *C) {
	correct := []string{
		"segp[-1@kv2]",
		"s3[1@node0]",
		"segp[1@node0]",
		"poot[1@node1]",
		"segp[1@node1]",
		"poot[1@node2]",
	}
	explanation := []string{
		"start 1 instances of 'segp' on node0;  no constraints",
		"start 1 instances of 'segp' on node1;  no constraints",
		"start 1 instances of 's3' on node0;  no constraints",
		"start 1 instances of 'poot' on node2;  good[s3.ready: actual(2) >= target(2)]",
		"stop 1 instances of 'segp' on kv2; no constraints",
		"start 1 instances of 'poot' on node1;  good[s3.ready: actual(2) >= target(2)]",
	}
	k.testDiff(c, prog6+"start poot after 2 of s3\n", correct, explanation)
}

func (k *KhanDiff) TestDiffSeqNfail(c *C) {
	correct := []string{
		"segp[-1@kv2]",
		"s3[1@node0]",
		"segp[1@node0]",
		"segp[1@node1]",
	}
	explanation := []string{
		"start 1 instances of 'segp' on node0;  no constraints",
		"start 1 instances of 'segp' on node1;  no constraints",
		"start 1 instances of 's3' on node0;  no constraints",
		"start no instances of 'poot' on node2;  bad[s3.ready: actual(2) < target(3)]",
		"stop 1 instances of 'segp' on kv2; no constraints",
		"start no instances of 'poot' on node1;  bad[s3.ready: actual(2) < target(3)]",
	}
	k.testDiff(c, prog6+"start poot after 3 of s3\n", correct, explanation)
}

func (k *KhanDiff) TestDiffSeqPercentFail(c *C) {
	correct := []string{
		"segp[-1@kv2]",
		"s3[1@node0]",
		"segp[1@node0]",
		"segp[1@node1]",
	}
	explanation := []string{
		"start 1 instances of 'segp' on node0;  no constraints",
		"start 1 instances of 'segp' on node1;  no constraints",
		"start 1 instances of 's3' on node0;  no constraints",
		"start no instances of 'poot' on node2;  bad[s3.ready: actual(66.7%) < target(70.0%)]",
		"stop 1 instances of 'segp' on kv2; no constraints",
		"start no instances of 'poot' on node1;  bad[s3.ready: actual(66.7%) < target(70.0%)]",
	}
	k.testDiff(c, prog6+"start poot after 70% of s3\n", correct, explanation)
}

func (k *KhanDiff) testDiff(c *C, prog string, correct []string, explanation []string) {
	delta, whatever, who, expl, err := diff(k.h.Adm, k.h.Act, prog, nil)
	c.Logf("explanation: >>%s<<", expl)
	c.Logf("correct explanation: >>%s<<", explanation)
	c.Logf("whatever=%+v; %+v", whatever, whatever[0])
	c.Logf("who=%#v", who)
	explan := strings.Split(expl, "\n")
	n := len(explan)
	if explan[n-1] == "" {
		explan = explan[:n-1]
	}
	explan = deNode(explan, "poot")
	explanation = deNode(explanation, "poot")
	sort.Strings(explan)
	sort.Strings(explanation)
	assertSameSlice(c, explan, explanation)

	wmap := make(map[string]bool, 0)
	for _, w := range who {
		wmap[w.service] = true
	}
	var whoosh []string
	for k := range wmap {
		whoosh = append(whoosh, k)
	}
	sort.Strings(whoosh)
	goodWho := []string{"poot", "segp", "s3"}
	sort.Strings(goodWho)
	display(c, whoosh, goodWho)
	assertSameSlice(c, whoosh, goodWho)

	if err != nil {
		for _, l := range delta {
			c.Logf("\t%s", l.String())
		}
	}
	c.Assert(err, IsNil)
	// get answer
	var ans []string
	for _, w := range delta {
		ans = append(ans, w.String())
	}
	ans = deNode(ans, "poot")
	correct = deNode(correct, "poot")
	sort.Strings(ans)
	sort.Strings(correct)
	c.Logf("ans=%v   correct=%v", ans, correct)
	assertSameSlice(c, correct, ans)
}

func display(c *C, a, b []string) {
	k := len(a)
	if k > len(b) {
		k = len(b)
	}
	for i := 0; i < k; i++ {
		c.Logf("%d:%v:%30s|%30s", i, a[i] == b[i], a[i], b[i])
	}
	for i := k; i < len(a); i++ {
		c.Logf("%d:%v:%30s|%30s", i, false, a[i], "")
	}
	for i := k; i < len(b); i++ {
		c.Logf("%d:%v:%30s|%30s", i, false, "", b[i])
	}
}
