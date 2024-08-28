// (c) Ericsson Inc. 2015-2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// Provision workflow testing code for tomaton/meta funcitons
// Also provides an example "stub" for user workflow.

package pro

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nuid"

	"github.com/erixzone/crux/pkg/auto"
)

// WORKFLOW TASK - Customer
func upsertCustomer(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	sase = sase.Got().ID(nuid.Next())
	sase = sase.Str(params["name"].Dstring)
	sase = sase.Int(params["num"].Dint)
	*done <- sase
}

func deleteCustomer(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	*done <- sase.Inv()
}

func makeCustomerTask() auto.TaskT {

	timeout := gettimeout()
	task := auto.TaskT{}
	task = task.AddName("customer")
	task = task.AddForwardFn(upsertCustomer, timeout, 0, 0, 10)
	task = task.AddInverseFn(deleteCustomer, timeout, 0, 0, 10)
	task = task.AddReverseDepends([]string{"node", "image"})
	return task
}

// WORKFLOW TASK - Image

func getImage(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	sase = sase.Got().ID(nuid.Next())
	sase = sase.Str(fmt.Sprintf("Image For %s ", params["customer"].Dstring))
	*done <- sase
}

func deleteImage(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	*done <- sase.Inv()
}

func makeImageTask() auto.TaskT {

	timeout := gettimeout()
	task := auto.TaskT{}
	task = task.AddName("image")
	task = task.AddForwardFn(getImage, timeout, 0, 0, 10)
	task = task.AddInverseFn(deleteImage, timeout, 0, 0, 10)
	task = task.AddDepends([]string{"customer"})
	task = task.AddReverseDepends([]string{"provision"})
	task = task.AddAllDepends([]string{"customer"})
	return task
}

// WORKFLOW TASK - Size
func getSize(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	sase = sase.Got().ID(nuid.Next())
	sase = sase.Str(fmt.Sprintf("Size: %s ", params["size"].Dstring))
	*done <- sase
}

func deleteSize(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	*done <- sase.Inv()
}

func makeSizeTask() auto.TaskT {

	timeout := gettimeout()
	task := auto.TaskT{}
	task = task.AddName("size")
	task = task.AddForwardFn(getSize, timeout, 0, 0, 10)
	task = task.AddInverseFn(deleteSize, timeout, 0, 0, 10)
	task = task.AddReverseDepends([]string{"node"})
	return task
}

// WORKFLOW TASK - Node
func getNode(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	uid := nuid.Next()
	sase = sase.Got().ID(uid)
	sase = sase.Str(fmt.Sprintf("Node %s Obtained For %s %s ",
		uid,
		params["customer"].Dstring,
		params["size"].Dstring))

	*done <- sase
}

func deleteNode(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	*done <- sase.Inv()
}

func makeNodeTask() auto.TaskT {

	timeout := gettimeout()
	task := auto.TaskT{}
	task = task.AddName("node")
	task = task.AddForwardFn(getNode, timeout, 0, 0, 10)
	task = task.AddInverseFn(deleteNode, timeout, 0, 0, 10)
	task = task.AddDepends([]string{"customer", "size"})
	task = task.AddReverseDepends([]string{"provision"})
	task = task.AddAllDepends([]string{"customer", "size"})
	return task
}

// WORKFLOW TASK - Provision
func provision(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	sase = sase.Got().ID(nuid.Next())
	sase = sase.Str(fmt.Sprintf("Provisioning: %s ", params["node"].Dstring))
	*done <- sase
}

func undoProvision(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	*done <- sase.Inv()
}

func makeProvisionTask() auto.TaskT {

	timeout := gettimeout()
	slowtimeout := getslowtimeout()
	task := auto.TaskT{}
	task = task.AddName("provision")
	task = task.AddForwardFn(provision, timeout, 0, 0, 4)
	task = task.AddInverseFn(undoProvision, timeout, 0, 0, 4)
	task = task.AddFallbackFn(provision, slowtimeout, 0, 0, 4)
	task = task.AddDepends([]string{"node", "image"})
	task = task.AddReverseDepends([]string{"boot"})
	task = task.AddAllDepends([]string{"customer", "size", "node", "image"})
	return task
}

// WORKFLOW TASK - Boot

func boot(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	sase = sase.Got().ID(params["provision"].Dnuid)
	reply := fmt.Sprintf("Booted container %s -->  %s",
		params["provision"].Dnuid, params["provision"].Dstring)
	sase = sase.Str(reply)
	*done <- sase
}

func shutdown(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	time.Sleep(time.Second / TIMESLICE)
	reply := fmt.Sprintf("SHUT DOWN CONTAINER %s - %s",
		params["result"].Dnuid, params["provision"].Dstring)
	*done <- sase.Inv().Str(reply)
}

func makeBootTask() auto.TaskT {

	timeout := gettimeout()
	task := auto.TaskT{}
	task = task.AddName("boot")
	task = task.AddForwardFn(boot, timeout, 0, 0, 10)
	task = task.AddInverseFn(shutdown, timeout, 0, 0, 10)
	task = task.AddDepends([]string{"provision"})
	task = task.AddAllDepends([]string{"provision", "customer", "size", "node", "image"})
	return task
}

