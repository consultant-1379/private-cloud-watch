package lib

/*
testing strategy:

	set up a few clients which accept EventFS's, and print a string version on an output channel
	send a bunch of EventFs's
	sort all the outputs together and check against the correct set
*/

import (
	"fmt"
	"sort"

	. "gopkg.in/check.v1"
)

const hugeChan = 10000
const eof = "EOF"

type FSRouteTester struct {
}

func init() {
	Suite(&FSRouteTester{})
}

func (s *FSRouteTester) SetUpSuite(c *C) {
}

func (s *FSRouteTester) TearDownSuite(c *C) {
}

func (s *FSRouteTester) TestChore1(c *C) {
	correct := []string{
		"c2: /path2\n",
		"c1: /path1\n",
		"c1: /xaz\n",
	}
	// generate channels
	outputs := make(chan string, hugeChan)
	c1 := make(chan EventFS, hugeChan)
	c2 := make(chan EventFS, hugeChan)
	quitc := make(chan bool, hugeChan)
	rcmd := make(chan FSRouterCmd)
	fs := make(chan EventFS)
	finished := make(chan bool)
	// start clients and router
	var nclients int
	go FSRouter(rcmd, fs, finished)
	go client("c1", c1, outputs, quitc)
	nclients++
	go client("c2", c2, outputs, quitc)
	nclients++
	// configure router
	rcmd <- FSRouterCmd{Op: FSRopen, Dest: c1, Files: []string{"/path1"}}
	rcmd <- FSRouterCmd{Op: FSRprefix, Dest: c1, Files: []string{"/x"}}
	rcmd <- FSRouterCmd{Op: FSRopen, Dest: c2, Files: []string{"/path2"}}
	// drive router
	feeder(fs)
	// close stuff down
	rcmd <- FSRouterCmd{Op: FSRexit}
	<-finished // wait for it to shut down
	for i := 0; i < nclients; i++ {
		quitc <- true
	}
	// collect outputs
	got := gather(outputs, nclients)
	// did we get the right answer
	c.Logf("got %v\n", got)
	sort.Strings(correct)
	sort.Strings(got)
	assertSameSlice(c, correct, got)
	c.Logf("++++++fsroute test done\n")
}

func gather(ch chan string, eofs int) []string {
	var got []string
	for eofs > 0 {
		s := <-ch
		if s == eof {
			eofs--
		} else {
			got = append(got, s)
		}
	}
	return got
}

func feeder(fs chan EventFS) {
	fs <- EventFS{Path: "/y"}
	fs <- EventFS{Path: "/path1"}
	fs <- EventFS{Path: "/path2"}
	fs <- EventFS{Path: "/xaz"}
}

func client(id string, fs chan EventFS, out chan string, quit chan bool) {
	// process mix of cmds and events
normalLoop:
	for {
		select {
		case f := <-fs:
			out <- fmt.Sprintf("%s: %s\n", id, f.Path)
		case <-quit:
			break normalLoop
		}
	}
	// drain events
drainLoop:
	for {
		select {
		case f := <-fs:
			out <- fmt.Sprintf("%s: %s\n", id, f.Path)
		default:
			break drainLoop
		}
	}
	out <- eof
}
