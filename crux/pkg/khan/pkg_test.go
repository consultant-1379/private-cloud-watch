package khan

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/horde"
	"github.com/erixzone/crux/pkg/kv"
)

func TestKhan(t *testing.T) { TestingT(t) }

func assertSameSlice(c *C, should, was []string) {
	c.Assert(len(should), Equals, len(was), Commentf("expected len=%d, got len=%d", len(should), len(was)))
	for i := range should {
		c.Assert(should[i], Equals, was[i], Commentf("%d: expected '%s', got '%s'", i, should[i], was[i]))
	}
}

func sameSSmap(should, was map[string]string) string {
	if len(should) != len(was) {
		return fmt.Sprintf("expected len=%d, got len=%d", len(should), len(was))
	}
	did := make(map[string]bool, 0)
	for k, v := range should {
		if v != was[k] {
			return fmt.Sprintf("%s: expected '%s', got '%s'", k, v, was[k])
		}
		did[k] = true
	}
	for k, v := range was {
		if !did[k] {
			return fmt.Sprintf("%s: unexpectedly got '%s'", k, v)
		}
	}
	return ""
}

func dumpKV(c *C, kv kv.KV) {
	c.Logf("KV dump:")
	keys := kv.GetKeys("")
	sort.Strings(keys)
	for _, key := range keys {
		val, err := kv.Get(key)
		if err != nil {
			c.Logf("<<get error: %s>>", err.String())
			return
		}
		c.Logf("%s: %s", key, val)
	}
	c.Logf("-------------dump done")
}

func makenode(c *C, a horde.Administer, name string, tags []string) {
	e := a.RegisterNode(name, tags)
	c.Assert(e, IsNil)
}

func makesvc(c *C, a horde.Action, service, node, addr string) {
	a.Start1(node, service, addr)
	return
}

func genCluster(c *C, a horde.Administer) {
	makenode(c, a, "kv0", []string{"KV", "Leader"})
	makenode(c, a, "kv1", []string{"KV"})
	makenode(c, a, "kv2", []string{"KV"})
	makenode(c, a, "node0", nil)
	makenode(c, a, "node1", nil)
	makenode(c, a, "node2", nil)
	makenode(c, a, "node3", nil)
	makenode(c, a, "node9", nil)
}

func extract(ww []Where, service, node string) []string {
	var result []string
	for _, w := range ww {
		if service != w.service {
			continue
		}
		if strings.HasPrefix(w.node, node) {
			result = append(result, w.String())
		}
	}
	return result
}

func deNode(list []string, svc string) []string {
	digits := "1234567890"
	var ret []string
	for _, s := range list {
		if strings.Index(s, svc) >= 0 {
			for i := 0; i < 10; i++ {
				s = strings.Replace(s, "node"+digits[i:i+1], "node", -1)
			}
			ret = append(ret, s)
		} else {
			ret = append(ret, s)
		}
	}
	return ret
}
