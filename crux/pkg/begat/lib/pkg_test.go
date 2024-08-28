package lib

import (
	"sort"
	"sync"
	"testing"

	"github.com/erixzone/crux/pkg/begat/common"

	. "gopkg.in/check.v1"
)

func TestBegatLib(t *testing.T) { TestingT(t) }

func assertSameSlice(c *C, should, was []string) {
	if len(should) != len(was) {
		c.Logf("should:")
		for i, x := range should {
			c.Logf("%6d %s", i, x)
		}
		c.Logf("was:")
		for i, x := range was {
			c.Logf("%6d %s", i, x)
		}
	}
	c.Assert(len(should), Equals, len(was), Commentf("expected len=%d, got len=%d", len(should), len(was)))
	for i := range should {
		c.Assert(should[i], Equals, was[i], Commentf("%d: expected '%s', got '%s'", i, should[i], was[i]))
	}
}

func postcheck(c *C, stuff [][]string, post []EventFS, dicts []string) {
	c.Logf("postcheck: post=%v dicts=%v", post, dicts)
	fs := make(map[string]string)
	var did []string
	for _, x := range stuff {
		switch x[1] {
		case "exec":
			did = append(did, x[0])
		case "fs":
			fs[x[3]] = x[4]
		}
	}
	// compare dicts
	sort.Strings(dicts)
	sort.Strings(did)
	assertSameSlice(c, dicts, did)
	// compare files
	for _, f := range post {
		c.Logf("%s: want %d, got %d", f.Path, f.Hash, common.GetHashString(fs[f.Path]))
	}
}

// testing interfaces
type testHistory struct {
	sync.Mutex
	h map[string]*Travail
}

func newHistory() *testHistory {
	h := testHistory{h: make(map[string]*Travail)}
	return &h
}

func (th *testHistory) Clear() {
	th.Lock()
	defer th.Unlock()
	th.h = make(map[string]*Travail)
}

func (th *testHistory) GetTravail(ch *Chore) (*Travail, error) {
	th.Lock()
	defer th.Unlock()
	key := getSig(ch)
	//fmt.Printf("HIST: fetch(%20q) %p\n", key, th.h[key])
	if t := th.h[key]; t != nil {
		return t, nil
	}
	return nil, nil
}

func (th *testHistory) PutTravail(t *Travail) error {
	th.Lock()
	defer th.Unlock()
	key := getSig(&(t.Chore))
	//fmt.Printf("HIST: store(%20q) %p %+v\n", key, t, *t)
	th.h[key] = t
	return nil
}

func (th *testHistory) Quit() {
}
