package khan

import (
	"fmt"
	"sort"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/horde"
	kd "github.com/erixzone/crux/pkg/khan/defn"
)

// Where describes the instances of a service on a node
type Where struct {
	service string
	node    string
	count   int
}

// What describes service/stage combinations actually existing and wanted.
type What struct {
	service string
	stage   string
	acount  int // actual total count
	wcount  int // wanted total count
}

// diff returns the changes needed to satisfy the spec
func diff(adm horde.Administer, act horde.Action, spec string, pending []horde.Service) ([]Where, []*What, []Where, string, *crux.Err) {
	_, nodes, err0 := extractKV(adm)
	if err0 != nil {
		return nil, nil, nil, "", err0
	}
	is, whatever, err1 := existing(act, pending)
	if err1 != nil {
		return nil, nil, nil, "", err1
	}
	want, constraints, err2 := lineup(adm, spec, where2diaspora(is, nodes))
	if err2 != nil {
		return nil, nil, nil, "", err2
	}
	fmt.Printf("Diff0: want=%+v   is=%+v  pending=%v\n", want, is, pending)
	// constrain the is list to our scope, which is stuff in want
	is = scope(is, want)
	//fmt.Printf("Diff: want=%+v   is=%+v  pending=%v\n", want, is, pending)
	// do the diff via maps
	isMap := make(map[string]*Where, 0)
	wantMap := make(map[string]*Where, 0)
	for ix, x := range is {
		isMap[x.node+"|"+x.service] = &is[ix]
	}
	for ix, x := range want {
		wantMap[x.node+"|"+x.service] = &want[ix]
	}
	// generally safer to add new processes first, and then delete
	var out []Where
	var explanation string
	for key, w := range wantMap {
		var delta int
		var exp string
		if i, ok := isMap[key]; ok {
			delta, exp = allow(w.count-i.count, w.service, w.node, constraints, whatever, want)
		} else {
			delta, exp = allow(w.count, w.service, w.node, constraints, whatever, want)
		}
		explanation += exp
		if delta != 0 {
			out = append(out, Where{node: w.node, service: w.service, count: delta})
		}
		delete(isMap, key) // to mark that we did this one
	}
	// clean up any existing that ought not to be
	for _, i := range isMap {
		out = append(out, Where{node: i.node, service: i.service, count: -i.count})
		explanation += fmt.Sprintf("stop %d existing '%s's on %s; not needed\n", i.count, i.service, i.node)
	}
	return out, whatever, want, explanation, nil
}

// return the state of the world as it is. this also includes a notion of pending
// actions, because we don't want to overstart services.
func existing(act horde.Action, pending []horde.Service) ([]Where, []*What, *crux.Err) {
	svcs := act.What()
	// first generate What slice
	wmap := make(map[string]*What, 0)
	for _, n := range svcs {
		k := n.Name + "|" + string(n.Stage)
		w, ok := wmap[k]
		if !ok {
			w = &What{service: n.Name, stage: string(n.Stage), acount: 0}
			wmap[k] = w
		}
		w.acount++
	}
	var whatever []*What
	for _, w := range wmap {
		whatever = append(whatever, w)
	}
	// now, pending map
	pmap := make(map[string]*horde.Service, 0)
	// build adjust map
	for i := range pending {
		s := pending[i]
		k := s.Node + "," + s.Name + "+" + s.UniqueID
		pmap[k] = &pending[i]
	}
	// accumulate counts
	count := make(map[string]*Where)
	for _, n := range svcs {
		// first, deal with adjustments
		k := n.Node + "," + n.Name + "+" + n.UniqueID
		if s, ok := pmap[k]; ok {
			if s.Delete {
				continue // ignore
			}
			// otherwise, it has already added so delete it
			delete(pmap, k)
		}
		k = n.Node + "," + n.Name
		w := count[k]
		if w == nil {
			w = &Where{node: n.Node, service: n.Name, count: 0}
			count[k] = w
		}
		w.count++
	}
	// add in "adds" from pending
	for _, s := range pmap {
		if s.Delete {
			continue // ignore deletes
		}
		k := s.Node + "," + s.Name
		w := count[k]
		if w == nil {
			w = &Where{node: s.Node, service: s.Name, count: 0}
			count[k] = w
		}
		w.count++
	}
	// deliver counts
	var vec []Where
	for _, w := range count {
		vec = append(vec, *w)
	}

	fmt.Printf("existing: svcs=%v pending=%v returns vec=%v\n", svcs, pending, vec)
	return vec, whatever, nil
}

