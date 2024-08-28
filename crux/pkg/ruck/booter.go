// (c) Ericsson Inc. 2015-2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package ruck

import (
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/auto"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/register"
	"github.com/erixzone/crux/pkg/rucklib"
	"github.com/erixzone/crux/pkg/steward"

	"golang.org/x/net/context"
)

// FlockParamT - arguments for starting the flock.
type FlockParamT struct {
	Block     string        `json:"block"`
	Port      int           `json:"port"`
	Skey      string        `json:"skey"`
	Ipname    string        `json:"ipname"`
	IP        string        `json:"ip"`
	Horde     string        `json:"horde"`
	Sleeptime time.Duration `json:"sleeptime"`
	Minlimit  time.Duration `json:"minlimit"`
	Maxlimit  time.Duration `json:"maxlimit"`
}

// FLOCKPARAMS - Encoding type int for FlockParamsT unmarshalling from an EventT
const FLOCKPARAMS int = 4551

// PUBKEYJSON - Encoding type int for PubKeyT
const PUBKEYJSON int = 3441

// WORKFLOW TASK - "flock"

// StartFlock First Task - forward - starts flocking protocol.
func StartFlock(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	fps := FlockParamT{}
	fps.Block = params["block"].Dstring
	fps.Port = params["port"].Dint
	fps.Skey = params["skey"].Dstring
	fps.Ipname = params["ipname"].Dstring
	fps.IP = params["ip"].Dstring
	fps.Horde = params["horde"].Dstring
	beacon := params["beacon"].Dstring
	networks := params["networks"].Dstring
	certdir := params["certdir"].Dstring
	visitor := params["visitor"].Dbool

	// test the Log Nabla event
	*done <- sase.Nabla(LOGEVENT).Str("StartFlock()")

	net := newFlock(fps.Port, fps.Skey, fps.Ipname, fps.IP, beacon, networks, certdir, visitor)
	if fps.Ipname == "" {
		fps.Ipname = net.GetNames().Node
	}
	if fps.IP == "" {
		fps.IP = fps.Ipname
	}
	Rs.Lock()
	Rs.Net = net
	var x crux.Confab = net
	Rs.Conf = &x
	Rs.IP = fps.IP
	Rs.Ipname = fps.Ipname
	Rs.Unlock()

	fps.Sleeptime = time.Duration(2.1 * float32(net.GetFflock().Heartbeat())) // just a little over two heartbeats should be good
	fps.Minlimit = 8 * net.GetFflock().NodePrune()                            // minimum time limit for stability to emerge
	fps.Maxlimit = 200 * time.Second                                          // max time limit for stability to emerge

	var c crux.Confab = net
	*done <- sase.Nabla(LOGEVENT).Str("StartFlock() - flocking system started")

	// Compose, send the Got event to with the crux.Confab interface
	// which can be passed on to FlockEvents
	jsonfps, _ := json.Marshal(fps)
	*done <- sase.Got().Iface(c).Jsn(jsonfps, FLOCKPARAMS) // Pass back interface + params as JSON for next task
}

// CloseFlock First Task - inverse - stops flocking protocol.
func CloseFlock(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Result should have the crux.Confab in it if it is still valid
	// we use it to call Flock.Close()
	*done <- sase.Nabla(LOGEVENT).Str("CloseFlock()")
	r, ok := params["result"]
	if ok {
		c := r.Dface
		if c != nil {
			confab := c.(crux.Confab)
			// NOTE - something is still reading UDP packets in the flocking protocol...
			// There is an unstopped go-routine logging...
			// So - don't block workflow - this is slow to return.
			closeit := func(c crux.Confab) {
				c.Close()
			}
			go closeit(confab)
			*done <- sase.Nabla(LOGEVENT).Str("CloseFlock() - flocking system closed")
		}
	}
	w.Hold = false
	*done <- sase.Inv()
}

func makeFlockTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("flock")
	task = task.AddForwardFn(StartFlock, 50*time.Millisecond, auto.ConfPROD, auto.TransitRPC, 0)
	task = task.AddInverseFn(CloseFlock, 50*time.Millisecond, auto.ConfPROD, auto.TransitRPC, 0)
	task = task.AddReverseDepends([]string{"stable"})
	return task
}

// WORKFLOW TASK - Muck - First task (concurrent with flock) "muck" - initializes local storage, the principal identifier

// StartMuck - start task that initializes .muck directory and principal identifier
func StartMuck(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StartMuck()")
	// establish a new .muck, or find the existing one, with default naming scheme
	// Modify here to pass in here a command line directory and/or a provided principal name
	merr := muck.InitMuck("", "")
	if merr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("%v", merr))
		return
	}
	principal, perr := muck.Principal()
	if perr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("%v", perr))
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("StartMuck() .muck is ready, principal is %s", principal))
	*done <- sase.Got().Str(principal)
}

// EndMuck - nil operation - muck could be tarballed/saved here or cleaned up.
func EndMuck(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("EndMuck()")
	// Invalidate prior result
	*done <- sase.Inv()
}

func makeMuckTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("muck")
	task = task.AddForwardFn(StartMuck, 100*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(EndMuck, 50*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddReverseDepends([]string{"selfkeys"})
	return task
}

// WORKFLOW TASK - Self-Keys "selfkeys" task - uses ssh-agent to make self keys, tests ssh-agent i/o

// NewSelfKeys - task that initializes self keys, ssh-agent and tests ssh-agent can read keys
func NewSelfKeys(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("NewSelfKeys()")
	// init self keys and ssh-agent
	kerr := grpcsig.InitSelfSSHKeys(false)
	if kerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("NewSelfKeys() :  InitSelfSSHKeys failed : %v", kerr))
		return
	}

	// quietly ensure sure ssh-agent can read keys
	_, lerr := grpcsig.ListKeysFromAgent(false)
	if lerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("NewSelfKeys() : ListKeysFromAgent failed accessing private keys with ssh-agent : %v", lerr))
		return
	}
	selfkey := grpcsig.GetSelfPubKey()
	selfkeyjson, jerr := grpcsig.PubKeyToJSONBytes(selfkey)
	if jerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("NewSelfKeys() : PubKeyToJSON failed : %v", jerr))
		return
	}
	*done <- sase.Got().Jsn(selfkeyjson, PUBKEYJSON)
}

// RemoveSelfKeys - removes self key files from .muck
func RemoveSelfKeys(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RemoveSelfKeys()")
	err := grpcsig.FiniSelfSSHKeys(false)
	if err != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RemoveSelfKeys() :  FiniSelfSSHKeys failed : %v", err))
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str("RemoveSelfKeys() self keys removed from .muck")
	*done <- sase.Inv()
}

func makeSelfKeysTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("selfkeys")
	task = task.AddForwardFn(NewSelfKeys, 200*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(RemoveSelfKeys, 100*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddDepends([]string{"muck"})
	task = task.AddAllDepends([]string{"muck"})
	task = task.AddReverseDepends([]string{"whitelistdb"})
	return task
}

// TODO search Nabla consistently apply function name ()

// WORKFLOW TASK - Whitelist DB "whitelistdb" - Starts BoltDB system, installs self-key

// StartWhitelistDB - task that initializes self keys, ssh-agent and tests ssh-agent can read keys
func StartWhitelistDB(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StartWhitelistDB()")
	// extract self public key
	selfkey := grpcsig.PubKeyT{}
	jerr := json.Unmarshal(params["selfkeys"].Djson, &selfkey)
	if jerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("StartWhitelistDB() : Unmarshal failed : %v", jerr))
		return
	}
	dbfile := muck.Dir() + "/whitelist.db"
	if !grpcsig.PubKeyDBExists(dbfile) {
		derr := grpcsig.StartNewPubKeyDB(dbfile)
		if derr != nil {
			*done <- sase.Fail().Err(fmt.Errorf("StartWhitelistDB() : StartNewPubKeyDB failed : %v", derr))
			return
		}
	}
	dblog := clog.Log.With("focus", "whitelistdb")
	ierr := grpcsig.InitPubKeyLookup(dbfile, dblog)
	if ierr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("StartWhitelistDB() : InitPubKeyLookup failed : %v", ierr))
		return
	}

	// Add self public key to whitelist DB
	kerr := grpcsig.AddPubKeyToDB(&selfkey)
	if kerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("StartWhitelistDB() : AddPubKeyToDB failed with self key: %v", kerr))
		return
	}
	// close for now - reeve will start it up in InitDefaultService
	grpcsig.FiniPubKeyLookup()
	// return db filename, and the db logger interface
	*done <- sase.Nabla(LOGEVENT).Str("StartWhitelistDB() started, session self key added")
	*done <- sase.Got().Str(dbfile).Iface(dblog)
}

