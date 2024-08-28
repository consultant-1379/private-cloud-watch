package lib

import (
	"sort"

	"github.com/twmb/algoimpl/go/graph"

	"github.com/erixzone/crux/pkg/crux"
)

type pair struct {
	a, b string // a "less" b
}

// Tsort stores the pairs of related items
type Tsort struct {
	list []pair
	nmap map[string]*graph.Node
	g    *graph.Graph
}

// NewTsort returns an initialised Tsort
func NewTsort() *Tsort {
	var t Tsort
	return &t
}

// Pair adds the relationship aRb
func (t *Tsort) Pair(a, b string) {
	t.list = append(t.list, pair{a: a, b: b})
}

func (t *Tsort) addPair(a, b string) {
	na, ok := t.nmap[a]
	if !ok {
		x := t.g.MakeNode()
		*x.Value = a
		t.nmap[a] = &x
		na = &x
	}
	nb, ok := t.nmap[b]
	if !ok {
		x := t.g.MakeNode()
		*x.Value = b
		t.nmap[b] = &x
		nb = &x
	}
	crux.Assert(t.g.MakeEdge(*na, *nb) == nil)
}

// Order sorts the given pairs.
func (t *Tsort) Order() [][]string {
	/*
		this is hard because
		1) we want equivalence sets, whereas most top-sorts just give you an order.
			we need to know how many we can fire at once (for more parallelism)
		2) there may be cycles, and i know of no simple algs for enumerating all the cycles.
			the graph library does that using strongly connected nodes

		so we do the obvious thing: use StronglyConnectedComponents to detect cycles,
		pretend those cycles are a mega-node, and now just do top-sort with equivalence
		classes by brute force.
	*/
	// first, make the graph
	t.g = graph.New(graph.Directed)
	t.nmap = make(map[string]*graph.Node)
	for _, p := range t.list {
		t.addPair(p.a, p.b)
	}
	// next, eliminate any cycles
	megaNode := make(map[string]string)
	megaVal := make(map[string][]string)
	sc := t.g.StronglyConnectedComponents()
	for i := range sc {
		if len(sc[i]) > 1 {
			name := crux.LargeID()
			var this []string
			for _, x := range sc[i] {
				me := (*x.Value).(string)
				this = append(this, me)
				megaNode[me] = name
			}
			megaVal[name] = this
		}
	}
	// form the graph again, converting nodes on cycles to a meganode
	t.g = graph.New(graph.Directed)
	t.nmap = make(map[string]*graph.Node)
	all := make(map[string]bool)
	for _, p := range t.list {
		pa := p.a
		xa, ok := megaNode[pa]
		if ok {
			pa = xa
		}
		pb := p.b
		xb, ok := megaNode[pb]
		if ok {
			pb = xb
		}
		t.addPair(pa, pb)
		all[pa] = true
		all[pb] = true
	}
	// now we make the equivalence classes in reverse order
	var ec [][]string
	used := make(map[string]bool, len(all))
	//	fmt.Printf("equiv!! all=%v megaNode=%v megaVal=%v\n", all, megaNode, megaVal)
	for len(used) != len(all) {
		var cur, ucur []string
		for cand := range all {
			if used[cand] {
				continue
			}
			viable := true
			_, isMegaNode := megaVal[cand]
			for _, p := range t.list {
				applies := p.a == cand
				if !applies && isMegaNode {
					for _, x := range megaVal[cand] {
						applies = applies || (x == p.a)
					}
				}
				//				fmt.Printf("\tcand=%s p=%v applies=%v used[p.b]=%v\n", cand, p, applies, used[p.b])
				// only test if applicable and it isn't targeting ourself
				pb := p.b
				if mn, ok := megaNode[p.b]; ok {
					if mn == cand {
						applies = false
					}
					pb = mn
				}
				if applies {
					viable = viable && used[pb]
				}
			}
			if viable {
				if _, ok := megaVal[cand]; ok {
					cur = append(cur, megaVal[cand]...)
				} else {
					cur = append(cur, cand)
				}
				ucur = append(ucur, cand)
			}
			//			fmt.Printf("len(u)=%d len(a)=%d: cand=%s viable=%v, ismn=%v\n", len(used), len(all), cand, viable, isMegaNode)
		}
		//		fmt.Printf("--cur = %v  (used=%v)\n", cur, used)
		ec = append(ec, cur)
		for _, x := range ucur {
			used[x] = true
		}
	}
	// reverse
	lec := len(ec) - 1
	for i := 0; i < len(ec)/2; i++ {
		ec[i], ec[lec-i] = ec[lec-i], ec[i]
	}
	//	fmt.Printf("Order returns %v (list = %v)\n", ec, t.list)
	return ec
}

// Slices2String is a standard way to represent a slice of string slices as a string
func Slices2String(s [][]string, sep string) string {
	var ret string
	for i := range s {
		cursep := ""
		sorted := make([]string, len(s[i]))
		copy(sorted, s[i][0:])
		sort.Strings(sorted)
		for _, x := range sorted {
			ret = ret + cursep + x
			cursep = sep
		}
		ret = ret + "\n"
	}
	return ret
}
