// (c) Ericsson Inc. 2016 All Rights Reserved
// Contributors:
//      Christopher W. V. Hogue

package auto

import (
	"encoding/json"
	"sync"
	"time"
)

// Event Constants

// WAKE - event message to wake up a worker
const WAKE string = "Wake"

// DONE - event message
const DONE string = "Done"

// ADDCHILD - event message adding a child worker
const ADDCHILD string = "AddChild"

// GOTCHILD - event message return from a child worker
const GOTCHILD string = "GotChild"

// GOT - event message, function completed
const GOT string = "Got"

// INV - event message, inverse function completed
const INV string = "Inv"

// TIMEOUT - event message, function timed out
const TIMEOUT string = "Timeout"

// CANCEL - event message, function is cancelled
const CANCEL string = "Cancel"

// FAIL - event message - function failed with error
const FAIL string = "Fail"

// RMV - event message - revoke a function's result and trigger workflow reversal
const RMV string = "Rmv"

// STATUS - event message - status
const STATUS string = "Status"

// ADD - event message - add
const ADD string = "Add"

// DEL - event message - del
const DEL string = "Del"

// PartialFraction - Fraction of child workers
const PartialFraction float64 = 0.18

// JSONDataMap - Flag for a DataMap in json - for child worker handling
const JSONDataMap int = 1

// EventT - the event struct
type EventT struct {
	Broadcast bool `json:"broadcast"`
	// should this be sent/scanned by all workers?

	ToHub bool `json:"tohub"`
	// Higher-ordered Event (status, progress) sent to upstream Hub/Nabla,
	// has own WorkerT in address, but is ignored by Worker.

	Timeless bool `json:"timeless"`
	// This event does not expire, or has no reference start time from
	// which to calculate an expiry duration

	Workerid string `json:"workerid"`
	// worker that this event belongs to, if known

	Senderid string `json:"senderid"`
	// worker sending this event, if not the same as Workerid

	Taskkey string `json:"taskkey"`
	// task this event belongs to

	Expires time.Time `json:"ts"`
	// timestamp after which this event is to be disregarded
	// provided to the actionFuncT for its events.

	NCode int `json:"ncode"`
	// Code for Nabla function routing

	Msg string `json:"msg"`
	// class of message

	Data DataT `json:"data"`
	// return values or errors
}

// Some EventT methods - for one-liner event setup
// e.g.
// event = event.To(workerid).Task("customer").Expires(expires).Inv()

// Got - changes the event message to GOT
func (e EventT) Got() EventT {
	e.Msg = GOT
	return e
}

// Wake - changes the event message to WAKE
func (e EventT) Wake() EventT {
	e.Msg = WAKE
	return e
}

// Fail - changes the event message to FAIL
func (e EventT) Fail() EventT {
	e.Msg = FAIL
	return e
}

// Cancel - changes the event message to CANCEL
func (e EventT) Cancel() EventT {
	e.Msg = CANCEL
	return e
}

// Rmv - changes the event message to RMV
// Also wakes up a worker
func (e EventT) Rmv() EventT {
	e.Msg = RMV
	return e
}

// Inv - changes the event message to INV
func (e EventT) Inv() EventT {
	e.Msg = INV
	return e
}

// Nabla - changes the event message to route to the nabla with the provided code
func (e EventT) Nabla(code int) EventT {
	e.ToHub = true
	e.NCode = code
	return e
}

// Status - changes the event message to STATUS
func (e EventT) Status() EventT {
	e.Msg = STATUS
	return e
}

// To - changes the address of the message to the provided worker ID
func (e EventT) To(w string) EventT {
	e.Workerid = w
	return e
}

// ToAll - changes the address of the message to broadcast to all workers
func (e EventT) ToAll() EventT {
	e.Broadcast = true
	e.Workerid = "0000000000000000000000"
	return e
}

// From - changes the from address of the event to the provided worker ID
func (e EventT) From(w string) EventT {
	e.Senderid = w
	return e
}

// Task - changes the task of the address to the provided task string
func (e EventT) Task(t string) EventT {
	e.Taskkey = t
	return e
}

// Expires and Timeless are mutually exclusive,
// so last one invoked wins.

// Expiry - changes the expiry time on the event to the provided time duration
func (e EventT) Expiry(t time.Time) EventT {
	e.Expires = t
	e.Timeless = false
	return e
}

// NoExpiry - changes the expiry time on the event to no expiry time
func (e EventT) NoExpiry() EventT {
	e.Expires = time.Time{}
	e.Timeless = true
	return e
}

// Now - changes the expiry time on the event to Now plus some duration
func (e EventT) Now(to time.Duration) EventT {
	e.Expires = time.Now().UTC().Add(to)
	return e
}

// ID - adds a string to the Dnuid field of the event's data
func (e EventT) ID(s string) EventT {
	e.Data.Dnuid = s
	return e
}

// Str - adds a string to the Dstring field of the event's data
func (e EventT) Str(s string) EventT {
	e.Data.Dstring = s
	return e
}

// Int - adds an int to the Dint field of the event's data
func (e EventT) Int(i int) EventT {
	e.Data.Dint = i
	return e
}

