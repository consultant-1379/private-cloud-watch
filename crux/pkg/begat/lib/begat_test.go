package lib

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

type BegatTester struct {
	recording bool
}

func init() {
	bt := BegatTester{recording: false}
	Suite(&bt)
	flag.BoolVar(&bt.recording, "record", false, "cause event traces to be recorded")
	logf, err := os.Create("junk.log")
	if err != nil {
		fmt.Printf("create failed: %s\n", err.Error())
		crux.Assert(false)
	}
	clog.Log = crux.GetLoggerW(logf)
	//Log.SetLevel(walrus.DebugLevel)
}

func (s *BegatTester) SetUpSuite(c *C) {
	if !flag.Parsed() {
		flag.Parse()
	}

}

func (s *BegatTester) TearDownSuite(c *C) {
	// nothing
}

func (s *BegatTester) TestBegat1(c *C) {
	myhist := newHistory()
	myfs := LocalFS()
	myexec := LocalExec(myfs)
	myexec.InitN(2)

	s.runFile(c, "../tests/test4", []string{"rx/all.wc"}, myfs, myhist, myexec)

	c.Logf("++++++begat test1 done\n")
}

func printStatus(c *C, ch chan EventStatus) {
	var zero EventStatus
	for {
		e := <-ch
		if e == zero {
			break
		}
		c.Logf("got status: %+v", e)
	}
}

func (s *BegatTester) runFile(c *C, srcPath string, targets []string, fsi BIfs, hist BIhistory, exec BIexec) {
	status := make(chan EventStatus, 99)
	go printStatus(c, status)
	bbox := NewBlackBox()
	err := BegatFile(srcPath+".begat", targets, clog.Log, fsi, hist, exec, status, bbox)
	if err != nil {
		fmt.Printf("err=%s\n", err.Error())
	}
	close(status)
	c.Assert(err, IsNil, Commentf("begat error: %s", err))
	tracefile := srcPath + ".trace"
	if s.recording {
		e := bbox.ToFile(tracefile, bbox.Playback())
		c.Assert(e, IsNil, Commentf("ToFile error: %s", e))
	} else {
		c.Logf("bbox playback:")
		for i, s := range bbox.Playback() {
			c.Logf("%3d %s", i, s)
		}
		c.Logf("------------")
		ref, err := bbox.FileToFielded(tracefile)
		c.Assert(err, IsNil, Commentf("FileToFielded error: %s", err))
		compare(c, ref, bbox.MemToFielded(bbox.Playback()))
	}
}

/*
		conceptually, comparing two traces is simple.
	we first break it down to comparing traces for an indiviual dictum.
	we then extract what matters of a dictum's trace into a "view" struct.
	we then compare the two views.

	control: we just want the sequence correct. we dont care about interplay with other types.
	status: collapse sequences into singletons. we dont care about interplay with other types.
	exec: just get the last one (beware of cycles -- i don't think we care about interior executions).
		the exec text includes the hashes for the input files, so we don't care about
		how it aligned with fs messages. (if inputs weren't included, we'd have to analyse
		the fs stream to see what the inputs were.)
	fs: given exec's are already handled, we just want to know the end state.
*/

type view struct {
	exec   string   // inputs and return status for the last executed recipe
	status []string // all statuses in order
	ctl    []string // all controls in order
	fs     []string // all files in sorted order
}

// compare two fielded sets
func compare(c *C, want, got [][]string) {
	c.Logf("compare: %d <> %d", len(want), len(got))
	saw := make(map[string]bool, 1)
	for _, x := range want {
		if !saw[x[0]] {
			saw[x[0]] = true
			compare1(c, x[0], want, got)
		}
	}
	for _, x := range got {
		if !saw[x[0]] {
			c.Errorf("got unexpected dictum %s", x[0])
			saw[x[0]] = true
		}
	}
}

func compare1(c *C, dict string, want, got [][]string) {
	// check last exec
	comparev(c, dict, build(c, dict, want), build(c, dict, got))
}

func build(c *C, dict string, stuff [][]string) view {
	var v view
	var estatus string
	justSawExec := false
	fs := make(map[string]string, 0)
	for _, x := range stuff {
		if x[0] != dict {
			continue
		}
		switch x[1] {
		case "exec":
			estatus = strings.Join(x[2:], bboxSep)
			justSawExec = true
		case "execret":
			eret := "NO EXEC RETURN"
			if justSawExec {
				eret = x[2]
				justSawExec = false
			}
			v.exec = estatus + bboxSep + eret
		case "status":
			v.status = append(v.status, strings.Join(x[2:], bboxSep))
		case "control":
			v.ctl = append(v.ctl, strings.Join(x[2:], bboxSep))
		case "fs":
			fs[x[3]] = x[4]
		default:
			c.Logf("dict %s: unknown trace field '%s'", dict, x[1])
		}
	}
	// construct fs field
	var f []string
	for k, v := range fs {
		f = append(f, k+bboxSep+v)
	}
	sort.Strings(f)
	v.fs = f
	return v
}

func comparev(c *C, lab string, want, got view) {
	c.Logf("comparev(%s):", lab)
	c.Logf("\texec:")
	c.Logf("\t\twant: %s", want.exec)
	c.Logf("\t\tgot: %s", got.exec)
	ok := want.exec == got.exec
	c.Logf("\tstatus:")
	c.Logf("\t\twant: %s", want.status)
	c.Logf("\t\tgot: %s", got.status)
	ok = ok && sliceEQ(want.status, got.status)
	c.Logf("\tcontrol:")
	c.Logf("\t\twant: %s", want.ctl)
	c.Logf("\t\tgot: %s", got.ctl)
	ok = ok && sliceEQ(want.ctl, got.ctl)
	c.Logf("\tfiles:")
	c.Logf("\t\twant: %s", want.fs)
	c.Logf("\t\tgot: %s", got.fs)
	ok = ok && sliceEQ(want.fs, got.fs)
	c.Assert(ok, Equals, true)
}

func sliceEQ(a, b []string) bool {
	return strings.Join(a, bboxSep) == strings.Join(b, bboxSep)
}
