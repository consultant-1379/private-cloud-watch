package khan

import (
	"sort"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/horde"
)

type KhanLineup struct {
	h *horde.Horde
}

func init() {
	Suite(&KhanLineup{})
}

func (k *KhanLineup) SetUpSuite(c *C) {
	var err error
	k.h, err = horde.GetHorde("", "crux,local", "lineup unit test")
	c.Assert(err, IsNil)
	genCluster(c, k.h.Adm)
}

func (k *KhanLineup) TearDownSuite(c *C) {
	// nothing
}

func (k *KhanLineup) TestLineup(c *C) {
	d := newDiaspora()
	d.assign("segp", []int{1, 1, 1, 1, 1, 0, 0})
	d.assign("s3", []int{0, 0, 0, 1, 1, 0, 1})
	d.assign("poot", []int{0, 0, 0, 0, 0, 2, 1})
	lup, a, err := lineup(k.h.Adm, `sp := pick(segp, 5, ALL)
	s3 := pick(s3, 3, !LABEL(KV))
	pick(poot, 2, s3&sp)
	start poot after 30% of segp.ready 2 of s3
	`, d)
	correctAfter := []string{
		"poot after (30% of segp.ready) (2 of s3)",
	}
	var aft []string
	for _, x := range a {
		aft = append(aft, x.String())
	}
	assertSameSlice(c, correctAfter, aft)
	wsort(lup)
	correct := []string{
		"segp[1@kv0]",
		"segp[1@kv1]",
		"segp[1@kv2]",
		"s3[1@node0]",
		"segp[1@node0]",
		"s3[1@node1]",
		"segp[1@node1]",
		"s3[1@node2]",
	}

	if err != nil {
		for _, l := range lup {
			c.Logf("\t%s", l.String())
		}
	}
	c.Assert(err, IsNil)
	var ans []string
	for _, l := range lup {
		ans = append(ans, l.String())
	}
	// tricky; some answers may vary, so just snarf them from the ans and add to "correct"
	correct = append(correct, extract(lup, "poot", "node")...)
	sort.Strings(correct)
	sort.Strings(ans)
	for i := range ans {
		c.Logf("==%d: %s <> %s", i, ans[i], correct[i])
	}
	assertSameSlice(c, correct, ans)
}