// ASSEMBLE THE PROVISION WORKER
// Function to gather up all the makeXxxTask functions
// into an array of makeTaskFuncT.
// Order of appearance is not important,
// as dependencies indicate Task runtime order.
// Marked as ** where it needs to be customized for other workflows
func provisionTasks() []auto.MakeTaskFuncT {

	taskfns := []auto.MakeTaskFuncT{}
	// ** Append all the individual makeXxxTasks
	taskfns = append(taskfns, makeBootTask)
	taskfns = append(taskfns, makeProvisionTask)
	taskfns = append(taskfns, makeNodeTask)
	taskfns = append(taskfns, makeImageTask)
	taskfns = append(taskfns, makeSizeTask)
	taskfns = append(taskfns, makeCustomerTask)
	// **
	return taskfns

}

// NewProvisionWorker - Return a New Provision Worker
// Marked as ** where it needs to be customized for other workflows
func NewProvisionWorker(name string, parent string) auto.WorkerT {

	// ** Get your list of makeTaskFuncT
	taskfns := provisionTasks()
	// **
	tasks := []auto.TaskT{}
	// make the list of all your TaskT structs
	for _, tf := range taskfns {
		task := tf()
		tasks = append(tasks, task)
	}
	// ** make the worker stating your Goal and Start Tasks
	worker := auto.NewWorker(name,
		"boot",                       // The GOAL task
		[]string{"customer", "size"}, // Start Tasks
		tasks...)
	// **
	// This can be nil when no parent
	worker.Parentid = parent
	// ** Worker fine-tuning
	worker.UndoOnFail = true // Enable Undo workflow when a task returns a "Fail" event
	// **
	return worker

}

// LoadProvisionWorker - Function to load a Provision worker and re-attach its functions.
// Kind of janky, a Candidate for the inline pattern...
func LoadProvisionWorker(filename string) (auto.WorkerT, error) {
	var w auto.WorkerT
	f, err := os.Open(filename)
	if err != nil {
		return w, err
	}
	defer f.Close()
	parseJSON := json.NewDecoder(f)
	err = parseJSON.Decode(&w)
	if err != nil {
		return w, err
	}
	// a lack of indirection here - inline pattern would help
	Boottask := w.Tasks["boot"]
	Boottask.Forward.Fn = boot
	Boottask.Inverse.Fn = shutdown
	w.Tasks["boot"] = Boottask
	Provtask := w.Tasks["provision"]
	Provtask.Forward.Fn = provision
	Provtask.Fallback.Fn = provision
	Provtask.Inverse.Fn = undoProvision
	w.Tasks["provision"] = Provtask
	Nodetask := w.Tasks["node"]
	Nodetask.Forward.Fn = getNode
	Nodetask.Inverse.Fn = deleteNode
	w.Tasks["node"] = Nodetask
	Imagetask := w.Tasks["image"]
	Imagetask.Forward.Fn = getImage
	Imagetask.Inverse.Fn = deleteImage
	w.Tasks["image"] = Imagetask
	Sizetask := w.Tasks["size"]
	Sizetask.Forward.Fn = getSize
	Sizetask.Inverse.Fn = deleteSize
	w.Tasks["size"] = Sizetask
	Customertask := w.Tasks["customer"]
	Customertask.Forward.Fn = upsertCustomer
	Customertask.Inverse.Fn = deleteCustomer
	w.Tasks["customer"] = Customertask
	w.Lock = &sync.RWMutex{}
	return w, nil
}

// This is a test workflow with timeouts instead of actual
// RPC or REST calls.
// Below are functions related to the test procedure

// TIMESLICE - divisor to vary timeouts
const TIMESLICE time.Duration = 10000 // Unitless

// Timeouts are used for changing relative times
// to trigger behavior, and are
// listed as string constants for my convenience,

// TIMEOUT - for normal speed
const TIMEOUT string = "0.01s"

// TIMEOUTSLOW - for slowing down timeout
const TIMEOUTSLOW string = "0.02s"

func gettimeout() time.Duration {
	timeout, err := time.ParseDuration(TIMEOUT)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Fatal - Time Parse Error: %v\n", err)
		os.Exit(1)
	}
	return timeout
}

func getslowtimeout() time.Duration {
	timeout, err := time.ParseDuration(TIMEOUTSLOW)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Fatal - Time Parse Error: %v\n", err)
		os.Exit(1)
	}
	return timeout
}

// ProvisionTooSlow - functions for testing Provision - swaps out a slow version of the function
func ProvisionTooSlow(done *chan auto.EventT, params auto.DataMapT, sase auto.EventT, w *auto.WorkerT) {

	sase = sase.Got().ID(nuid.Next())
	sase = sase.Str(fmt.Sprintf("Provisioning: %s ", params["node"].Dstring))

	time.Sleep(20 * time.Second * 20 / TIMESLICE)

	*done <- sase
}

// SlowProvisionTask - changes worker as above.
func SlowProvisionTask(t auto.TaskT) auto.TaskT {
	t.Forward.Fn = ProvisionTooSlow
	return t
}

// SlowProvisionFallback - changes worker as above.
func SlowProvisionFallback(t auto.TaskT) auto.TaskT {
	t.Forward.Fn = ProvisionTooSlow
	t.Fallback.Fn = ProvisionTooSlow
	t.Fallback.To = getslowtimeout()
	return t
}
