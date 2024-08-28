package khan

import (
	"sync"

	kd "github.com/erixzone/crux/pkg/khan/defn"
)

/*
	 diaspora is a map of services to slices of counts per node, that is, a distribution.
the slices are indexed by node-numbers (to keep size down; we copy these a lot).

we use diasporas a lot. when we compute possible solutions (via reduce and driver), we
range each pick operator over a set of "solutions".

NOTE: To keep the count slices sane, the mapping from node name to node number must be stable.
which means being careful when constructing "nodes []string" slices
to pass to function(s) creating diaspora's.
*/
type diaspora struct {
	guard sync.Mutex
	m     map[string][]int
}

func (d *diaspora) assign(name string, dist []int) {
	x := make([]int, len(dist))
	copy(x, dist)
	d.guard.Lock()
	d.m[name] = x
	d.guard.Unlock()
}

func newDiaspora() *diaspora {
	d := diaspora{m: make(map[string][]int, 0)}
	return &d
}

func where2diaspora(w []Where, nodes []string) *diaspora {
	nm := make(map[string]int, 0)
	for i, s := range nodes {
		nm[s] = i
	}
	d := newDiaspora()
	for _, wh := range w {
		if _, ok := d.m[wh.service]; !ok {
			d.m[wh.service] = make([]int, len(nodes))
		}
		d.m[wh.service][nm[wh.node]] = wh.count
	}
	return d
}

func evalConst(expr *kd.Expr, all []int) {
	expr.Traverse(func(ex *kd.Expr) {
		switch ex.Op {
		case kd.OpNot:
			if (ex.ExprL.V != nil) && (!ex.ExprL.V.IsNum) {
				ex.V = kd.SetValue(kd.SetInverse(ex.ExprL.V.Set, all))
			}
		case kd.OpNum:
			ex.V = kd.NumValue(ex.Num)
		}
	})
}

func evalConsts(exprs []*kd.Expr, all []int) {
	for _, e := range exprs {
		evalConst(e, all)
	}
}