// StopWhitelistDB - removes self keys from db and closes it.
func StopWhitelistDB(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StopWhitelistDB()")
	// Extract the dbfile and logger from the results, ensure results are valid
	msg := ""
	dbfile := ""
	var dblog clog.Logger
	r, ok := params["result"]
	if ok {
		dbfile = r.Dstring
		if r.Dface != nil {
			dblog = r.Dface.(clog.Logger)
		} else {
			msg = "StopWhiteListDB() failed : no database logger interface"
		}
	} else {
		msg = "StopWhiteListDB() failed : unable to extract dbfile, logger from prior result"
	}
	if len(dbfile) == 0 {
		msg = msg + "StopWhiteListDB() failed : unable to extract dbfile, from prior result"
	}
	if len(msg) > 0 {
		*done <- sase.Fail().Err(fmt.Errorf(msg))
		return
	}
	// Remove self key from whitelistdb
	//   close if already open, open the DB
	ierr := grpcsig.DBRestart(dbfile, dblog)
	//    remove the self keys
	derr := grpcsig.RemoveSelfPubKeysFromDB()
	//    close the whitelist database
	grpcsig.FiniPubKeyLookup()
	// Return any errors accumulated.
	if ierr != nil {
		msg = fmt.Sprintf("StopWhitelistDB() : grpcsig.DBRestart() failed : %v", ierr)
	}
	if derr != nil {
		msg = msg + fmt.Sprintf("StopWhitelistDB() : grpcsig.RemoveSelfPubKeyFromDB() failed : %v", derr)
	}
	if len(msg) > 0 {
		*done <- sase.Fail().Err(fmt.Errorf(msg))
		return
	}
	// All good
	*done <- sase.Nabla(LOGEVENT).Str("StopWhitelistDB() stopped - session self key removed")
	*done <- sase.Inv()
}

func makeWhitelistDBTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("whitelistdb")
	task = task.AddForwardFn(StartWhitelistDB, 200*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(StopWhitelistDB, 100*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddDepends([]string{"selfkeys"})
	task = task.AddAllDepends([]string{"muck", "selfkeys"})
	task = task.AddReverseDepends([]string{"reeveapi"}) // CWVH fix
	return task
}

// WORKFLOW TASK - "stable"

// FlockStable -- Waits for stable emergence of flock leader, steward, reeve host.
func FlockStable(c crux.Confab, sleeptime, minlimit, maxlimit time.Duration, regkey string, logboot clog.Logger, w *auto.WorkerT) (cluster, leader, me, regaddress, stewaddress, fkey string, eret error) {
	var stable bool
	start := time.Now()
	var wait1, total time.Duration
	iter := 0
	logboot.Log(nil, "flockstable: sleep=%s min=%s max=%s", sleeptime.String(), minlimit.String(), maxlimit.String())

	// Attempts to give the flocking protocol time to restart if muliple leaders emerge
	// Flock leader and host of registry/steward must be non-0, self-consistent to escape
	for {
		// Delay 1: Wait a minimum time for stable flag to emerge from flocking protocol
		start1 := time.Now()
		j := 0
		for {
			j++
			time.Sleep(sleeptime)
			if w.CancelTask("stable") { // Have we been cancelled by worker runner while waiting for this?
				logboot.Log(nil, "FlockStable() Cancelled By Worker")
				eret = fmt.Errorf("Cancelled by Worker")
				return
			}

			cn := c.GetNames()
			cluster, leader, stable, me = cn.Bloc, cn.Leader, cn.Stable, cn.Node
			logboot.Log("FlockStable() c.GetNames()", fmt.Sprintf("%d.%d", iter, j), "cluster", cluster, "leader", leader, "stable", stable, "me", me)
			wait1 = time.Now().Sub(start1)
			if wait1 > minlimit {
				if stable {
					break
				}
			}
			total = time.Now().Sub(start)
			if total > maxlimit {
				logboot.Log(nil, "FlockStable() Delay 1 iter %d - Flock not stable after %s- aborting service start", iter, total.String())
				eret = fmt.Errorf("Timeout")
			}
		}

		// Skip first time -
		// test previously circulated ports for consistency with leader, then we can proceed
		if iter > 0 {
			cn := c.GetNames()
			regaddress, fkey, stewaddress = cn.RegistryAddr, cn.RegistryKey, cn.Steward
			logboot.Log("FlockStable() c.GetRegistry()", fmt.Sprintf("%d", iter), "registry", regaddress, "flockkey", fkey, "steward", stewaddress)
			// Did we recover the intended port numbers (non-0) back from flocking?
			if stewPort == idutils.SplitPort(stewaddress) && regPort == idutils.SplitPort(regaddress) {
				// Is the host for steward/registry same as the leader?
				if leader == idutils.SplitHost(stewaddress) && leader == idutils.SplitHost(regaddress) {
					if leader == me {
						logboot.Log(nil, fmt.Sprintf("FlockStable() %d I AM LEADER", iter))
					}
					// Commit to this being stable for now, bring up services
					eret = nil
					return // with named variables
				}
			}
		}

		// On first iteration or when last iteration was not consistent
		// Current Leader "candidate" injects proposed registry/steward values
		if leader == me {
			logboot.Log(nil, "FlockStable() %d I THINK I MAY BE LEADER (reg=%d stew=%d)", iter, regPort, stewPort)
			if len(fkey) == 0 {
				c.SetRegistry(me, regPort, regkey)
			} else {
				c.SetRegistry(me, regPort, fkey)
			}
			c.SetSteward(me, stewPort)
		}
		iter++
	}
}

// Stable - task function to wait for flock to become stable using FlockStable()
func Stable(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Extract the confab and arguments from previous task
	*done <- sase.Nabla(LOGEVENT).Str("Stable()")
	f, ok := params["flock"]
	if !ok {
		// can't do anything.
		*done <- sase.Fail().Err(fmt.Errorf("Stable - no \"flock\" result found"))
	}
	confab := f.Dface.(crux.Confab)
	fps := FlockParamT{}
	_ = json.Unmarshal(params["flock"].Djson, &fps)

	// Wait for the initial flock to stablize, (it may restart flocking on its own)
	// declare consistent leadership and locations for startup of registry & steward
	reevekey := fps.Skey
	watchboot := clog.Log.With("focus", "stable", "node", fps.Ipname)
	w.ClearTask("stable") // Clear any prior cancellation immediately
	cluster, leader, me, regaddress, stewaddress, fkey, err :=
		FlockStable(confab, fps.Sleeptime, fps.Minlimit, fps.Maxlimit, reevekey, watchboot, w)
	if err != nil {
		*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("Stable is cancelled - %v", err))
		return // We were cancelled by workflow, so exit without sending result event

	}
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("Stable() leader = %s Me = %s regaddress = %s stewaddress = %s fkey = %s",
		leader, me, regaddress, stewaddress, fkey))
	*done <- sase.Got().Str(cluster)
}