// Bool - adds a bool to the Dbool field of the event's data
func (e EventT) Bool(b bool) EventT {
	e.Data.Dbool = b
	return e
}

// Iface - adds an interface to the Dface field of the event's data
func (e EventT) Iface(i interface{}) EventT {
	e.Data.Dface = i
	return e
}

// QChan - adds a pointer to a r/w boolean quit channel to the Dchan field of the event's data
func (e EventT) QChan(c *chan bool) EventT {
	e.Data.Dchan = c
	return e
}

// Jsn - adds a marshalled json byte arry to the Djson field of the event's data with encd
// as a code for type checking the downstream unmarshalling
func (e EventT) Jsn(j json.RawMessage, enc int) EventT {
	e.Data.Djson = j
	e.Data.Denc = enc
	return e
}

// Err - adds an error to the Derr field of the event's data
func (e EventT) Err(err error) EventT {
	e.Data.Derr = err
	return e
}

// DataT - the typed data that an event can carry.
type DataT struct {
	// holds primitive values in DataMapT
	Dbool   bool            `json:"dbool,omitempty"`   // Some user-defined boolean
	Dint    int             `json:"dint,omitempty"`    // Some user-defined integer
	Dnuid   string          `json:"dnuid,omitempty"`   // nuid to replace uuid
	Dstring string          `json:"dstring,omitempty"` // Some user defined string
	Denc    int             `json:"denc,omitempty"`    // User-defined index for struct of Djson contents
	Djson   json.RawMessage `json:"djson,omitempty"`   // json []bytes, unmarshalled based on Denc
	Derr    error           `json:"derr,omitempty"`    // A Go error
	Dface   interface{}     `json:"-"`                 // An interface (for anything) not serizlizable
	Dchan   *chan bool      `json:"-"`                 // A r/w boolean channel (for quitting) not serializable
}

// DataMapT - a map of strings (arg names if you like) to DataT
// DataMapT carries all the parameters needed by the funciton
type DataMapT map[string]DataT

// assigns arg string to value DataT
type actionFuncT func(*chan EventT, DataMapT, EventT, *WorkerT)

// *chan EventT is the worker's event channel as *done
// the sase EventT is the "self addressed, stamped envelope" for
// stuffing with return information

// TasktoDataMapT - A map of string task names to their DataMapT args & values
type TasktoDataMapT map[string]DataMapT

// FnSetT - function calls and timeout/retry/failure parameters
type FnSetT struct { // Function and failure parameters
	Fn       actionFuncT   `json:"-"`                  // the Function to call
	Ts       time.Time     `json:"ts,omitempty"`       // timestamp for first invocation
	TsRetry  time.Time     `json:"tsretry,omitempty"`  // timestamp from last time called
	To       time.Duration `json:"to,omitempty"`       // timeout for event response
	Transit  int           `json:"transit,omitempty"`  // see timeout.go - use const TRANSIT_
	Conf     int           `json:"conf,omitempty"`     // see timeout.g - use const CONF_
	Elapsed  time.Duration `json:"elapsed,omitempty"`  // Time from Ts when Event appeared
	Rt       int           `json:"rt,omitempty"`       // retry count
	Maxrt    int           `json:"maxrt,omitempty"`    // max retries
	Overtime bool          `json:"overtime,omitempty"` // past timeout
	Overtry  bool          `json:"overtry,omitempty"`  // past Maxrt
	Failure  bool          `json:"failure,omitempty"`  // Overtime || Overtry == true
	Errs     []string      `json:"errs,omitempty"`     // accumulated errors as strings
}

// TaskWorkerT - Child Workers
type TaskWorkerT struct {
	Results DataMapT `json:"results,omitempty"` // Map where string is nuid of child
	Partial bool     `json:"partial,omitempty"` //True when >18% of child results in
	Workers []string `json:"workers,omitempty"`
}

// TaskT - the task struct where Inputs, retults, state, are stored.
type TaskT struct { // Task inputs, outputs, functions, dag, control variables
	Key string `json:"key"`
	// name of task, used in map as key
	Inputs DataMapT `json:"inputs,omitempty"`
	// args, values of all input data
	Result *DataT `json:"result,omitempty"`
	// returned values when completed
	HasResult bool `json:"hasresult,omitempty"`
	// true when the task's goal is met
	Working bool `json:"working,omitempty"`
	// task is awaiting Event or Timeout
	Cancel bool `json:"cancel,omitempty"`
	// boolean for long running tasks to poll for cancellation
	Forward *FnSetT `json:"forward"`
	// Forward function to call with Inputs
	Fallback *FnSetT `json:"fallback,omitempty"`
	// Forward function fallback function
	Inverse *FnSetT `json:"inverse"`
	// Inverse (undo) function to call with
	// Inputs + Results
	Depends []string `json:"depends,omitempty"`
	// dag keys for input (apiFn) task call dependencies
	Revdeps []string `json:"revdeps,omitempty"`
	// dag keys for reverse (.inverse.fn) dependencies
	Alldeps []string `json:"alldeps,omitempty"`
	// dag keys for all dependent tasks back to w.starttasks
	// Child Workers
	Children  *TaskWorkerT `json:"children,omitempty"`
	Nchildren int          `json:"nchildren,omitempty"` // Number of child workers
}

