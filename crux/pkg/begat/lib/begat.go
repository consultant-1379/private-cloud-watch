package lib

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

/*
	these routines do the setup and management of the various lib routines
to actually perform the begat function. they depend on the BI* interfaces
for interacting with the world.

	the governing commentary on this code is in doc.go; you need to look at execute.go as well.
the code below tries to look as much like the pseudocode (in doc.go) as much as possible.

TBD: the whole targets thing is bullshit. we need to just implement what the epistle says.
*/

// Bbox represents a thing to be tracked.
type Bbox struct {
	label string
	tape  BlackBox
}

// BegatFile does exactly that
func BegatFile(srcPath string, targets []string, log clog.Logger, fsi BIfs, hist BIhistory, exec BIexec, status chan EventStatus, tape BlackBox) *crux.Err {
	// set up a logger
	blog := log.Log("who", "begat")
	// parse the source
	p, err := ParseFile(srcPath)
	if err != nil {
		return crux.ErrE(err)
	}
	fmt.Printf("parse returns %+v\n", p.code)
	// convert the parsed output into executable Chores
	chores, err := p.prep(targets)
	if err != nil {
		return crux.ErrE(err)
	}
	blog.Logc("chores", chores, nil, "prep returns %d chores", len(chores))
	/*
		a general caveat: this control routine does not look at (or care about) the traffic on the status channel.
		the chore feedback is done by the reply part of the rpc in stepping the chores.
	*/

	// spin up and execute the chores. the chores are a slice of groups.
	blog.Logc("initialising")
	for _, x := range chores {
		for _, y := range x {
			y.Ctl = make(chan EventControl, 99)
			y.ret = make(chan EventControl, 1)
			// compute pretendable
			pretend := true
			for _, e := range y.D.Outputs {
				if mem(filepath.Join(y.Dir, e), targets) {
					pretend = false
				}
			}
			// go execute

			go y.execute(pretend, status, y.Ctl, log, fsi, hist, exec, &Bbox{label: y.D.LogicalID, tape: tape})
		}
	}

	// main control loop
	var success bool
	setExec(chores, true)
	for progress := true; progress || !success; {
		progress = false
		blog.Logm("main control loop")
		for i := len(chores) - 1; i >= 0; i-- { // bottom to top
			g := chores[i]
			any, mustbuild := runGroup(g)
			blog.Logm(nil, "group %d return any=%v mb=%v", i, any, mustbuild)
			if mustbuild {
				setExec(chores, false)
				// down to bottom because thats the way MUSTBUILDs propagate
				for i < len(chores) {
					_, _ = runGroup(chores[i])
					blog.Logm(nil, "cranking down; just did group %d", i)
					i++
				}
				setExec(chores, true)
				blog.Logm("resuming march to victory")
				continue
			}
			if any {
				progress = true
			}
		}
		success = true
		for _, c := range chores[0] {
			success = success && ((c.Status == StatusPretend) || (c.Status == StatusExecuted))
		}
		blog.Logm(nil, "success = %v", success)
	}
	// success indicates whether we won!
	// shut it all down concurrently
	blog.Logc("shutting down chores")
	for _, x := range chores {
		for _, y := range x {
			y.Ctl <- EventControl{T: time.Now(), Op: OpQuit, Return: y.ret}
		}
	}
	// now wait for them to be shut down
	for _, x := range chores {
		for _, y := range x {
			blog.Logc(nil, "---waiting for chore %s", y.RunID)
			<-y.ret
			close(y.Ctl)
			close(y.ret)
		}
	}
	// final cleanup
	blog.Logc("done")
	return nil
}

// runGroup cranks each chore, returning progress and mustbuildness.
func runGroup(g []*Chore) (progress bool, mustb bool) {
	// set up a logger
	rlog := clog.Log.Log("who", "rungroup")
	rlog.Logc("rg: starting")
	for _, x := range g {
		x.Ctl <- EventControl{T: time.Now(), Op: OpCrank, Return: x.ret}
	}
	for _, x := range g {
		ec := <-x.ret
		rlog.Logm(nil, "-----rg: %+v", ec)
		progress = progress || ec.Progress
		mustb = mustb || ec.MustBuild
	}
	rlog.Logc(nil, "rg: returns progress=%v mustb=%v", progress, mustb)
	return
}

func setExec(chores [][]*Chore, val bool) {
	for _, x := range chores {
		for _, y := range x {
			y.Ctl <- EventControl{T: time.Now(), Op: OpExec, Progress: val, Return: y.ret}
			<-y.ret
			fmt.Printf("sent OpExec to %s\n", y.RunID)
		}
	}
	fmt.Printf("setExec done\n")
}

func mem(s string, set []string) bool {
	for _, x := range set {
		if x == s {
			return true
		}
	}
	return false
}