// UndoStable - cancels any running FlockStable() and invalidates any saved output from Stable
func UndoStable(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Send an event to stop the FlockStable function loop if it is still in a wait state
	*done <- sase.Nabla(LOGEVENT).Str("UndoStable()")
	event := auto.EventT{}
	event.Msg = auto.CANCEL
	event = event.To(w.Workerid).Task("stable").NoExpiry()
	*done <- event
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeStableTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("stable")
	task = task.AddForwardFn(Stable, 210*time.Second, auto.ConfPROD, auto.TransitLAN, 0) //
	//  Seen on myriad:
	//     "to":     210000000000,
	//     "elapsed": 50406734608
	task = task.AddInverseFn(UndoStable, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddDepends([]string{"flock"})
	task = task.AddReverseDepends([]string{"watchleader", "watchregistry", "watchsteward", "stewarddb"})
	task = task.AddAllDepends([]string{"flock"})
	return task
}

// WORKFLOW TASK - "watchleader" - monitors flocking for changes in flock leader.

// WatchLeader - workflow task that watches the flock stable & leader settings via confab, monitors for cancel event
func WatchLeader(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Extract the confab and arguments from previous task
	*done <- sase.Nabla(LOGEVENT).Str("WatchLeader()")
	confab := params["flock"].Dface.(crux.Confab)
	oleader := confab.GetNames().Leader
	w.ClearTask("watchleader")
	mon := func(oleader string, confab crux.Confab, delay time.Duration, sase auto.EventT, w *auto.WorkerT) {
		for {
			if w.CancelTask("watchleader") { // Have we been cancelled by worker runner while waiting for this?
				*done <- sase.Nabla(LOGEVENT).Str("WatchLeader() cancelled")
				return
			}
			cn := confab.GetNames()
			leader, stable := cn.Leader, cn.Stable
			if !stable {
				// Back it all up and re-run "stable"
				*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("WatchLeader() sees unstable flock!"))
				*done <- sase.Task("stable").Rmv().NoExpiry() // Invalidate anything we previously reported, trigger reverse workflow.
				return                                        // only trigger the event once
			}
			if leader != oleader {
				// Leader is changed
				*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("WatchLeader() leader changed from %s to %s", oleader, leader))
				*done <- sase.Task("stable").Rmv().NoExpiry() // Invalidate anything we previously reported, trigger reverse workflow.
				return                                        // only trigger the event once
			}
			time.Sleep(delay)
		}
	}
	go mon(oleader, confab, 2*time.Second, sase, w)
	*done <- sase.Got().Str(oleader)
}

// UnWatchLeader - workflow reverse task - sends an event to stop WatchLeader, invalidates old output
func UnWatchLeader(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Send an event to stop the WatchLeader function loop if it is still in a wait state
	*done <- sase.Nabla(LOGEVENT).Str("UnWatchLeader()")
	event := auto.EventT{}
	event.Msg = auto.CANCEL
	event = event.To(w.Workerid).Task("watchleader").NoExpiry()
	*done <- event
	// Wipe any saved task output
	*done <- sase.Inv()
}

func makeWatchLeaderTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("watchleader")
	task = task.AddForwardFn(WatchLeader, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddInverseFn(UnWatchLeader, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddDepends([]string{"flock", "stable"})
	task = task.AddReverseDepends([]string{"reeveapi"})
	task = task.AddAllDepends([]string{"flock", "stable"})
	return task
}

// WORKFLOW TASK - "watchregistry" - monitors flocking for changes in node registry.

// WatchRegistry - workflow task that watches the registry settings with confab, monitors for cancel event
func WatchRegistry(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Extract the confab and arguments from previous task
	*done <- sase.Nabla(LOGEVENT).Str("WatchRegistry()")
	confab := params["flock"].Dface.(crux.Confab)
	cn := confab.GetNames()
	oregaddress, oregkey := cn.RegistryAddr, cn.RegistryKey
	w.ClearTask("watchregistry") // Clear any prior cancellation immediately
	mon := func(oregaddress, oregkey string, confab crux.Confab, delay time.Duration, sase auto.EventT, w *auto.WorkerT) {
		for {
			if w.CancelTask("watchregistry") { // Have we been cancelled by worker runner while waiting for this?
				*done <- sase.Nabla(LOGEVENT).Str("WatchRegistry() cancelled")
				return
			}
			regaddress := confab.GetNames().RegistryAddr
			if regaddress != oregaddress {
				// Registry is changed
				*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("WatchRegistry() registry changed from %s to %s", oregaddress, regaddress))
				// Mark the output of this task as invalid in the workflow, trigger reverse workflow if not already
				*done <- sase.Rmv().NoExpiry() // Also wakes up holding worker
				return                         // only trigger the event once
			}
			time.Sleep(delay)
		}
	}
	go mon(oregaddress, oregkey, confab, 2*time.Second, sase, w)
	*done <- sase.Got().Str(oregaddress).ID(oregkey)
}

// UnWatchRegistry - workflow undo function - sends cancel event to WatchRegistry - invalidates prior output
func UnWatchRegistry(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Send an event to stop the WatchRegistry function loop if it is still in a wait state
	*done <- sase.Nabla(LOGEVENT).Str("UnWatchRegistry()")
	event := auto.EventT{}
	event.Msg = auto.CANCEL
	event = event.To(w.Workerid).Task("watchregistry").NoExpiry()
	*done <- event
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeWatchRegTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("watchregistry")
	task = task.AddForwardFn(WatchRegistry, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddInverseFn(UnWatchRegistry, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddDepends([]string{"flock", "stable"})
	task = task.AddReverseDepends([]string{"reeveapi"})
	task = task.AddAllDepends([]string{"flock", "stable"})
	return task
}

// WORKFLOW TASK - "watchsteward" - monitors flocking for changes in steward .

// WatchSteward - workflow task that watches the steward settings with confab, monitors for cancel event
func WatchSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	// Extract the confab and arguments from previous task
	*done <- sase.Nabla(LOGEVENT).Str("WatchSteward()")
	confab := params["flock"].Dface.(crux.Confab)
	ostewaddress := confab.GetNames().Steward
	w.ClearTask("watchsteward") // Clear any prior cancellation immediately
	mon := func(ostewaddress string, confab crux.Confab, delay time.Duration, sase auto.EventT, w *auto.WorkerT) {
		for {
			if w.CancelTask("watchsteward") { // Have we been cancelled by worker runner while waiting for this?
				*done <- sase.Nabla(LOGEVENT).Str("WatchSteward() cancelled")
				return
			}
			stewaddress := confab.GetNames().Steward
			if stewaddress != ostewaddress {
				// Steward is changed
				*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("WatchSteward() steward changed from %s to %s", ostewaddress, stewaddress))
				// Mark the output of this task as invalid in the workflow, trigger reverse workflow if not already
				*done <- sase.Rmv().NoExpiry() // Also wakes up holding worker
				return                         // only trigger the event once
			}
			time.Sleep(delay)
		}
	}
	go mon(ostewaddress, confab, 2*time.Second, sase, w)
	*done <- sase.Got().Str(ostewaddress)
}

// UnWatchSteward - workflow undo function - sends cancel event to WatchSteward - invalidates prior output
func UnWatchSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("UnWatchSteward()")
	// Send an event to stop the WatchSteward function loop if it is still in a wait state
	event := auto.EventT{}
	event.Msg = auto.CANCEL
	event = event.To(w.Workerid).Task("watchsteward").NoExpiry()
	*done <- event
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeWatchStewTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("watchsteward")
	task = task.AddForwardFn(WatchSteward, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddInverseFn(UnWatchSteward, 50*time.Millisecond, auto.ConfPROD, auto.TransitRAM, 0)
	task = task.AddDepends([]string{"flock", "stable"})
	task = task.AddReverseDepends([]string{"reeveapi"})
	task = task.AddAllDepends([]string{"flock", "stable"})
	return task
}

// WORKFLOW TASK - "reeveapi" - initialize reeveAPI stuff up to the reeveapi interface return

// StartReeveAPI - reeveAPI configured
func StartReeveAPI(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StartReeveAPI()")

	// TODO -error checking here.

	// Extract dbname from whitelistdb
	dbname := params["whitelistdb"].Dstring
	// Extract cluster name from stable
	//	cluster := params["stable"].Dstring
	// gather what we need from workflow parameters
	confab := params["flock"].Dface.(crux.Confab)
	stewaddress := confab.GetNames().Steward
	fps := FlockParamT{}
	_ = json.Unmarshal(params["flock"].Djson, &fps)
	logreeve := clog.Log.With("focus", "srv_reeve", "node", fps.Ipname)

	// Return an interface to reeveapi for its non-grpc services
	// for client grpcsig signing, and for server public key database lookups
	// which are local, pointer based structs that are passed via interface{}
	reeveapiif, err := startReeveAPI(dbname, fps.Block, fps.Horde, fps.Ipname, reevePort, stewaddress, confab.GetCertificate(), logreeve)
	if err != nil {
		// send  FAIL event with error
		msg := fmt.Sprintf("Failed to startReeveAPI :  %v", err)
		logreeve.Log("SEV", "ERROR", "node", fps.Ipname, msg)
		*done <- sase.Fail().Err(fmt.Errorf(msg))
		return
	}
	Rs.Lock()
	Rs.Reeveapi = reeveapiif
	Rs.Unlock()

	// Package up the return value - it is just the reeveapiif interface {}
	*done <- sase.Nabla(LOGEVENT).Str("ReeveAPI() started")
	*done <- sase.Got().Iface(reeveapiif) // Pass back interface
}

