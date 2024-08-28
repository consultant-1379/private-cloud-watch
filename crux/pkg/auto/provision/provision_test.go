// (c) Ericsson Inc. 2015-2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// NESDA implementation Testing of /tomaton/auto with provision worker
// WIP - many more conditions to be added to this test

package pro

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/auto"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

func Test(t *testing.T) { TestingT(t) }

type asuite struct {
	dir string
}

var _ = Suite(&asuite{})

func (s *asuite) SetUpTest(c *C) {
	s.dir = c.MkDir()
	// s.dir = "./"
	clog.Log = crux.GetLoggerW(os.Stdout)
}

// Tests Output Files (Saved Worker States)
// To see these in testing change SetUpTest above to s.dir = "./"
//
// p_0.json : Make a new worker, save it. Worker Not run.
//
// p_1.json : Add inputs to the worker, save it, Worker Not run.
//
// p_2.json : Launch worker in hub, run workflow, save it.
//
// p_3.json : Launch worker in hub, run workflow. Send Rmv event,
//            runs Inverse workflow, runs forward workflow. Save it.
//            **Mimics a node going down mid-workflow
//
// p_4.json : Launch worker in hub, run workflow, call FailWorker().
//            workflow undone, worker ends, save it with no
//            no results, just error messages and elasped time.
//            ** Mimics a 400 series error (our fault),
//
// p_5.json : Test reloading saved workfow from p_1.json and running it.
//
// p_6.json : Test workflow with provision task exceeding timeout,
//            should see 4 errs, then hitting fallback once.
//            ** Mimics a network down problem or server down 500 error
//
// p_7.json : Test workflow with provision task exceeding timeout and
//            fallback also exceeding timeout, triggering Inv workflow
//            undoing everything, with Lasterror . and 5 errors in
//            both provision task forward and fallback
//            ** Mimics a bigger network/server down 500 error
//
// p_8.json : Test as above but UndoOnFail set to false, which leaves
//            accumulated results intact. with lasterr and 5 errors
//            in both provision task forward and fallback