// narrow range
func scope(list []Where, universe []Where) []Where {
	// easy; build a map of allowable services
	dict := make(map[string]bool, len(universe))
	for _, w := range universe {
		dict[w.service] = true
	}
	// now just trim list to stuff in our dictionary
	var ret []Where
	for _, w := range list {
		if _, ok := dict[w.service]; ok {
			ret = append(ret, w)
		}
	}
	return ret
}

/*
	we don't care about efficiency here. so we compute helper maps each time and so forth.
	we can improve that should the need arise.
*/

type scount struct {
	atotal int
	wtotal int
	stc    map[string]int // key is stage
}

func inc(m map[string]*scount, service, stage string, acount, wcount int) {
	var ok bool
	var sc *scount
	if sc, ok = m[service]; !ok {
		sc = &scount{atotal: 0, wtotal: 0, stc: make(map[string]int, 0)}
		m[service] = sc
	}
	sc.wtotal += wcount
	sc.atotal += acount
	sc.stc[stage] += acount
}

// this implements the scheduling constraints
func allow(count int, service string, node string, constraints []*kd.After, is []*What, want []Where) (int, string) {
	if count == 0 { // nothing to do
		return 0, ""
	}
	if count < 0 { // currently no constraints on stopping stuff
		return count, fmt.Sprintf("stop %d instances of '%s' on %s; no constraints\n", -count, service, node)
	}
	// assemble counts
	counts := make(map[string]*scount, 0)
	for _, w := range is {
		inc(counts, w.service, w.stage, w.acount, 0)
	}
	for _, w := range want {
		inc(counts, w.service, "junk", 0, w.count)
	}
	// we can actually do something now!!
	doit := true
	reason := ""
	for _, c := range constraints {
		if c.Service != service {
			continue
		}
		// ignore input stage at this point
		fmt.Printf(">>>%s:  %s\n", service, c.String())
		for _, cc := range c.Pre {
			if cc.Stage == "" {
				cc.Stage = string(horde.StageReady)
			}
			cnt, ok := counts[cc.Service]
			if cc.IsCount {
				var actual int
				if ok {
					actual = cnt.stc[cc.Stage]
				}
				if float64(actual) >= cc.Val {
					reason += fmt.Sprintf(" good[%s.%s: actual(%d) >= target(%.0f)]",
						cc.Service, cc.Stage, actual, cc.Val)
				} else {
					doit = false
					reason += fmt.Sprintf(" bad[%s.%s: actual(%d) < target(%.0f)]",
						cc.Service, cc.Stage, actual, cc.Val)
				}
			} else {
				var actual, denom int
				fmt.Printf("ok=%v stage=%s act=%d\n", ok, cc.Stage, cnt.stc[cc.Stage])
				if ok {
					actual = cnt.stc[cc.Stage]
					denom = cnt.wtotal
				}
				perc := 100.0 * float64(actual) / (float64(denom) + 1e-20)
				if perc >= cc.Val {
					reason += fmt.Sprintf(" good[%s.%s: actual(%.1f%%) >= target(%.1f%%)]",
						cc.Service, cc.Stage, perc, cc.Val)
				} else {
					doit = false
					reason += fmt.Sprintf(" bad[%s.%s: actual(%.1f%%) < target(%.1f%%)]",
						cc.Service, cc.Stage, perc, cc.Val)
				}
			}
		}
	}
	if reason == "" {
		reason = " no constraints"
	}
	if doit {
		return count, fmt.Sprintf("start %d instances of '%s' on %s; %s\n", count, service, node, reason)
	}

	return 0, fmt.Sprintf("start no instances of '%s' on %s; %s\n", service, node, reason)
}

// sorting goo
type wlist []Where

func (w wlist) Len() int      { return len(w) }
func (w wlist) Swap(i, j int) { w[i], w[j] = w[j], w[i] }
func (w wlist) Less(i, j int) bool {
	a := &w[i]
	b := &w[j]
	if a.node < b.node {
		return true
	}
	if a.node > b.node {
		return false
	}
	if a.service < b.service {
		return true
	}
	if a.service > b.service {
		return false
	}
	return a.count < b.count
}

func wsort(w []Where) {
	sort.Sort(wlist(w))
}
