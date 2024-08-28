package lib

import (
	"fmt"
	"strings"

	. "gopkg.in/check.v1"
)

type ParseTester struct {
}

func init() {
	Suite(&ParseTester{})
}

func (s *ParseTester) SetUpSuite(c *C) {
}

func (s *ParseTester) TearDownSuite(c *C) {
	// nothing
}

func (s *ParseTester) TestParse1(c *C) {
	test3 := []string{
		"stmt calld cc[../tests/test3.begat:15] i=abc.c o=abc.o mount=[. RW lfs:.] dir=.",
		"stmt calld cc[../tests/test3.begat:15] i=a.c o=a.o mount=[. RW lfs:.] dir=.",
		"stmt calld cc[../tests/test3.begat:15] i=b.c o=b.o mount=[. RW lfs:.] dir=.",
		"stmt calld cc[../tests/test3.begat:15] i=c.c o=c.o mount=[. RW lfs:.] dir=.",
		"stmt calld cc[../tests/test3.begat:15] i=x.c o=x.o mount=[. RW lfs:. /data RO lfs:/x/y] dir=/tmp",
		"stmt calld cc[../tests/test3.begat:15] i=y.c o=y.o mount=[. RW lfs:. /data RO lfs:/x/y] dir=/tmp",
		"stmt calld link[../tests/test3.begat:6] i=%.c o=%.o mount=[. RW lfs:.] dir=.",
	}
	p, err := ParseFile("../tests/test3.begat")
	c.Assert(err, IsNil, Commentf("compile error: %s", err))

	gen := make([]string, 0)
	for _, s := range p.code {
		gen = append(gen, nub(s))
	}
	assertSameSlice(c, test3, gen)
	c.Logf("++++++parse test done\n")
}

func nub(s *Statement) string {
	res := "stmt "
	switch s.What {
	case StatementVar:
		res += fmt.Sprintf("var %s = %s", s.Vr.Name, s.Vr.Val)
	case StatementCallDict:
		res += fmt.Sprintf("calld %s mount=%s dir=%s", s.Dict, mstring(s.Mount), s.Dir)
	case StatementCallFunc:
		res += fmt.Sprintf("callf %s(%s)", s.Name, prettyList(s.Args))
	case StatementDict:
		res += fmt.Sprintf("dictum %s mount=%s dir=%s", s.Dict, mstring(s.Mount), s.Dir)
	case StatementApply:
		res += fmt.Sprintf("apply %s to %s", s.Args[0], prettyList(s.Args[1:]))
	case StatementFunc:
		res += fmt.Sprintf("func %s{}", s.Fn.Name)
	case StatementCd:
		res += fmt.Sprintf("cd %s {%s}", s.Dir, nubb(s.Block))
	case StatementMount:
		res += fmt.Sprintf("mount(%s)", prettyList(s.Args))
	}
	return res
}

func nubb(b *Block) string {
	if b == nil {
		return "<nil block>"
	}
	s := "??block??"
	switch len(b.Stmts) {
	case 0:
		s = ""
	case 1:
		s = b.Stmts[0].String()
	default:
		s = b.Stmts[0].String() + " ... " + b.Stmts[len(b.Stmts)-1].String()
	}
	return s
}

func mstring(mnt []*Statement) string {
	ret := ""
	for _, s := range mnt {
		ret += " " + strings.Join(s.Args, " ")
	}
	if len(ret) > 0 {
		ret = ret[1:]
	}
	return "[" + ret + "]"
}
