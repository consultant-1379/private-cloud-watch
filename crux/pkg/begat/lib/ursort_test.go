package lib

import (
	. "gopkg.in/check.v1"
)

type TsortTester struct {
}

func init() {
	Suite(&TsortTester{})
}

func (s *TsortTester) SetUpSuite(c *C) {
}

func (s *TsortTester) TearDownSuite(c *C) {
	// nothing
}

func (s *TsortTester) TestTsortAcyclic(c *C) {
	correct := `a
b
c d
e
`

	t := NewTsort()
	t.Pair("a", "e")
	t.Pair("d", "e")
	t.Pair("c", "e")
	t.Pair("a", "c")
	t.Pair("a", "d")
	t.Pair("a", "b")
	t.Pair("b", "d")
	ord := t.Order()
	str := Slices2String(ord, " ")
	c.Assert(str == correct, Equals, true)
}

func (s *TsortTester) TestTsortCyclic1(c *C) {
	correct := `e
a b c
f
`
	t := NewTsort()
	t.Pair("e", "a")
	t.Pair("a", "b")
	t.Pair("b", "c")
	t.Pair("c", "a")
	t.Pair("b", "f")
	ord := t.Order()
	str := Slices2String(ord, " ")
	c.Assert(str == correct, Equals, true)
}

func (s *TsortTester) TestTsortCyclic2(c *C) {
	correct := `x
a b c
d e f
y
`
	t := NewTsort()
	t.Pair("x", "a")
	t.Pair("a", "b")
	t.Pair("b", "c")
	t.Pair("c", "a")
	t.Pair("b", "f")
	t.Pair("d", "e")
	t.Pair("e", "f")
	t.Pair("f", "d")
	t.Pair("e", "y")
	ord := t.Order()
	str := Slices2String(ord, " ")
	c.Assert(str == correct, Equals, true)
}