// StopReeveAPI - reeveAPI stop
func StopReeveAPI(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StopReeveAPI()")
	// grpcsig.FiniDefaultService() // CWVH this will knock out any higher-level grpc injectors
	// from checking pubkeys that may not be affected by a return to "stable"
	// For now - leave this as a no-op.
	*done <- sase.Inv()

	// TODO UNLOAD SignersFromCurrentPubkeys -this is safe to re-run but reloads
	// ssh-agent keys.  These can be removed with ssh-add -d
	//  (Maybe figure out what happens when duplicates are reloaded)
	/*
	   StopReeveAPI Actions to undo..
	   makes new 	reeve StateT needs to be removed
	   makes new 	ClientSigner for Steward (? keys can stay)
	   makes new	make reeve NOD/NID (? )
	   TODO 		Undo SignersFromCurrentPubkeys() - load existing keys
	   makes new	Undo prepare the reeve StateT{}
	   ?		Undo InitReeveKeys()	Make/Find reeve keys
	   call FiniDefaultService()	Undo InitDefaultService() - first grpcsig service with reeve.
	*/
}

func makeReeveAPITask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("reeveapi")
	task = task.AddForwardFn(StartReeveAPI, 250*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(StopReeveAPI, 200*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddDepends([]string{"flock", "stable", "whitelistdb"})
	task = task.AddReverseDepends([]string{"launchreeve"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward"})
	return task
}

// WORKFLOW TASK - "launchreeve" - launch reeve grpc service

// LaunchReeve - reeve server launch
func LaunchReeve(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("LaunchReeve()")
	var reeveapi rucklib.ReeveAPI
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("LaunchReeve() - no reeveapi task result"))
		return
	}
	reevenod, reevenid, _, _, reeveimp := reeveapi.ReeveCallBackInfo()

	rnod, _ := idutils.NodeIDParse(reevenod)
	rnid, _ := idutils.NetIDParse(reevenid)

	// We need a stop channel to stop the worker
	// plumbed into the worker.
	stop := make(chan bool)

	// ReeveLaunch starts the service and provides a  goroutine to do a gentle shutdown
	rerr := reeve.Launch(rnod, rnid, reeveimp, &stop)
	if rerr != nil {
		// Throw the FAIL event with error
		msg := fmt.Sprintf("LaunchReeve() reeveLaunch failed for %s/%s : %v", reevenod, reevenid, rerr)
		*done <- sase.Fail().Err(fmt.Errorf(msg))
		return
	}
	// ALL GOOD. We return with GOT event, and the quit channel
	*done <- sase.Nabla(LOGEVENT).Str("LaunchReeve() reeve started")
	*done <- sase.Got().QChan(&stop)
}

// RemoveReeve - reeve server graceful shutdown
func RemoveReeve(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RemoveReeve()")
	r, ok := params["result"]
	// Fish out the stop channel from the results of LaunchReeve
	if ok {
		quit := r.Dchan
		if quit != nil { // As it will be if registry is not running on node
			*quit <- true
			*done <- sase.Nabla(LOGEVENT).Str("RemoveReeve() - sent quit to reeve grpc server")
		} else {
			*done <- sase.Nabla(LOGEVENT).Str("RemoveReeve() - node not running reeve : nil quit channel")
		}
	} else {
		*done <- sase.Nabla(LOGEVENT).Str("RemoveReeve() error - no launchreeve task result ")
	}
	*done <- sase.Inv()
}

func makeLaunchReeveTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("launchreeve")
	task = task.AddForwardFn(LaunchReeve, 500*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(RemoveReeve, 1000*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddDepends([]string{"reeveapi"})
	task = task.AddReverseDepends([]string{"launchregistry"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi"})
	return task
}

// WORKFLOW TASK - "launchregistry" - starts the registry if node is the leader

// LaunchRegistry - registry launch on leader - placeholder
func LaunchRegistry(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("LaunchRegistry()")

	// Get the flocking informaiton
	confab := params["flock"].Dface.(crux.Confab)
	cn := confab.GetNames()
	leader, me := cn.Leader, cn.Node
	regaddress, regkey, stewaddress := cn.RegistryAddr, cn.RegistryKey, cn.Steward
	if leader != me { // No-op, all good
		*done <- sase.Nabla(LOGEVENT).Str("LaunchRegistry() not the leader")
		*done <- sase.Got()
		return
	}

	// Get the Flock parameters
	fps := FlockParamT{}
	_ = json.Unmarshal(params["flock"].Djson, &fps)

	// make a nodeid for Registry
	regnodeid, ferr := idutils.NewNodeID(fps.Block, fps.Horde, fps.Ipname, register.RegistryName, register.RegistryAPI)
	if ferr != nil {
		// send a Fail event
		msg1 := fmt.Sprintf("error - invalid nodeid params for registry: %v", ferr)
		*done <- sase.Fail().Err(fmt.Errorf(msg1))
	}

	// Use the reeveapi interface to get reeve service informaiton
	var reeveapi rucklib.ReeveAPI
	reeveapi = params["reeveapi"].Dface.(rucklib.ReeveAPI)
	_, reevenetid, _, _, reeveimp := reeveapi.ReeveCallBackInfo()
	if reeveimp == nil {
		// send the FAIL event with error
		msg2 := fmt.Sprintf("Failed reeveapi.ReeveCallBackInfo - no reeveimp")
		*done <- sase.Fail().Err(fmt.Errorf(msg2))
	}

	reevenid, ierr := idutils.NetIDParse(reevenetid)
	if ierr != nil {
		// Throw the FAIL event with error
		msg3 := fmt.Sprintf("failed to parse reevenetid : %v", ierr)
		*done <- sase.Fail().Err(fmt.Errorf(msg3))
	}

	// This is how long we give a reeve to execute a grpc callback
	reevetimeout := 10 * time.Second

	// Usually we would call reeveapi.SecureService(servicerev) for a service
	// but here, registry is not grpcsig secured. It has a reverse mechanism,
	// where it needs to do lookups based on the reeve servicerev, so...
	// We need a SecureService interface{} for reeve callback authentication
	imp := reeveapi.SecureService(reevenid.ServiceRev)
	if imp == nil {
		// Throw the FAIL event with error
		msg4 := fmt.Sprintf("Failed reeveapi.SecureService")
		*done <- sase.Fail().Err(fmt.Errorf(msg4))
	}

	// Now we can run the registry server
	// We need a stop channel to send in to this,
	// plumbed into the worker.
	stop := make(chan bool)

	// RegistryLaunch starts the service and a goroutine to do a gentle shutdown
	rerr := register.RegistryLaunch(regnodeid, regaddress, stewaddress, regkey, reevetimeout, imp, &stop)
	if rerr != nil {
		// Throw the FAIL event with error
		msg5 := fmt.Sprintf("Failed register.RegistryLaunch %s, %v", fps.Ipname, rerr)
		*done <- sase.Fail().Err(fmt.Errorf(msg5))
	}
	// ALL GOOD. We return with GOT event, and the quit channel
	*done <- sase.Nabla(LOGEVENT).Str("LaunchRegistry() registry server launched")
	*done <- sase.Got().QChan(&stop)
}

// RemoveRegistry - removes the registry with graceful shutdown
func RemoveRegistry(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RemoveRegistry()")
	r, ok := params["result"]
	// Fish out the quit channel from the results of LaunchRegistry
	if ok {
		quit := r.Dchan
		if quit != nil { // As it will be if registry is not running on node
			*quit <- true
			*done <- sase.Nabla(LOGEVENT).Str("RemoveRegistry() - sent quit to registry grpc server")
		} else {
			*done <- sase.Nabla(LOGEVENT).Str("RemoveRegistry() - node not running registry : nil quit channel")
		}
	} else {
		*done <- sase.Nabla(LOGEVENT).Str("RemoveRegistry() error - no launchregistry task result ")
	}
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeLaunchRegistryTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("launchregistry")
	task = task.AddForwardFn(LaunchRegistry, 500*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(RemoveRegistry, 1000*time.Millisecond, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddDepends([]string{"flock", "watchregistry", "reeveapi"})
	task = task.AddReverseDepends([]string{"launchsteward"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb",
		"stable", "watchleader", "watchregistry", "watchsteward",
		"reeveapi", "launchreeve"})
	return task
}

// WORKFLOW TASK - "stewarddb" -- starts up / shuts down the steward database and injestor routines

// StartStewardDB - if we are the leader, we start the sqlite db in pkg/registrydb and ingest/fanout event loops.
func StartStewardDB(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StartStewardDB()")

	// Get the flocking informaiton
	confab := params["flock"].Dface.(crux.Confab)
	cn := confab.GetNames()
	leader, me := cn.Leader, cn.Node
	// stewaddress, _ := confab.GetSteward()
	if leader != me { // No-op, all good
		*done <- sase.Nabla(LOGEVENT).Str("StartStewardDB() not the leader")
		*done <- sase.Got()
		return
	}
	//
	dbpath := muck.StewardDir() + "/steward.db"

	// Start up the Steward Ingestor, Initialize db & tables, load rules, open database, start fanout
	// keep its previous contents (inverse task renames db file when node is no longer leader)
	// DB has its own logger
	dblog := clog.Log.With("focus", steward.StewardRev, "mode", "steward-DB")
	// TODO consider passing in the flocking clock tick from some sensor elsewhere...maybe put it in flocking udp
	derr := steward.StartStewardDB(dbpath, dblog, false) // clear = false - reopen existing database if we are leader.
	if derr != nil {
		msg := fmt.Sprintf("StartStewardDB : steward.StartStewardDB() failed - %v", derr)
		pidstr, ts := grpcsig.GetPidTS()
		dblog.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		*done <- sase.Fail().Err(fmt.Errorf(msg))
	}
	// returns dbpath (so we know we started this db as leader))
	*done <- sase.Nabla(LOGEVENT).Str("StartStewardDB() steward database started")
	*done <- sase.Got().Str(dbpath)
}

// StopStewardDB - stops event loops for fanout, and injestor, then stops database.
func StopStewardDB(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("StopStewardDB()")
	// Get the flocking informaiton
	f, ok := params["flock"]
	if !ok {
		// can't do anything.
		*done <- sase.Inv().Err(fmt.Errorf("StopStewardDB() - no \"flock\" result found"))
	}
	confab := f.Dface.(crux.Confab)
	cn := confab.GetNames()
	leader, me := cn.Leader, cn.Node
	r, ok := params["result"]
	// Fish out the dbpath string
	if ok {
		dbpath := r.Dstring
		if len(dbpath) > 0 { // there is a steward db started on this node, stop it
			steward.StopStewardDB(false)
			if leader != me { // I am no longer leader - archive the stopped db
				*done <- sase.Nabla(LOGEVENT).Str("StopStewardDB() - FAKED archive stopped db ")
				// TODO archive the db file
			}
			*done <- sase.Nabla(LOGEVENT).Str("StopStewardDB() - steward db stopped ")
			*done <- sase.Inv()
			return
		}
	}
	*done <- sase.Nabla(LOGEVENT).Str("StopStewardDB - no steward db on node")
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeStewardDBTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("stewarddb")
	task = task.AddForwardFn(StartStewardDB, 12*time.Second, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(StopStewardDB, 22*time.Second, auto.ConfPROD, auto.TransitDISK, 0) // 22 = 10 + 12 (for startup time)
	// should start concurrently with "launchreeve" - needs nothing from reeve, just to know it is on leader
	task = task.AddDepends([]string{"flock", "watchsteward"})
	task = task.AddReverseDepends([]string{"launchsteward"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward"})
	return task
}

/*
---
	It would be good to really know that the steward grpc service is flushed cleanly.
	We'd have to wait for the steward grpc stopper to return something...

	Ingestor.Quit() stops clock, stops event loop - anything buffered since last clock tick -
        waiting in linked list - is never processed - never spewed.
	However process(dlhead, c.clock) is a goroutine. which calls steward.DBList - pushes db info,
	activates Fanout by pushing the clock tick to its event loop.

	It would be good to know that the Spewer is not in the middle of spewing. Event loop
	stops. There are no goroutines in fanout, so it has to have finished.

	steward.StopStewardDB(debug)
	// check confab to see if we are still leader -
	// if we are not leader and there is a dbpath - stop with debug. after DB.Close(), rename DB file (so it is not reloaded)
	// if we are leader and there is a dbpath - stop without debug. Do not rename DB file (we may recover)

*/

// WORKFLOW TASK - "launchsteward"  -- starts up /shuts down the steward gprc service

// LaunchSteward - steward server launch
func LaunchSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("LaunchSteward()")

	// See if we are leader
	confab := params["flock"].Dface.(crux.Confab)
	cn := confab.GetNames()
	leader, me := cn.Leader, cn.Node
	if leader != me { // No-op, all good
		*done <- sase.Nabla(LOGEVENT).Str("LaunchSteward() not the leader")
		*done <- sase.Got()
		return
	}

	// Need values for nodeid
	fps := FlockParamT{}
	_ = json.Unmarshal(params["flock"].Djson, &fps)
	var reeveapi rucklib.ReeveAPI

	// Need reeveapi
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("LaunchSteward() - no reeveapi task result"))
		return
	}

	// make a nodeid for steward
	stewnod, werr := idutils.NewNodeID(fps.Block, fps.Horde, me, steward.StewardName, steward.StewardAPI)
	if werr != nil {
		msg1 := fmt.Sprintf("LaunchSteward() - invalid nodeid params for steward: %v", werr)
		*done <- sase.Fail().Err(fmt.Errorf(msg1))
		return
	}

	// make a netid for steward
	principal, _ := muck.Principal()
	stewnid, eerr := idutils.NewNetID(steward.StewardRev, principal, me, stewPort)
	if eerr != nil {
		msg2 := fmt.Sprintf("LaunchSteward() error - invalid netid params for steward: %v", eerr)
		*done <- sase.Fail().Err(fmt.Errorf(msg2))
	}

	// need a SecureService interface{} for steward grpcsig authentication
	stewimp := reeveapi.SecureService(steward.StewardRev)
	if stewimp == nil {
		msg3 := "LaunchSteward() error - failed reeveapi.SecureService for steward"
		*done <- sase.Fail().Err(fmt.Errorf(msg3))
	}

	//  need a stop channel to stop the worker
	// plumbed into the worker.
	stop := make(chan bool)

	// steward.Launch starts the service and provides a  goroutine to do a gentle shutdown
	rerr := steward.Launch(stewnod, stewnid, stewimp, &stop)
	if rerr != nil {
		// Throw the FAIL event with error
		msg4 := fmt.Sprintf("LaunchSteward() steward.Launch() failed for %s/%s : %v", stewnod.String(), stewnid.String(), rerr)
		*done <- sase.Fail().Err(fmt.Errorf(msg4))
		return
	}
	// ALL GOOD. We return with GOT event, and the quit channel
	*done <- sase.Nabla(LOGEVENT).Str("LaunchSteward() steward server launched")
	// keep the nid and the nod for registration later
	*done <- sase.Got().QChan(&stop).ID(stewnid.String()).Str(stewnod.String())
}

// RemoveSteward - removes steward with graceful shutdown
func RemoveSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RemoveSteward()")
	r, ok := params["result"]
	// Fish out the quit channel from the results of LaunchSteward
	if ok {
		quit := r.Dchan
		if quit != nil { // As it will be if steward is not running on node
			*quit <- true
			*done <- sase.Nabla(LOGEVENT).Str("RemoveSteward() - sent quit to steward grpc server")
		} else {
			*done <- sase.Nabla(LOGEVENT).Str("RemoveSteward() - node not running steward : nil quit channel")
		}
	} else {
		*done <- sase.Nabla(LOGEVENT).Str("RemoveSteward() - error : no launchsteward task result ")
	}
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeLaunchStewardTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("launchsteward")
	task = task.AddForwardFn(LaunchSteward, 60*time.Millisecond, auto.ConfPROD, auto.TransitRPC, 0)
	task = task.AddInverseFn(RemoveSteward, 60*time.Millisecond, auto.ConfPROD, auto.TransitRPC, 0)
	task = task.AddDepends([]string{"flock", "reeveapi", "launchregistry", "stewarddb"})
	task = task.AddReverseDepends([]string{"register"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "launchregistry", "stewarddb"})
	return task
}

// WORKFLOW TASK - "register" -- register this reeve with global registry

// RegisterNode - does the registry handshake
func RegisterNode(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RegisterNode()")
	var reeveapi rucklib.ReeveAPI
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterNode() - no reeveapi task result"))
		return
	}
	reevenod, reevenid, _, reevepkjson, reeveimp := reeveapi.ReeveCallBackInfo()
	// need regaddress, regkey
	confab := params["flock"].Dface.(crux.Confab)
	cn := confab.GetNames()
	regaddress, regkey := cn.RegistryAddr, cn.RegistryKey

	// Register this node's Reeve Service on the Registry Server
	// allowing this Reeve to communicate with the centralized Steward Service
	// This works also on the node hosting the Registry & Steward Servers themselfves

	// Make a client to talk to the Registry Server, holding our reeve
	// callback information
	var reg crux.RegisterClient
	registercli := newRegisterClient(reevenod, reevenid, reeveimp)
	if registercli == nil {
		msg1 := "RegisterNode - newRegisterClient failed"
		*done <- sase.Fail().Err(fmt.Errorf(msg1))
		return
	}
	reg = registercli // handle the interface

	// Dial the Register server and
	// invoke the registration handshake to register our
	// reeve with the flock, executing
	// the two-way exchange of public keys
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("RegisterNode() registered %s %s on %s", reevenod, reevenid, regaddress))
	gerr := reg.AddAReeve(regaddress, regkey, reevepkjson)
	if gerr != nil {
		// Fatal if we cannot register
		msg2 := fmt.Sprintf("RegisterNode() - error - could not AddAReeve() %v", gerr)
		*done <- sase.Fail().Err(fmt.Errorf(msg2))
		return
	}
	// close registercli ?

	*done <- sase.Nabla(LOGEVENT).Str("RegisterNode() completed")
	*done <- sase.Got().Str("registered")
}

// UnRegisterNode - removes steward with graceful shutdown
func UnRegisterNode(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("UnRegisterNode()")

	// Registration is dead. Must delete any keys that were recieved on this node during exchange.
	// Remove keys from whitelist that were exchanged during registration handshake

	// keys starting with ReeveRev/StewardRev arise from the steward/registry operations
	// each node should only have one of the ReevRev keys which comes from steward/registry
	// becausese reeve only talks to steward (other than the self-signer)
	err1 := grpcsig.RemoveServiceRevPubKeysFromDB(reeve.ReeveRev)
	if err1 != nil {
		msg1 := fmt.Sprintf("UnRegisterNode() - error removing ReeveRev keys from whitelist %v", err1)
		*done <- sase.Fail().Err(fmt.Errorf(msg1))
	}

	// Leader has some work to do it will have 2 keys from each node
	// keys starting with StewardRev and ReeveRev from each the registered nodes
	// StewardRev key is used for reeve-steward grpc calls from each node
	// ReeveRev key was used in the reverse-signing of the node during the handshake phase
	// so the registry/steward node will have one of each these - for each node
	err2 := grpcsig.RemoveServiceRevPubKeysFromDB(steward.StewardRev)
	if err2 != nil {
		msg2 := fmt.Sprintf("UnRegisterNode() - error removing StewardRev keys from whitelist %v", err2)
		*done <- sase.Fail().Err(fmt.Errorf(msg2))
	}
	// Once these keys are removed, nodes trying to interact with Steward() will be blocked/unauthorized.
	// Steward calls out to Reeve will also be blocked/unauthorized.
	*done <- sase.Nabla(LOGEVENT).Str("UnRegisterNode() - all reeve and steward public keys removed from whitelist")
	*done <- sase.Inv()
}

func makeRegisterTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("register")
	task = task.AddForwardFn(RegisterNode, 100*time.Second, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddInverseFn(UnRegisterNode, 10*time.Second, auto.ConfPROD, auto.TransitDISK, 0)
	task = task.AddDepends([]string{"flock", "reeveapi", "launchreeve", "launchsteward", "launchregistry"})
	task = task.AddReverseDepends([]string{"syncsteward"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "registrydb", "launchregistry",
		"stewarddb", "launchsteward"})
	return task
}

/*
	// Establish that reeeve - to - steward gRPC communication works.
	// this
	// clientsLocalIni()
	// endpointsLocalIni()
	/  wakeUpSteward() - PingSleep blocks until Steward node appears,
	// startIngest() - starts event loop



	reeveapi.StopStewardIO() // shuts off the reeve event loop that sends client/update grpc calls to steward.
	no - enpointsLocalFini() This is always checkpointed after a change. Do we wipe it from mem? Do we move Completed to Pending?
	no - clientsLocalFini() This is alwasy checkpointed after a change. Do we wipe it from mem? Do we move Completed to Pending?

	No routines to move ClientPending to ClientCompleted.
	So for now ClientCompleted is empty.

	Need to put something in steward to call back this reeve with "ClientCompleted" signal after fanout.
	When it has some idle time.
	Something in reeve server to respond to this steward call and move Pending->Completed

	Need to put something in reeve to trigger
	- move all completed to pending
	- resend all pending to steward
	-

	OR something like Andrew's Sync that pings everything, when this succeeds, mark completed.

	If I just do this now, it will resend what is in pending. Completed will never populate.

	On bring up, just resend everything? Never worry about "completed" state?

	What good is it that we save the Clock State in whitelist.db ?

*/

// WORKFLOW TASK - "stewardio" -- connect reeve to steward, pingtest until up

// SyncSteward - wait until reeve - to - steward gRPC communication works, start reeve Ingest event
// loop, also load in previously registered Clients/Endpoints lists
func SyncSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("SyncSteward()")
	var reeveapi rucklib.ReeveAPI
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("SyncSteward() - no reeveapi task result"))
		return
	}

	// How long pingsleep while waiting for steward to come up?
	stewtimeout := 50 * time.Second
	stewerr := reeveapi.StartStewardIO(stewtimeout)
	if stewerr != nil {
		// Fatal if we cannot talk to steward
		msg := fmt.Sprintf("SyncSteward() reeve.StartStewardIO() failed after %v : %v", stewtimeout, stewerr)
		*done <- sase.Fail().Err(fmt.Errorf(msg))
		return
	}
	// Henceforth Steward is up, and Reeve can talk to Steward.
	*done <- sase.Nabla(LOGEVENT).Str("SyncSteward() complete - this reeve can communicate with steward")
	*done <- sase.Got().Str("synced")
}