// Task helper methods

// MakeTaskFuncT - a function that makes and returns a task
type MakeTaskFuncT func() TaskT

// AddForwardFn - adds a forward function pointer and a few settings
func (t TaskT) AddForwardFn(fn actionFuncT, timeout time.Duration, conf, transit, maxretries int) TaskT {
	fwd := new(FnSetT)
	fwd.Fn = fn
	fwd.Maxrt = maxretries
	fwd.To = timeout
	fwd.Conf = conf
	fwd.Transit = transit
	t.Forward = fwd
	return t
}

// AddFallbackFn - adds a fallback function pointer and a few settings
func (t TaskT) AddFallbackFn(fn actionFuncT, timeout time.Duration, conf, transit, maxretries int) TaskT {
	fwd := new(FnSetT)
	fwd.Fn = fn
	fwd.Maxrt = maxretries
	fwd.To = timeout
	fwd.Conf = conf
	fwd.Transit = transit
	t.Fallback = fwd
	return t
}

// AddInverseFn - adds an inverse (reverse - or undo) function pointer and a few settings
func (t TaskT) AddInverseFn(fn actionFuncT, timeout time.Duration, conf, transit, maxretries int) TaskT {
	fwd := new(FnSetT)
	fwd.Fn = fn
	fwd.Maxrt = maxretries
	fwd.To = timeout
	fwd.Conf = conf
	fwd.Transit = transit
	t.Inverse = fwd
	return t
}

// AddDepends - adds forward dependencies to a task. These strings are from earlier tasks,
// and their outputs will be provided as inputs to t
func (t TaskT) AddDepends(deps []string) TaskT {
	t.Depends = deps
	return t
}

// AddReverseDepends - adds reverse dependencies to a task. These strings are from later
// executed tasks.
func (t TaskT) AddReverseDepends(deps []string) TaskT {
	t.Revdeps = deps
	return t
}

// AddAllDepends - all the tasks that much happen before this task
func (t TaskT) AddAllDepends(deps []string) TaskT {
	t.Alldeps = deps
	return t
}

// AddName - names the task
func (t TaskT) AddName(name string) TaskT {
	t.Key = name
	return t
}

// Worker

// WorkerT - the worker struct
type WorkerT struct {
	Name       string      `json:"name"`
	Workerid   string      `json:"workerid,omitempty" pgt:"varchar(24)"`
	Parentid   string      `json:"parentid,omitempty" pgt:"varchar(24)"`
	Parenttask string      `json:"parenttask,omitempty" pgt:"varchar(40)"`
	Parentexp  time.Time   `json:"parentexp,omitempty" pgt:"timestamptz"`
	Events     chan EventT `json:"-"` // 'done' events come back here.
	Hub        *HubT       `json:"-"` // Hub for this worker group
	// Inbound Events with different NUIDs (e.g. parent) or Broadcast = true
	// are forwarded to the worker's Hub.family channel for routing
	// to other workers in the group
	Lasterror string `json:"lasterr,omitempty" pgt:"varchar(200)"`
	// use to pass the Worker global input data.
	//  TaskMap  map[string]int      `json:"taskmap" pgt:"jsonb"`
	//  I may reinstate this and change Tasks back to []TaskT to alleviate jankiness

	Tasks map[string]TaskT `json:"tasks" pgt:"jsonb"`
	// working tasks
	Forward bool `json:"forward" pgt:"bool"`
	Undo    bool `json:"undo,omitempty" pgt:"bool"`
	// switches reverse back to fwd
	Goaltask string `json:"goaltask" pgt:"varchar(30)"`
	// active goal - can change
	Goal     string `json:"goal" pgt:"varchar(30)"`
	AutoSave bool   `json:"autosave" pgt:"bool"`
	// forward goal - stays put
	Goalstate   bool `json:"goalstate" pgt:"bool"`
	ForwardGoal bool `json:"forwardgoal" pgt:"bool"`      // Worker hit forward goal.
	Awake       bool `json:"awake" pgt:"bool"`            // Worker is in a running state
	Hold        bool `json:"hold" pgt:"bool"`             // Keep worker in memory until explicitly terminated
	Dienow      bool `json:"dienow,omitempty" pgt:"bool"` // Worker termination signal
	UndoOnFail  bool `json:"undoonfail" pgt:"bool"`
	// if true, reverses workflow before worker stops
	// if false, worker just stops
	TSorigin   string        `json:"tsorigin" pgt:"timestamptz"`
	Started    time.Time     `json:"started" pgt:"timeestamp"`
	Elapsed    time.Duration `json:"elapsed,omitempty" pgt:"bigint"` // TODO pgt for time.Duration
	Starttasks []string      `json:"starttasks" pgt:"jsonb"`
	// Lock the Tasks map when operating on it
	Lock *sync.RWMutex `json:"-"`
}

// Goal when: w.Tasks[w.Task[w.Taskmap[w.goaltask]].hasResult == w.goalstate
