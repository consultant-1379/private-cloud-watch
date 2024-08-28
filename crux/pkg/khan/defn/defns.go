package defn

// this subdirectory package is because the generate grammar file lives over in another package
// and we therefore have to do this to avoid an import loop.

import (
	"fmt"
	"strings"
	"sync"
)

// Operator is the operator holder
type Operator int

// random constants plus tokens
const (
	Opunused Operator = iota
	OpPick
	OpPickh
	OpAssign
	OpVAssign
	OpAll
	OpLabel
	OpNot
	OpOr
	OpAnd
	OpNum
	OpSize
	OpPlus
	OpMinus
	OpMult
	OpDivide
	OpVar
	OpSet // just a constant set
)

// Expr holds a compiled expression. when we evaluate an expression, we don't
// recurse down if the Value field is non-nil.
type Expr struct {
	Op           Operator
	ExprL, ExprR *Expr
	Num          float64
	Str          string
	Strlist      []string
	V            *Value // filled in later
}

// Value holds a value (number or set)
type Value struct {
	IsNum bool
	Num   float64
	Set   []int
}

// Vsym is the symtable map
type Vsym map[string]*Value

// Vset holds a collection of named values
type Vset struct {
	Guard sync.Mutex
	V     Vsym
}

// CopyVset makes a copy
func CopyVset(v *Vset) Vsym {
	vv := make(Vsym, len(v.V))
	for i, j := range v.V {
		vv[i] = j
	}
	return vv
}

// SetValue generates  Value (of type set)
func SetValue(set []int) *Value {
	v := Value{IsNum: false, Set: make([]int, len(set))}
	copy(v.Set, set)
	return &v
}

// SetValueDist creates a set value from a distribution
func SetValueDist(set []int) *Value {
	v := Value{IsNum: false, Set: make([]int, 0)}
	for i := range set {
		if set[i] > 0 {
			v.Set = append(v.Set, i)
		}
	}
	return &v
}

// NumValue creates a value (of a number)
func NumValue(x float64) *Value {
	v := Value{IsNum: true, Num: x}
	return &v
}

// Traverse is a normal tree traversal function; leaves first, then node.
func (e *Expr) Traverse(fn func(*Expr)) {
	switch e.Op {
	case OpPick, OpPickh, OpMinus, OpPlus, OpMult, OpDivide, OpOr, OpAnd:
		e.ExprL.Traverse(fn)
		e.ExprR.Traverse(fn)
	case OpAssign, OpNot, OpSize:
		e.ExprL.Traverse(fn)
	case OpVAssign:
		// nothing
	}
	fn(e)
}

// Evalable returns whether or not this expression could be evaluated (no unknown variables)
func (e *Expr) Evalable(syms *Vset) bool {
	if (e.Op != OpVar) && (e.V != nil) {
		return true
	}
	switch e.Op {
	case OpPick, OpPickh, OpMinus, OpPlus, OpMult, OpDivide, OpOr, OpAnd:
		return e.ExprL.Evalable(syms) && e.ExprR.Evalable(syms)
	case OpAssign, OpNot, OpSize:
		return e.ExprL.Evalable(syms)
	case OpVar:
		syms.Guard.Lock()
		defer syms.Guard.Unlock()
		_, ok := syms.V[e.Str]
		return ok
	case OpVAssign:
		return true
	}
	return false
}

// Eval actually evaluates the expression.
// this is broken up into seperate routines because stupid cyclo wrongly computes
// how hard it is to read a case statement.
// the least repugnant way is to segregate monadic and dyadic operators.
func (e *Expr) Eval(syms *Vset, all []int) *Value {
	if e.V != nil {
		return e.V
	}
	switch e.Op {
	case OpPick, OpPickh:
		return nil
	case OpAssign:
		e.ExprL.Eval(syms, all)
		e.V = e.ExprL.V
		syms.Guard.Lock()
		syms.V[e.Str] = e.V
		syms.Guard.Unlock()
	case OpAll:
		e.V = SetValue(all)
	case OpLabel, OpSet, OpNum, OpVAssign:
		// do nothing here
	case OpNot, OpSize:
		// monadic evals e.ExprL
		e.monadic(syms, all)
	case OpOr, OpAnd, OpPlus, OpMinus, OpMult, OpDivide:
		// dyadic evals e.ExprL and e.ExprR
		e.dyadic(syms, all)
	case OpVar:
		e.V = varValue(e.Str, syms)
	}
	return e.V
}

func (e *Expr) monadic(syms *Vset, all []int) {
	e.ExprL.Eval(syms, all)
	if e.ExprL.V != nil {
		switch e.Op {
		case OpNot:
			e.V = SetValue(SetInverse(e.ExprL.V.Set, all))
		case OpSize:
			e.V = NumValue(float64(len(e.ExprL.V.Set)))
		default:
			panic("internal error monadic")
		}
	}
}