// UnSyncSteward - stops reeve Ingest event loop - this reeve will not send anything to steward.
func UnSyncSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("UnSyncSteward()")
	var reeveapi rucklib.ReeveAPI
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("UnSyncSteward() - no reeveapi task result"))
		return
	}
	reeveapi.StopStewardIO()
	*done <- sase.Inv()
}

func makeSyncStewardTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("syncsteward")
	task = task.AddForwardFn(SyncSteward, 60*time.Second, auto.ConfPROD, auto.TransitLAN, 5) // 5 retries
	task = task.AddInverseFn(UnSyncSteward, 60*time.Second, auto.ConfPROD, auto.TransitRPC, 0)
	task = task.AddDepends([]string{"reeveapi", "register"})
	task = task.AddReverseDepends([]string{"registerreeve"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "registrydb", "launchregistry",
		"stewarddb", "launchsteward", "register"})
	return task
}

// WORKFLOW TASK - "registerreeve" - endpont and client

// RegisterReeve - register reeve endpoint and client with itself
func RegisterReeve(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RegisterReeve()")

	// get reeveapi
	var reeveapi rucklib.ReeveAPI
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - no reeveapi task result"))
		return
	}
	reevenodstr, reevenidstr, reevekeyid, reevepubkeyjson, reeveimp := reeveapi.ReeveCallBackInfo()

	if reeveimp == nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - no reeve grpcsig implementation"))
		return
	}

	reevenid, ierr := idutils.NetIDParse(reevenidstr)
	if ierr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - failed to parse reeve netid : %v", ierr))
		return
	}
	reevenod, merr := idutils.NodeIDParse(reevenodstr)
	if merr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - failed to parse reeve nodeid : %v", merr))
		return
	}

	// We will now register Reeve with Steward via itself formally.
	// Get the self-signer

	selfsig := reeveapi.SelfSigner()
	// Dial the local gRPC client
	reglog := clog.Log.With("focus", "registerreeve")

	reeveclien, cerr := reeve.OpenGrpcReeveClient(reevenid, selfsig, reglog)
	if cerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - failed to open local grpc reeve client"))
		return
	}

	// Construct what we need to RegisterEndpoint
	reeveep := pb.EndpointInfo{
		Tscreated: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Tsmessage: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Status:    pb.ServiceState_UP,
		Nodeid:    reevenod.String(),
		Netid:     reevenid.String(),
		Filename:  reeve.ReeveRev, // any plugin file hash goes here
	}

	// Make the gRPC call to local reeve to register itself
	ackE, xerr := reeveclien.RegisterEndpoint(context.Background(), &reeveep)
	if xerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - failed to register endpoint : %v", xerr))
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("RegisterReeve() - reeve is endpoint registered with reeve: %v", ackE))

	// Now, Register reeve's steward client, formally

	reevecl := pb.ClientInfo{
		Nodeid:  reevenod.String(),
		Keyid:   reevekeyid,
		Keyjson: reevepubkeyjson,
		Status:  pb.KeyStatus_CURRENT,
	}

	// Make the gRPC call to local reeve to register itself
	ackC, yerr := reeveclien.RegisterClient(context.Background(), &reevecl)
	if yerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - failed to register client : %v", yerr))
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("RegisterReeve() - reeve is client registered with reeve: %v", ackC))
	reeve.CloseGrpcReeveClient()
	*done <- sase.Got()
}

