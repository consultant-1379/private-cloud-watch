package cmd

import (
	"fmt"
	"github.com/erixzone/crux/pkg/crux"
	"io"
)

type begatTest struct {
	name  string
	prior []string
	pre   begatPre
	post  begatPost
}

type begatPre struct {
	begatfile begatThing
	history   begatThing
	pastiche  []begatThing
	inputs    []begatThing
}

type begatPost struct {
	dictums []begatThing
	outputs []begatThing
}

type begatThing struct {
	add bool
	id  string
	chk string
}

func parseSpec(rd io.Reader) ([]*begatTest, error) {
	ret := begat("huh?", rd, false)
	if ret != nil {
		return nil, crux.ErrE(ret)
	}
	btests := begatTests
	return btests, nil
}

func generate(spec *begatTest, allspec []*begatTest, unit bool, wr io.Writer) error {
	if spec == nil { // dodgy way of dealing with header/trailer
		if unit {
			pr1(wr)
		} else {
			// no trailer stuff yet
		}
		return nil
	}
	crux.Assert(unit) // lazy protection until we figure out integration test
	fmt.Printf("// generate(%+v)\n", *spec)
	bfile := spec.pre.begatfile.id
	if bfile == "" {
		// must be in one of the priors, i guess
		for _, p := range spec.prior {
			if tst := lookup(p, allspec); (tst != nil) && (tst.pre.begatfile.id != "") {
				bfile = tst.pre.begatfile.id
			}
		}
	}
	if bfile == "" {
		return fmt.Errorf("no begatfile specified")
	}
	bfile = `"` + bfile + `"`
	fmt.Fprintf(wr, "\nfunc (bat *BegatAutoTest) TestBegatXX%sx(c *C) {\n", spec.name)
	fmt.Fprintf(wr, "\tmyhist := newHistory()\n")
	fmt.Fprintf(wr, "\tmyfs := LocalFS()\n")
	fmt.Fprintf(wr, "\tmyexec := LocalExec(myfs)\n")
	fmt.Fprintf(wr, "\tmyexec.InitN(2)\n")
	for _, p := range spec.prior {
		fmt.Fprintf(wr, "\tjustX%s(c, %s, []string{\"rx/all.wc\"}, Log, myfs, myhist, myexec)\n", p, bfile)
	}
	fmt.Fprintf(wr, "\tres := justX%s(c, %s, []string{\"rx/all.wc\"}, Log, myfs, myhist, myexec)\n", spec.name, bfile)

	fmt.Fprintf(wr, "\t// do post\n")
	fmt.Fprintf(wr, "\tpost := []EventFS{\n")
	for _, t := range spec.post.outputs {
		fmt.Fprintf(wr, "\t\t{Op: FSEnormal, Path: \"%s\", Hash: common.GetHashString(\"%s\")},\n", t.id, t.chk)
	}
	fmt.Fprintf(wr, "\t}\n")
	fmt.Fprintf(wr, "\tdicts := []string{\n")
	for _, t := range spec.post.dictums {
		fmt.Fprintf(wr, "\t\t\"%s\",\n", t.id)
	}
	fmt.Fprintf(wr, "\t}\n")
	fmt.Fprintf(wr, "\tpostcheck(c, res, post, dicts)\n")

	fmt.Fprintf(wr, "\tc.Logf(\"++++++begat %s done\\n\")\n", spec.name)
	fmt.Fprintf(wr, "}\n")

	fmt.Fprintf(wr, "\nfunc justX%s(c *C, bfile string, targets []string, log crux.Logger, fsi BIfs, hist BIhistory, exec BIexec) [][]string {\n", spec.name)
	fmt.Fprintf(wr, "\tstatus := make(chan EventStatus, 99)\n")
	fmt.Fprintf(wr, "\tgo drainStatus(c, status)\n")
	fmt.Fprintf(wr, "\tbbox := NewBlackBox()\n")
	fmt.Fprintf(wr, "\t// do prep\n")
	if spec.pre.history.id == "clear" {
		fmt.Fprintf(wr, "\thist.Clear()\n")
	}
	if len(spec.pre.inputs) > 0 {
		fmt.Fprintf(wr, "\tpreload := []EventFS{\n")
		for _, t := range spec.pre.inputs {
			fmt.Fprintf(wr, "\t\t{%v, \"%s\", \"%s\"},\n", t.add, t.id, t.chk)
		}
		fmt.Fprintf(wr, "\t}\n")
		fmt.Fprintf(wr, "\texec.Preload(preload)\n")
	}

	fmt.Fprintf(wr, "\t// just do it\n")
	fmt.Fprintf(wr, "\terr := BegatFile(bfile, targets, log, fsi, hist, exec, status, bbox)\n")
	fmt.Fprintf(wr, "\tc.Assert(err, IsNil, Commentf(\"err: %%s\", err.Error()))\n")
	fmt.Fprintf(wr, "\tclose(status)\n")

	fmt.Fprintf(wr, "\treturn bbox.MemToFielded(bbox.Playback())\n")
	fmt.Fprintf(wr, "}\n")
	return nil
}

func lookup(id string, all []*begatTest) *begatTest {
	for _, x := range all {
		if id == x.name {
			return x
		}
	}
	return nil
}

func pr1(wr io.Writer) {
	stuff := `package lib
	
// DO NOT EDIT! this is auto-generate by testgen

import (
	"fmt"
	"os"

	"github.com/erixzone/crux/pkg/begat/common"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/walrus"

	. "gopkg.in/check.v1"
)

type BegatAutoTest struct {
}

//var Log *walrus.Logger

func init() {
	bt := BegatAutoTest{}
	Suite(&bt)
	Log = walrus.New()
	logf, err := os.Create("junk.log")
	if err != nil {
		fmt.Printf("create failed: %s\n", err.Error())
		crux.Assert(false)
	}
	_ = logf
	Log.Out = os.Stdout
	Log.SetLevel(walrus.DebugLevel)
}

func (bat *BegatAutoTest) SetUpSuite(c *C) {
	// nothing
}

func (bat *BegatAutoTest) TearDownSuite(c *C) {
	// nothing
}

func drainStatus(c *C, ch chan EventStatus) {
	var zero EventStatus
	for {
		e := <-ch
		if e == zero {
			break
		}
	}
}
`
	fmt.Fprintf(wr, "%s\n", stuff)
}
