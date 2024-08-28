package khan

import (
	"math"

	"github.com/erixzone/crux/pkg/crux"
	kd "github.com/erixzone/crux/pkg/khan/defn"
)

/*
	excuse this homegrown solver; we should be using something like microsoft's Z3
	or some other SMT solver, but that is too much work right now.
	so we just will use brute force.

	reduce takes a set of expressions and a variable->value map, and looks to see if there
	are any pick operators in them. if there is one or more that can be evaluated, pick one and
		1) reduce the expressions by the one that contains that operator
		2) solve the remaining expressions for each possible value for that operator.
	return all valid results via the solns channel. this allows optimisation
	of an objective function (such as we do in Lineup, which minimises the "distance" from
	the existing configuration)

	reduce takes a fair amount of setup; therefore, we test reduce via lineup_test.
*/

func reduce(exprs []*kd.Expr, syms *kd.Vset, all []int, dist *diaspora, soln chan *diaspora) (*diaspora, *crux.Err) {
	var problem *kd.Expr
	for i, e := range exprs {
		ee, op, ok := pickable(e, syms)
		if ok {
			newe := make([]*kd.Expr, 0, len(exprs))
			newe = append(newe, exprs[0:i]...)
			newe = append(newe, exprs[i+1:]...)
			return drive(e, newe, syms, all, dist, soln)
		}
		if op {
			problem = ee
		}
	}
	if problem != nil {
		return dist, crux.ErrF("can't evaluate pick in %s", problem.String())
	}
	// so no pick operaters. so just check if all remain exprs are evalable.
	for _, e := range exprs {
		if !e.Evalable(syms) {
			return nil, nil
		}
	}
	return dist, nil
}

/*
	get the pick operator, systematically give it a value (one of a zillion) and then evaluate the whole
driver expression (e). then call reduce on the remaining expressions.

we need to be careful about the symbol table. the problem is that setting a variable
may allow an arbitrary number of variables to be set (e.g. in a1:=size(x); a2:=size(x);
setting x allows a1 and a2 to resolve.) so we handle it by
	1) take a copy of the symbol table coming so we can restore it on return
	2) create a new symtable (copy of the param symtable) and pass that to the recursive reduce
		that amended symtable is thrown away
	3) restore symtable and return
*/
func drive(e *kd.Expr, exprs []*kd.Expr, syms *kd.Vset, all []int, dist *diaspora, solns chan *diaspora) (*diaspora, *crux.Err) {
	// first copy our symbol table so we can restore it on return
	syms.Guard.Lock()
	osyms := copyv(syms.V)
	syms.Guard.Unlock()
	// get the pick operator
	pick, _, _ := pickable(e, syms)
	n := pick.ExprL.Eval(syms, all)
	set := pick.ExprR.Eval(syms, all)
	var ld *diaspora
	for _, soln := range strew(count(n), set.Set) {
		// be careful: strew returns distributions
		pick.V = kd.SetValueDist(soln)
		dist.assign(pick.Str, soln)
		// this Eval may set stuff in syms, so reset
		syms.Guard.Lock()
		syms.V = copyv(osyms)
		syms.Guard.Unlock()
		e.Eval(syms, all)
		d, e := reduce(exprs, syms, all, dist, solns)
		if (d != nil) && (e == nil) {
			ld = d
			solns <- d
			//fmt.Printf("GORP: %+v\n", syms.V)
		}
	}
	// reset pick's value back to nil
	pick.V = nil
	// restore symbol table
	syms.Guard.Lock()
	syms.V = osyms
	syms.Guard.Unlock()
	// no possible value for the pick worked
	return ld, nil
}

func copyv(m kd.Vsym) kd.Vsym {
	x := make(kd.Vsym, len(m))
	for k, v := range m {
		x[k] = v
	}
	return m
}

// pickable returns any pick operator, and if its evalable
func pickable(e *kd.Expr, syms *kd.Vset) (picky *kd.Expr, found, good bool) {
	good = true
	e.Traverse(func(ex *kd.Expr) {
		switch ex.Op {
		case kd.OpPick, kd.OpPickh:
			picky = ex
			found = true
			v1 := ex.ExprL.Evalable(syms)
			v2 := ex.ExprR.Evalable(syms)
			if (!v1) || (!v2) {
				good = false
			}
		}
	})
	if !found {
		good = false
	}
	return
}

// don't let fractions spoil our day -- use ceil
func count(e *kd.Value) int {
	return int(math.Ceil(e.Num))
}