// UnRegisterReeve - un-register reeve endpoint and client from itself
func UnRegisterReeve(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("UnRegisterReeve()")
	// Unregister - send deprecate msg to reeve (expect forwarding to steward to fail), move info back to pending.
	// anyhow, move to pending if completed.
	// This is where any pending entries should be re-submitted.??
	// SO when the leader fails in "ending" it will shut off its steward/register services
	// so we can un-register with local reeve, but reeve cannot forward anything to steward
	// as the reeve client/endpoint injestor will throw an error and be unable to connect.

	// We can only assume the steward database is gone.
	// The forward task will reinstall these registrations.
	//

	// in the reeve injestor - the clientUpdateSteward/endpointUpdateSteward
	// will hit an error - and the TODO is to push to completed or fail.
	// In this case we consider "completed" as accepted tx to Steward.
	// and "fail" as failed to tx to Steward.

	// So we don't retry the fail in the injestor.
	// Any retry should be at the tomaton RegisterReeve retry counts.

	*done <- sase.Inv() // for now just invalidate results.
}

func makeRegisterReeveTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("registerreeve")
	task = task.AddForwardFn(RegisterReeve, 5*time.Second, auto.ConfPROD, auto.TransitLAN, 0) // 0 retries ???
	task = task.AddInverseFn(UnRegisterReeve, 5*time.Second, auto.ConfPROD, auto.TransitLAN, 0)
	task = task.AddDepends([]string{"reeveapi", "syncsteward"})
	task = task.AddReverseDepends([]string{"registersteward"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "registrydb", "launchregistry",
		"stewarddb", "launchsteward", "register", "syncsteward"})
	return task
}