func (e *Expr) dyadic(syms *Vset, all []int) {
	e.ExprL.Eval(syms, all)
	e.ExprR.Eval(syms, all)
	if (e.ExprL.V != nil) && (e.ExprR.V != nil) {
		switch e.Op {
		case OpOr:
			e.V = SetValue(setOr(e.ExprL.V.Set, e.ExprR.V.Set))
		case OpAnd:
			e.V = SetValue(setAnd(e.ExprL.V.Set, e.ExprR.V.Set))
		case OpPlus:
			e.V = NumValue(e.ExprL.V.Num + e.ExprR.V.Num)
		case OpMinus:
			e.V = NumValue(e.ExprL.V.Num - e.ExprR.V.Num)
		case OpMult:
			e.V = NumValue(e.ExprL.V.Num * e.ExprR.V.Num)
		case OpDivide:
			if e.ExprR.V.Num < 1e-8 { // avoid divide by zero
				e.V = NumValue(1e8)
			} else {
				e.V = NumValue(e.ExprL.V.Num / e.ExprR.V.Num)
			}
		default:
			panic("internal error dyadic")
		}
	}
}

// SetInverse computes the complement of a set
func SetInverse(set, all []int) []int {
	did := make([]bool, len(all), len(all))
	for _, i := range set {
		did[i] = true
	}
	var r []int
	for _, i := range all {
		if !did[i] {
			r = append(r, i)
		}
	}
	return r
}

func setOr(v1, v2 []int) []int {
	m := SetMax(v1, v2) + 1
	ans := make([]int, m)
	for _, i := range v1 {
		ans[i] = 1
	}
	for _, i := range v2 {
		ans[i] = 1
	}
	var ret []int
	for i := range ans {
		if ans[i] > 0 {
			ret = append(ret, i)
		}
	}
	return ret
}

func setAnd(v1, v2 []int) []int {
	m := SetMax(v1, v2) + 1
	ans := make([]int, m)
	for _, i := range v1 {
		ans[i] |= 1
	}
	for _, i := range v2 {
		ans[i] |= 2
	}
	var ret []int
	for i := range ans {
		if ans[i] == 3 {
			ret = append(ret, i)
		}
	}
	return ret
}

// SetMax returns the largest element in two sets
func SetMax(v1, v2 []int) int {
	maxx := 0
	for _, i := range v1 {
		if i > maxx {
			maxx = i
		}
	}
	for _, i := range v2 {
		if i > maxx {
			maxx = i
		}
	}
	return maxx
}

func varValue(name string, syms *Vset) *Value {
	syms.Guard.Lock()
	defer syms.Guard.Unlock()
	if v, ok := syms.V[name]; ok {
		return v
	}
	return nil
}

func (e *Expr) String() string {
	var s string

	switch e.Op {
	case OpPick:
		s = fmt.Sprintf("pick(%s, [%s], [%s])", e.Str, e.ExprL.String(), e.ExprR.String())
	case OpPickh:
		s = fmt.Sprintf("pickh(%s, [%s], [%s])", e.Str, e.ExprL.String(), e.ExprR.String())
	case OpAssign:
		s = fmt.Sprintf("%s := [%s]", e.Str, e.ExprL.String())
	case OpVAssign:
		s = fmt.Sprintf("%s := (%s)", e.Str, strings.Join(e.Strlist, ", "))
	case OpAll:
		s = "ALL"
	case OpLabel:
		s = fmt.Sprintf("label(%s)", e.Str)
	case OpNot:
		s = fmt.Sprintf("![%s]", e.ExprL.String())
	case OpOr:
		s = fmt.Sprintf("[%s]|[%s]", e.ExprL.String(), e.ExprR.String())
	case OpAnd:
		s = fmt.Sprintf("[%s]&[%s]", e.ExprL.String(), e.ExprR.String())
	case OpNum:
		s = fmt.Sprintf("%g", e.Num)
	case OpSize:
		s = fmt.Sprintf("size(%s)", e.ExprL.String())
	case OpPlus:
		s = fmt.Sprintf("[%s]+[%s]", e.ExprL.String(), e.ExprR.String())
	case OpMinus:
		s = fmt.Sprintf("[%s]-[%s]", e.ExprL.String(), e.ExprR.String())
	case OpMult:
		s = fmt.Sprintf("[%s]*[%s]", e.ExprL.String(), e.ExprR.String())
	case OpDivide:
		s = fmt.Sprintf("[%s]/[%s]", e.ExprL.String(), e.ExprR.String())
	case OpVar:
		s = fmt.Sprintf("var(%s)", e.Str)
	case OpSet:
		s = fmt.Sprintf("setconstant")
	default:
		return fmt.Sprintf("unknown exxpr op %d", e.Op)
	}
	if e.V != nil {
		s += "=" + e.V.String()
	}
	return s
}

func (v Value) String() string {
	if v.IsNum {
		return fmt.Sprintf("%g", v.Num)
	}
	if len(v.Set) > 0 {
		s := ""
		for _, i := range v.Set {
			s += fmt.Sprintf(",%d", i)
		}

		return "[" + s[1:] + "]"
	}
	return "[]"
}

// After is a sequencing constraint structure.
type After struct {
	Service string
	Stage   string
	Pre     []*Condition
}

// Condition describes when a khan action might be delayed.
type Condition struct {
	IsCount bool    // true means an absolute count; false means a percentage (or target)
	Val     float64 // value of the Vale expression
	Vale    *Expr
	Service string
	Stage   string
}

func (a *After) String() string {
	res := a.Service
	if a.Stage != "" {
		res += "." + a.Stage
	}
	res += " after"
	for _, c := range a.Pre {
		res += " (" + c.String() + ")"
	}
	return res
}

func (c *Condition) String() string {
	res := fmt.Sprintf("%g", c.Val)
	if !c.IsCount {
		res += "%"
	}
	res += " of " + c.Service
	if c.Stage != "" {
		res += "." + c.Stage
	}
	return res
}
