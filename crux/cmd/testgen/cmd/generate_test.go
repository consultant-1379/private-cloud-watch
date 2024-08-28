package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

type GenTester struct {
}

func init() {
	Suite(&GenTester{})
}

func TestGen(t *testing.T) { TestingT(t) }
func (s *GenTester) SetUpSuite(c *C) {
}

func (s *GenTester) TearDownSuite(c *C) {
	// nothing
}

var spec = `test1: {
	begatfile: test1.bg=4527a83f
	history: clear
	inputs: [ file1=23deadbeef file2=deadbeef42 ]
} => {
	dictums: [ d1 d3 d6 d9 d10 ]
	outputs: [ poot=71fea624 ]
}

test1a: test1 + { } => {
	dictums: []
}

test1b: test1 + {
	inputs: [ file1=77665544 ]
} => {
	dictums: [ d6 d10 ]
	outputs: [ poot=28ac321 ]
}

test1c: test1 + test5 + {
	pastiche: [ -abc.o=26a7bb2 ]
} => {
	dictums: []
}

test5: {
	begatfile: test1.bg=4527a83f
	history: clear
	inputs: [ file1=23deadbeef file2=deadbeef42 ]
} => {
	dictums: [ d1 d3 d6 d9 d10 ]
	outputs: [ poot=71fea624 ]
}
`

func (s *GenTester) TestParse(c *C) {
	out := `{name:test1 prior:[] pre:{begatfile:{add:true id:test1.bg chk:4527a83f} history:{add:true id:clear chk:} pastiche:[] inputs:[{add:true id:file1 chk:23deadbeef} {add:true id:file2 chk:deadbeef42}]} post:{dictums:[{add:true id:d1 chk:} {add:true id:d3 chk:} {add:true id:d6 chk:} {add:true id:d9 chk:} {add:true id:d10 chk:}] outputs:[{add:true id:poot chk:71fea624}]}}
{name:test1a prior:[test1] pre:{begatfile:{add:false id: chk:} history:{add:false id: chk:} pastiche:[] inputs:[]} post:{dictums:[] outputs:[]}}
{name:test1b prior:[test1] pre:{begatfile:{add:false id: chk:} history:{add:false id: chk:} pastiche:[] inputs:[{add:true id:file1 chk:77665544}]} post:{dictums:[{add:true id:d6 chk:} {add:true id:d10 chk:}] outputs:[{add:true id:poot chk:28ac321}]}}
{name:test1c prior:[test1 test5] pre:{begatfile:{add:false id: chk:} history:{add:false id: chk:} pastiche:[{add:false id:test5 chk:abc.o}] inputs:[]} post:{dictums:[] outputs:[]}}
{name:test5 prior:[] pre:{begatfile:{add:true id:test1.bg chk:4527a83f} history:{add:true id:clear chk:} pastiche:[] inputs:[{add:true id:file1 chk:23deadbeef} {add:true id:file2 chk:deadbeef42}]} post:{dictums:[{add:true id:d1 chk:} {add:true id:d3 chk:} {add:true id:d6 chk:} {add:true id:d9 chk:} {add:true id:d10 chk:}] outputs:[{add:true id:poot chk:71fea624}]}}
`
	rd := strings.NewReader(spec)
	tests, err := parseSpec(rd)
	c.Assert(err, IsNil, Commentf("generate error: %s", err))
	for i := range tests {
		fmt.Printf("tests[%d] = %+v\n", i, tests[i])
	}
	var xx string
	for _, t := range tests {
		xx += fmt.Sprintf("%+v\n", *t)
	}
	c.Logf("wanted:\n%s\ngot\n%s\n", out, xx)
	c.Assert(xx, Equals, out)
	c.Logf("++++++generate test done\n")
}

func (s *GenTester) TestGen1(c *C) {
	rd := strings.NewReader(spec)
	tests, err := parseSpec(rd)
	c.Assert(err, IsNil, Commentf("generate error: %s", err))
	generate(nil, nil, true, os.Stdout) // header
	generate(nil, tests[0:1], true, os.Stdout)
	generate(nil, tests[1:2], true, os.Stdout)
	generate(nil, nil, false, os.Stdout) // trailer
}