func (s *asuite) TestAuto(c *C) {

	// NewProvisionWorker
	fmt.Printf("\n\n\nMake a New Provision Worker \n")

	// Make a new Worker with the tomaton/autotest/provision
	// test set.
	Provisioner := NewProvisionWorker("provisiontest", "")
	c.Assert(Provisioner.Tasks, HasLen, 6)

	// Emit JSON
	pathspec := []string{s.dir, "p_0.json"}
	path0 := strings.Join(pathspec, "")
	writerr := Provisioner.WorkerToFile(path0)
	c.Assert(writerr, IsNil)

	// Initialize customer, size argument maps
	customerargs, cerr := auto.MapArgData([]string{"name", "num", "datacenter"},
		auto.DataT{Dstring: "Bob"},
		auto.DataT{Dint: 66},
		auto.DataT{Dstring: "JP-12345678910-01"})
	c.Assert(cerr, IsNil)

	sizeargs, serr := auto.MapArgData([]string{"size"},
		auto.DataT{Dstring: "Small"})
	c.Assert(serr, IsNil)

	// Package up the two maps in a TasktoDataMapT
	Inputs := make(auto.TasktoDataMapT)
	Inputs["customer"] = customerargs
	Inputs["size"] = sizeargs

	// Set Provisioner's Inputs
	Provisioner.SetWorkerInputs(Inputs)

	// Emit JSON
	pathspec = []string{s.dir, "p_1.json"}
	path1 := strings.Join(pathspec, "")
	writerr = Provisioner.WorkerToFile(path1)
	c.Assert(writerr, IsNil)

	fmt.Printf("\n\n\nReLoad New Provision Worker \n")

	p0, err0 := auto.WorkerFromFile(path0)
	c.Assert(err0, IsNil)
	p1, err1 := auto.WorkerFromFile(path1)
	c.Assert(err1, IsNil)
	c.Assert(p0.Tasks, HasLen, 6)
	c.Assert(p1.Tasks, HasLen, 6)
	c.Assert(p0.Workerid, Equals, "")
	c.Assert(p1.Workerid, Not(Equals), "")
	c.Assert(p0.Name, Equals, "provisiontest")
	c.Assert(p1.Name, Equals, "provisiontest")
	c.Assert(p0.Tasks["customer"].Inputs, IsNil)
	c.Assert(p0.Tasks["size"].Inputs, IsNil)
	c.Assert(p1.Tasks["customer"].Inputs, Not(IsNil))
	c.Assert(p1.Tasks["size"].Inputs, Not(IsNil))
	c.Assert(p1.Tasks["customer"].Inputs, HasLen, 3)
	c.Assert(p1.Tasks["size"].Inputs, HasLen, 1)
	c.Assert(p1.Tasks["customer"].Inputs["datacenter"].Dstring, Equals, "JP-12345678910-01")
	c.Assert(p1.Tasks["customer"].Inputs["name"].Dstring, Equals, "Bob")
	c.Assert(p1.Tasks["customer"].Inputs["num"].Dint, Equals, 66)
	c.Assert(p1.Tasks["size"].Inputs["size"].Dstring, Equals, "Small")

	fmt.Printf("\n\n\nLaunch Reloaded Provision Worker \n")

	Provisioner.SetHold() // Keep in memory for inverse workflow events
	Hub1 := auto.StartNewHub(0)
	Hub1.Launch(&Provisioner, false)

	// So with worker holding, this is no longer a valid way to test
	// to see if it as the goal.
	for {
		time.Sleep(time.Second / TIMESLICE)
		if !Provisioner.FwdGoal() {
			time.Sleep(time.Second)
			break
		}
	}

	fmt.Printf("\n\n\nCompare New (p_1) with Completed Provision Worker (p_2) \n")

	// Emit JSON
	pathspec = []string{s.dir, "p_2.json"}
	path2 := strings.Join(pathspec, "")
	writerr = Provisioner.WorkerToFile(path2)
	c.Assert(writerr, IsNil)

	// WorkflowComplete
	p1, err1 = auto.WorkerFromFile(path1)
	c.Assert(err1, IsNil)
	p2, err2 := auto.WorkerFromFile(path2)
	c.Assert(err2, IsNil)
	c.Assert(p1.Tasks, HasLen, 6)
	c.Assert(p2.Tasks, HasLen, 6)
	c.Assert(p1.Workerid, Equals, p1.Workerid)

	// Are p2 inputs intact?
	c.Assert(p2.Name, Equals, "provisiontest")
	c.Assert(p2.Tasks["customer"].Inputs, Not(IsNil))
	c.Assert(p2.Tasks["size"].Inputs, Not(IsNil))
	c.Assert(p2.Tasks["customer"].Inputs, HasLen, 3)
	c.Assert(p2.Tasks["size"].Inputs, HasLen, 1)
	c.Assert(p2.Tasks["customer"].Inputs["datacenter"].Dstring, Equals, "JP-12345678910-01")
	c.Assert(p2.Tasks["customer"].Inputs["name"].Dstring, Equals, "Bob")
	c.Assert(p2.Tasks["customer"].Inputs["num"].Dint, Equals, 66)
	c.Assert(p2.Tasks["size"].Inputs["size"].Dstring, Equals, "Small")

	// p1 has no Reults
	for task := range p1.Tasks {
		c.Assert(p1.Tasks[task].HasResult, Equals, false)
		c.Assert(p1.Tasks[task].Result, Equals, (*auto.DataT)(nil))
		c.Assert(p1.Tasks[task].Working, Equals, false)
	}

	// p2 has Results
	for task := range p2.Tasks {
		c.Assert(p2.Tasks[task].HasResult, Equals, true)
		c.Assert(p2.Tasks[task].Result, Not(IsNil))
		c.Assert(p2.Tasks[task].Result, Not(DeepEquals),
			auto.DataT{Dbool: false, Dint: 0, Dnuid: "", Dstring: "", Derr: error(nil)})
		c.Assert(p2.Tasks[task].Working, Equals, false) // Save point should be after task completed!
	}

	// Strings that should be in p2 task Results
	expect := map[string]string{}
	expect["boot"] = "Booted container"
	expect["customer"] = "Bob"
	expect["node"] = "Node"
	expect["provision"] = "Provisioning: Node"
	expect["size"] = "Size: Small"
	expect["image"] = "Image For"
	notime, errt := time.ParseDuration("0s")
	c.Assert(errt, IsNil)

	// time should have passed, strings should be present in p2
	for task := range p1.Tasks {
		c.Assert(p1.Tasks[task].Forward.Elapsed, Equals, notime)
		c.Assert(p1.Tasks[task].Forward.Elapsed, Not(Equals), p2.Tasks[task].Forward.Elapsed)
		c.Assert(p1.Tasks[task].Forward.Ts, Not(Equals), p2.Tasks[task].Forward.Ts)
		// Check Results
		c.Assert(strings.HasPrefix(p2.Tasks[task].Result.Dstring, expect[task]), Equals, true)
		// No inverse motion
		c.Assert(p2.Tasks[task].Inverse.Ts.IsZero(), Equals, true)
		c.Assert(p1.Tasks[task].Inverse.Ts.IsZero(), Equals, true)
	}

	// worker should be timestamped in p2, not p1; show elapsed time

	c.Assert(p1.Started.IsZero(), Equals, true)
	c.Assert(p1.Started, Not(Equals), p2.Started)
	c.Assert(p1.Elapsed, Equals, notime)
	c.Assert(p1.Elapsed, Not(Equals), p2.Elapsed)

	// UndoWorkflow

	fmt.Printf("\n\n\nWith (p_2) Undo/Redo Worker (p_3) - invalidate task node with Rmv()  \n")

	// Make the event to invalidate the node found by the worker
	bevent := auto.EventT{}
	bevent.Msg = "Rmv"
	bevent.Workerid = Provisioner.Workerid
	bevent.Broadcast = false
	bevent.Timeless = true
	bevent.Taskkey = "node"
	bevent.Data.Dnuid = Provisioner.Tasks["node"].Result.Dnuid
	var uid string = bevent.Data.Dnuid
	bevent.Data.Dstring = fmt.Sprintf("Node %s failed", uid)
	Provisioner.Events <- bevent

	for {
		time.Sleep(time.Second / TIMESLICE)
		if !Provisioner.FwdGoal() {
			time.Sleep(time.Second)
			break
		}
	}

	// Emit JSON

	pathspec = []string{s.dir, "p_3.json"}
	path3 := strings.Join(pathspec, "")
	writerr = Provisioner.WorkerToFile(path3)
	c.Assert(writerr, IsNil)

	// WorkflowUndone

	fmt.Printf("\n\n\nCompare Undone/Redo (p_3) Worker with Completed Provision Worker (p_2) \n")

	p3, err3 := auto.WorkerFromFile(path3)
	c.Assert(err3, IsNil)
	p2, err2 = auto.WorkerFromFile(path2)
	c.Assert(err2, IsNil)
	c.Assert(p3.Tasks, HasLen, 6)
	c.Assert(p2.Workerid, Equals, p3.Workerid)

	// Are p3 inputs intact?
	c.Assert(p3.Name, Equals, "provisiontest")
	c.Assert(p3.Tasks["customer"].Inputs, Not(IsNil))
	c.Assert(p3.Tasks["size"].Inputs, Not(IsNil))
	c.Assert(p3.Tasks["customer"].Inputs, HasLen, 3)
	c.Assert(p3.Tasks["size"].Inputs, HasLen, 1)
	c.Assert(p3.Tasks["customer"].Inputs["datacenter"].Dstring, Equals, "JP-12345678910-01")
	c.Assert(p3.Tasks["customer"].Inputs["name"].Dstring, Equals, "Bob")
	c.Assert(p3.Tasks["customer"].Inputs["num"].Dint, Equals, 66)
	c.Assert(p3.Tasks["size"].Inputs["size"].Dstring, Equals, "Small")

	// p3 has Results
	for task := range p3.Tasks {
		c.Assert(p3.Tasks[task].HasResult, Equals, true)
		c.Assert(p3.Tasks[task].Result, NotNil)
		c.Assert(p3.Tasks[task].Result, Not(DeepEquals),
			auto.DataT{Dbool: false, Dint: 0, Dnuid: "", Dstring: "", Derr: error(nil)})
		c.Assert(p3.Tasks[task].Working, Equals, false) // Save point should be after task completed!
	}

	// Strings that should be in p3 task Results
	expect = map[string]string{}
	expect["boot"] = "Booted container"
	expect["customer"] = "Bob"
	expect["node"] = "Node"
	expect["provision"] = "Provisioning: Node"
	expect["size"] = "Size: Small"
	expect["image"] = "Image For"
	notime, errt = time.ParseDuration("0s")
	c.Assert(errt, IsNil)

	// same prefix strings should be present in p3
	for task := range p3.Tasks {
		c.Assert(p2.Tasks[task].Inverse.Ts.IsZero(), Equals, true)
		c.Assert(p2.Tasks[task].Inverse.Elapsed, Equals, notime)
		// Check Results (same prefixes should apply, ID should differ
		c.Assert(strings.HasPrefix(p2.Tasks[task].Result.Dstring, expect[task]), Equals, true)
	}

	// p3 should record elapsed time, timestamps in node, provision, boot only

	c.Assert(p3.Tasks["customer"].Inverse.Elapsed, Equals, notime)
	c.Assert(p3.Tasks["customer"].Inverse.Ts.IsZero(), Equals, true)
	c.Assert(p3.Tasks["image"].Inverse.Elapsed, Equals, notime)
	c.Assert(p3.Tasks["image"].Inverse.Ts.IsZero(), Equals, true)
	c.Assert(p3.Tasks["size"].Inverse.Elapsed, Equals, notime)
	c.Assert(p3.Tasks["size"].Inverse.Ts.IsZero(), Equals, true)

	c.Assert(p3.Tasks["node"].Inverse.Elapsed, Not(Equals), notime)
	c.Assert(p3.Tasks["provision"].Inverse.Elapsed, Not(Equals), notime)
	c.Assert(p3.Tasks["boot"].Inverse.Elapsed, Not(Equals), notime)
	// New Node NUID, Provisioning NUID should appear in p3

	// p3 should have new NUIDs in node, provision, boot

	c.Assert(p2.Tasks["node"].Result.Dnuid, Not(DeepEquals), p3.Tasks["node"].Result.Dnuid)
	c.Assert(p2.Tasks["provision"].Result.Dnuid, Not(DeepEquals), p3.Tasks["provision"].Result.Dnuid)
	c.Assert(p2.Tasks["boot"].Result.Dnuid, Not(DeepEquals), p3.Tasks["boot"].Result.Dnuid)

	// worker should be timestamped differently in p2, p3

	c.Assert(p2.Elapsed, Not(Equals), p3.Elapsed)

	// ProvisionFailedWorkflow

	fmt.Printf("\n\n\nInject error into holding (p_3) Provision Worker to make failed worker (p_4) \n")

	// Use canned FailWorker which makes the event and sends it to itself.
	Provisioner.FailWorker("node", fmt.Errorf("400 Malformed Node request from Client"))

	for {
		time.Sleep(time.Second / TIMESLICE)
		if !Provisioner.FwdGoal() {
			time.Sleep(time.Second)
			break
		}
	}

	// Emit JSON

	pathspec = []string{s.dir, "p_4.json"}
	path4 := strings.Join(pathspec, "")
	writerr = Provisioner.WorkerToFile(path4)
	c.Assert(writerr, IsNil)

	// Notes - currently this leaves just timestamps.

	// FailedWorkflow

	p3, err3 = auto.WorkerFromFile(path3)
	c.Assert(err3, IsNil)
	p4, err4 := auto.WorkerFromFile(path4)
	c.Assert(err4, IsNil)
	c.Assert(p4.Tasks, HasLen, 6)
	c.Assert(p4.Workerid, Equals, p3.Workerid)

	// Are p4 inputs intact? (even though they are invalid, for posterity)
	c.Assert(p4.Name, Equals, "provisiontest")
	c.Assert(p4.Tasks["customer"].Inputs, Not(IsNil))
	c.Assert(p4.Tasks["size"].Inputs, Not(IsNil))
	c.Assert(p4.Tasks["customer"].Inputs, HasLen, 3)
	c.Assert(p4.Tasks["size"].Inputs, HasLen, 1)
	c.Assert(p4.Tasks["customer"].Inputs["datacenter"].Dstring, Equals, "JP-12345678910-01")
	c.Assert(p4.Tasks["customer"].Inputs["name"].Dstring, Equals, "Bob")
	c.Assert(p4.Tasks["customer"].Inputs["num"].Dint, Equals, 66)
	c.Assert(p4.Tasks["size"].Inputs["size"].Dstring, Equals, "Small")

	// p4 has NO Results (they have all been undone)
	for task := range p4.Tasks {
		c.Assert(p4.Tasks[task].HasResult, Equals, false)
		c.Assert(p4.Tasks[task].Result, Equals, (*auto.DataT)(nil))
		c.Assert(p4.Tasks[task].Working, Equals, false) // Save point should be after task completed!
	}

	notime, _ = time.ParseDuration("0s")

	// All Inverse tasks should have fired:
	for task := range p4.Tasks {
		c.Assert(p4.Tasks[task].Inverse.Ts.IsZero(), Equals, false)
		c.Assert(p4.Tasks[task].Inverse.Elapsed, Not(Equals), notime)
	}

	// p4 should have recorded the FAIL error message in Node node
	c.Assert(strings.HasPrefix(p4.Tasks["node"].Forward.Errs[0], "400 Malformed Node"), Equals, true)

	Provisioner.Release() // Allows the worker to remove itself
	Provisioner.Done()    // Sends the event to close the worker
	Hub1.Stop(false)

	// ReloadWorkflow

	fmt.Printf("\n\n\nReload Provision Worker (p_1) \n")

	fmt.Printf("\n\n\nReload Workflow from JSON (p_1) and run it to make (p_5)\n")

	ProvisionerLoaded, perr := LoadProvisionWorker(path1)
	c.Assert(perr, IsNil)

	ProvisionerLoaded.SetHold() // Keep in memory after goal attained
	Hub2 := auto.StartNewHub(0)
	Hub2.Launch(&ProvisionerLoaded, false)

	for {
		time.Sleep(time.Second / TIMESLICE)
		if !Provisioner.FwdGoal() {
			time.Sleep(time.Second)
			break
		}
	}

	// Emit JSON
	pathspec = []string{s.dir, "p_5.json"}
	path5 := strings.Join(pathspec, "")
	writerr = ProvisionerLoaded.WorkerToFile(path5)
	c.Assert(writerr, IsNil)

	// Reload - Workflow Complete

	p2, err2 = auto.WorkerFromFile(path5)
	c.Assert(err2, IsNil)
	c.Assert(p2.Tasks, HasLen, 6)

	// Are p2 inputs intact?
	c.Assert(p2.Name, Equals, "provisiontest")
	c.Assert(p2.Tasks["customer"].Inputs, Not(IsNil))
	c.Assert(p2.Tasks["size"].Inputs, Not(IsNil))
	c.Assert(p2.Tasks["customer"].Inputs, HasLen, 3)
	c.Assert(p2.Tasks["size"].Inputs, HasLen, 1)
	c.Assert(p2.Tasks["customer"].Inputs["datacenter"].Dstring, Equals, "JP-12345678910-01")
	c.Assert(p2.Tasks["customer"].Inputs["name"].Dstring, Equals, "Bob")
	c.Assert(p2.Tasks["customer"].Inputs["num"].Dint, Equals, 66)
	c.Assert(p2.Tasks["size"].Inputs["size"].Dstring, Equals, "Small")

	// p2 has Results
	for task := range p2.Tasks {
		c.Assert(p2.Tasks[task].HasResult, Equals, true)
		c.Assert(p2.Tasks[task].Result, Not(IsNil))
		c.Assert(p2.Tasks[task].Result, Not(DeepEquals),
			auto.DataT{Dbool: false, Dint: 0, Dnuid: "", Dstring: "", Derr: error(nil)})
		c.Assert(p2.Tasks[task].Working, Equals, false) // Save point should be after task completed!
	}

	for task := range p1.Tasks {
		c.Assert(p1.Tasks[task].Forward.Elapsed, Not(Equals), p2.Tasks[task].Forward.Elapsed)
		c.Assert(p1.Tasks[task].Forward.Ts, Not(Equals), p2.Tasks[task].Forward.Ts)
		// Check Results
		c.Assert(strings.HasPrefix(p2.Tasks[task].Result.Dstring, expect[task]), Equals, true)
		// No inverse motion
		c.Assert(p2.Tasks[task].Inverse.Ts.IsZero(), Equals, true)
		c.Assert(p1.Tasks[task].Inverse.Ts.IsZero(), Equals, true)
	}

	// worker should be timestamped in p2, not p1; show elapsed time

	c.Assert(p1.Started, Not(Equals), p2.Started)

	ProvisionerLoaded.Release() // Allows the worker to remove itself
	ProvisionerLoaded.Done()    // Sends the event to close the worker
	Hub2.Stop(false)
	time.Sleep(time.Second)

	// Too Slow Workflow - Timeout, Exausts Retries, then Fallback
	fmt.Printf("\n\n\nTest too slow workflow (from p_1)- exhaust retries, fallbck (p_6) \n")

	ProvisionerSlow, perr := LoadProvisionWorker(path1)
	c.Assert(perr, IsNil)
	ProvisionerSlow.Tasks["provision"] = SlowProvisionTask(ProvisionerSlow.Tasks["provision"])

	ProvisionerSlow.SetHold()
	Hub3 := auto.StartNewHub(0)
	Hub3.Launch(&ProvisionerSlow, false)

	for {
		time.Sleep(time.Second / TIMESLICE)
		if !ProvisionerSlow.FwdGoal() {
			time.Sleep(time.Second)
			break
		}
	}

	// Emit JSON
	pathspec = []string{s.dir, "p_6.json"}
	path6 := strings.Join(pathspec, "")
	writerr = ProvisionerSlow.WorkerToFile(path6)
	c.Assert(writerr, IsNil)

	ProvisionerSlow.Release() // Allows the worker to remove itself
	ProvisionerSlow.Done()    // Sends the event to close the worker
	Hub3.Stop(false)
	time.Sleep(time.Second)

	fmt.Printf("\n\n\nToo Slow Timeout and Fallback (from p_1), Inv Triggered (p_7)\n")

	// Too Slow Timeout and Too Slow Fallback, Triggers Inv workflow

	ProvisionerFBSlow, perr := LoadProvisionWorker(path1)
	c.Assert(perr, IsNil)
	ProvisionerFBSlow.Tasks["provision"] = SlowProvisionFallback(ProvisionerFBSlow.Tasks["provision"])

	ProvisionerFBSlow.SetHold()
	Hub4 := auto.StartNewHub(0)
	Hub4.Launch(&ProvisionerFBSlow, false)

	for {
		time.Sleep(time.Second / TIMESLICE)
		if ProvisionerFBSlow.Dienow {
			time.Sleep(time.Second)
			break
		}
	}

	// Emit JSON
	pathspec = []string{s.dir, "p_7.json"}
	path7 := strings.Join(pathspec, "")
	writerr = ProvisionerFBSlow.WorkerToFile(path7)
	c.Assert(writerr, IsNil)

	ProvisionerFBSlow.Release() // Allows the worker to remove itself
	ProvisionerFBSlow.Done()    // Sends the event to close the worker
	Hub4.Stop(false)

	time.Sleep(time.Second)
	fmt.Printf("\n\n\nToo Slow Timeout (from p_1), Exhaust Retries, FAILS (p_8)\n")

	// Too Slow Timeout and Too Slow Fallback, Fails

	ProvisionerFailSlow, perr := LoadProvisionWorker(path1)
	c.Assert(perr, IsNil)
	ProvisionerFailSlow.Tasks["provision"] = SlowProvisionFallback(ProvisionerFailSlow.Tasks["provision"])
	ProvisionerFailSlow.UndoOnFail = false

	ProvisionerFailSlow.SetHold()
	Hub5 := auto.StartNewHub(0)
	Hub5.Launch(&ProvisionerFailSlow, false)

	time.Sleep(time.Second)

	// Emit JSON
	pathspec = []string{s.dir, "p_8.json"}
	path8 := strings.Join(pathspec, "")
	writerr = ProvisionerFailSlow.WorkerToFile(path8)
	c.Assert(writerr, IsNil)

	ProvisionerFailSlow.Release() // Allows the worker to remove itself
	ProvisionerFailSlow.Done()    // Sends the event to close the worker
	Hub5.Stop(false)
}