// WORKFLOW TASK - "registersteward" - endpoint and client if we are leader

// RegisterSteward - register steward endpoint and client with reeve
func RegisterSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RegisterSteward()")
	// See if we are leader from the "flock" task confab interface
	confab := params["flock"].Dface.(crux.Confab)
	cn := confab.GetNames()
	leader, me := cn.Leader, cn.Node
	if leader != me { // No-op, all good
		*done <- sase.Nabla(LOGEVENT).Str("RegisterSteward() - not the leader")
		*done <- sase.Got()
		return
	}
	// get reeveapi for self signer from "reeveapi" task
	var reeveapi rucklib.ReeveAPI
	r, ok := params["reeveapi"]
	if ok {
		if r.Dface != nil {
			reeveapi = r.Dface.(rucklib.ReeveAPI)
		}
	}
	if reeveapi == nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - no reeveapi task result"))
		return
	}
	_, reevenidstr, _, _, _ := reeveapi.ReeveCallBackInfo()

	reevenid, ierr := idutils.NetIDParse(reevenidstr)
	if ierr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterSteward() - failed to parse reeve netid : %v", ierr))
		return
	}

	// Get stewnod, stewnid as strings from "launchsteward" task.
	s, good := params["launchsteward"]
	var stewnod, stewnid string
	if good {
		stewnod = s.Dstring
		stewnid = s.Dnuid
	}
	if len(stewnod) == 0 || len(stewnid) == 0 {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterReeve() - no launchsteward nodeid, netid found"))
		return
	}

	// Get the self-signer
	selfsig := reeveapi.SelfSigner()
	// Dial the local gRPC client
	reglog := clog.Log.With("focus", "registersteward")
	reeveclien, cerr := reeve.OpenGrpcReeveClient(reevenid, selfsig, reglog)
	if cerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterSteward() - failed to open local grpc reeve client"))
		return
	}

	// Construct what we need to RegisterEndpoint
	stewep := pb.EndpointInfo{
		Tscreated: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Tsmessage: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Status:    pb.ServiceState_UP,
		Nodeid:    stewnod,
		Netid:     stewnid,
		Filename:  steward.StewardRev, // any plugin file hash goes here
	}

	// Make the gRPC call to local reeve to register steward
	ackE, xerr := reeveclien.RegisterEndpoint(context.Background(), &stewep)
	if xerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterSteward() - failed to register endpoint : %v", xerr))
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("RegisterSteward() - steward is endpoint registered with reeve: %v", ackE))

	// Now, Register steward's reeve client, formally - whose public keys are held in register for
	// bootstrapping purposes.
	stewcl := pb.ClientInfo{
		Nodeid:  stewnod,
		Keyid:   register.GetStewardKeyID(),
		Keyjson: register.GetStewardPubkeyJSON(),
		Status:  pb.KeyStatus_CURRENT,
	}

	// Make the gRPC call to local reeve to register steward
	ackC, yerr := reeveclien.RegisterClient(context.Background(), &stewcl)
	if yerr != nil {
		*done <- sase.Fail().Err(fmt.Errorf("RegisterSteward() - failed to register steward client : %v", yerr))
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str(fmt.Sprintf("RegisterSteward() - steward is client registered with reeve: %v", ackC))
	reeve.CloseGrpcReeveClient()
	*done <- sase.Got()
}

// UnRegisterSteward - un-register steward endpoint and client from reeve
func UnRegisterSteward(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("UnRegisterSteward()")
	// Unregister - send deprecate msg to reeve (expect forwarding to steward to fail), move info back to pending.
	// anyhow, move to pending if completed.
	// This is where any pending entries should be re-submitted.??
	*done <- sase.Inv() // for now just invalidate results.
}

func makeRegisterStewardTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("registersteward")
	task = task.AddForwardFn(RegisterSteward, 5*time.Second, auto.ConfPROD, auto.TransitLAN, 0) // 0 retries ???
	task = task.AddInverseFn(UnRegisterSteward, 5*time.Second, auto.ConfPROD, auto.TransitLAN, 0)
	task = task.AddDepends([]string{"flock", "reeveapi", "launchsteward", "registerreeve"})
	task = task.AddReverseDepends([]string{"ending"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "registrydb", "launchregistry",
		"stewarddb", "launchsteward", "register", "syncsteward", "registerreeve"})
	return task
}

// TODO Tasks to add:

// PERHAPS I should not run this stuff in the same workflow.

// Rather - detect this workflow has w.FowardGoal = true in ruck/organza.go
// then start another workflow to launch this stuff.
// in 2nd layer.

// WORKFLOW TASK - "launchbar"
// WORKFLOW TASK - "registerbar"
// WORKFLOW TASK - "registerfoo"
// WORKFLOW TASK - "testfoobar"
// WORKFLOW TASK - "pastiche stuff"

// WORKFLOW TASK - "ending" - ending task for forcing leader to reverse

// Ending - ending task that triggers a complete workflow undo after a delay on the leader
func Ending(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("Ending()")
	f, ok := params["flock"]
	if !ok {
		// can't do anything.
		*done <- sase.Fail().Err(fmt.Errorf("Ending() - no flock result found"))
	}
	confab := f.Dface.(crux.Confab)
	cn := confab.GetNames()
	leader, me := cn.Leader, cn.Node
	if leader == me {
		*done <- sase.Nabla(LOGEVENT).Str("Ending() - as leader - terminating in 20s.")
		time.Sleep(20 * time.Second) // Give everyone time to catch up.
		*done <- sase.Got()
		// Invalidate the start task
		*done <- sase.Fail().Err(fmt.Errorf("Intentional Ending")) // Unwind all steps because
		return
	}
	*done <- sase.Nabla(LOGEVENT).Str("Ending() - not the leader - workflow complete, holding until triggered")
	*done <- sase.Got() // This triggers the end of the workflow.
	// Since we have w.Hold = true, worker stays around, and awake.
}

// RemoveEnding - removes the ending
func RemoveEnding(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RemoveEnding()")
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeEndingTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("ending")
	task = task.AddForwardFn(Ending, 30*time.Second, auto.ConfKNOWN, auto.TransitRAM, 0)
	task = task.AddInverseFn(RemoveEnding, 50*time.Millisecond, auto.ConfKNOWN, auto.TransitRAM, 0)
	task = task.AddDepends([]string{"flock", "registersteward"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "launchregistry", "stewarddb", "launchsteward",
		"register", "syncsteward", "registerreeve", "registersteward"})
	return task
}

// WORKFLOW TASK - "ending" - nominal goal - holding task for normal workflow

// Holding - ending task that triggers a complete workflow undo after a delay on the leader
func Holding(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("Holding() - workflow complete")
	*done <- sase.Got() // This triggers the end of the workflow.
	// Since we have w.Hold = true, worker stays around, and awake.
}

// Releasing - undo for the ending
func Releasing(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {
	*done <- sase.Nabla(LOGEVENT).Str("RemoveEnding()")
	// Invalidate any saved worker output
	*done <- sase.Inv()
}

func makeHoldingTask() auto.TaskT {
	task := auto.TaskT{}
	task = task.AddName("ending") // Use the same key as the test ending
	task = task.AddForwardFn(Holding, 50*time.Millisecond, auto.ConfKNOWN, auto.TransitRAM, 0)
	task = task.AddInverseFn(Releasing, 50*time.Millisecond, auto.ConfKNOWN, auto.TransitRAM, 0)
	task = task.AddDepends([]string{"registersteward"})
	task = task.AddAllDepends([]string{"flock", "muck", "selfkeys", "whitelistdb", "stable",
		"watchleader", "watchregistry", "watchsteward", "reeveapi",
		"launchreeve", "launchregistry", "stewarddb", "launchsteward",
		"register", "syncsteward", "registerreeve", "registersteward"})
	return task
}

// ASSEMBLE THE BOOTER WORKER - ORGANZA "Battle Royale" test

// Function to gather up all the MakeXxxTask functions
// into an array of MakeTaskFuncT.
// Order of appearance is not important,
// as dependencies indicate Task runtime order.
// Marked as ** where it needs to be customized for other workflows
func testbooterTasks() []auto.MakeTaskFuncT {

	taskfns := []auto.MakeTaskFuncT{}
	// ** Append all the individual MakeXxxTasks
	taskfns = append(taskfns, makeFlockTask)
	taskfns = append(taskfns, makeMuckTask)
	taskfns = append(taskfns, makeSelfKeysTask)
	taskfns = append(taskfns, makeWhitelistDBTask)
	taskfns = append(taskfns, makeStableTask)
	taskfns = append(taskfns, makeWatchLeaderTask)
	taskfns = append(taskfns, makeWatchRegTask)
	taskfns = append(taskfns, makeWatchStewTask)
	taskfns = append(taskfns, makeReeveAPITask)
	taskfns = append(taskfns, makeLaunchReeveTask)
	taskfns = append(taskfns, makeLaunchRegistryTask)
	taskfns = append(taskfns, makeStewardDBTask)
	taskfns = append(taskfns, makeLaunchStewardTask)
	taskfns = append(taskfns, makeRegisterTask)
	taskfns = append(taskfns, makeSyncStewardTask)
	taskfns = append(taskfns, makeRegisterReeveTask)
	taskfns = append(taskfns, makeRegisterStewardTask)
	taskfns = append(taskfns, makeEndingTask)
	// **
	return taskfns
}

// Function to gather up all the MakeXxxTask functions
// into an array of MakeTaskFuncT.
// Order of appearance is not important,
// as dependencies indicate Task runtime order.
// Marked as ** where it needs to be customized for other workflows
func booterTasks() []auto.MakeTaskFuncT {

	taskfns := []auto.MakeTaskFuncT{}
	// ** Append all the individual MakeXxxTasks
	taskfns = append(taskfns, makeFlockTask)
	taskfns = append(taskfns, makeMuckTask)
	taskfns = append(taskfns, makeSelfKeysTask)
	taskfns = append(taskfns, makeWhitelistDBTask)
	taskfns = append(taskfns, makeStableTask)
	taskfns = append(taskfns, makeWatchLeaderTask)
	taskfns = append(taskfns, makeWatchRegTask)
	taskfns = append(taskfns, makeWatchStewTask)
	taskfns = append(taskfns, makeReeveAPITask)
	taskfns = append(taskfns, makeLaunchReeveTask)
	taskfns = append(taskfns, makeLaunchRegistryTask)
	taskfns = append(taskfns, makeStewardDBTask)
	taskfns = append(taskfns, makeLaunchStewardTask)
	taskfns = append(taskfns, makeRegisterTask)
	taskfns = append(taskfns, makeSyncStewardTask)
	taskfns = append(taskfns, makeRegisterReeveTask)
	taskfns = append(taskfns, makeRegisterStewardTask)
	taskfns = append(taskfns, makeHoldingTask)
	// **
	return taskfns
}

// NewTestBooterWorker - assembles all the above code and function pointers into a booter worker
// Return a New Booter Worker that does the "Battle Royale" stye leader fail after delay
// Marked as ** where it needs to be customized for other workflows
func NewTestBooterWorker(name string, parent string) auto.WorkerT {

	// ** Get your list of MakeTaskFuncT
	taskfns := testbooterTasks()
	// **
	tasks := []auto.TaskT{}

	// Make the list of all your TaskT structs
	for _, tf := range taskfns {
		task := tf()
		tasks = append(tasks, task)
	}

	// ** Make the worker stating your Goal and Start Tasks
	worker := auto.NewWorker(name,
		"ending",                  // The GOAL task
		[]string{"flock", "muck"}, // Start Tasks
		tasks...)
	// **

	// ** Worker fine-tuning
	worker.UndoOnFail = true // Enable Undo workflow when a task returns a "Fail" event
	worker.Hold = true       // Keep worker in memory even after goal is reached.
	// Must be set true in this case, otherwise Cancelled tasks don't get to Done.
	// **
	return worker
}

// NewBooterWorker - assembles all the above code and function pointers into a booter worker
// Return a New Booter Worker
// Marked as ** where it needs to be customized for other workflows
func NewBooterWorker(name string, parent string) auto.WorkerT {

	// ** Get your list of MakeTaskFuncT
	taskfns := booterTasks()
	// **
	tasks := []auto.TaskT{}

	// Make the list of all your TaskT structs
	for _, tf := range taskfns {
		task := tf()
		tasks = append(tasks, task)
	}

	// ** Make the worker stating your Goal and Start Tasks
	worker := auto.NewWorker(name,
		"ending",                  // The GOAL task
		[]string{"flock", "muck"}, // Start Tasks
		tasks...)
	// **

	// ** Worker fine-tuning
	worker.UndoOnFail = true // Enable Undo workflow when a task returns a "Fail" event
	worker.Hold = true       // Keep worker in memory even after goal is reached.
	// Must be set true in this case, otherwise Cancelled tasks don't get to Done.
	// **
	return worker
}

// Function to load a Booter worker and re-attach its functions.
// - Not Implemented
// func LoadBooterWorker(filename string) (auto.WorkerT, error) {
//	return w, nil
//}
