// (c) Ericsson Inc. 2015-2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package auto

import (
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nuid"
)

// MapArgData - Helper function for organizing Worker arguments.
func MapArgData(args []string, values ...DataT) (DataMapT, error) {
	inputmap := make(DataMapT)
	if len(values) == 0 {
		return inputmap, fmt.Errorf("MapArgData: No values found for arguments")
	}
	n := len(args)
	if n == 0 {
		return inputmap, fmt.Errorf("MapArgData: No arguments found for values")
	}
	if n != len(values) {
		return inputmap, fmt.Errorf("MapInputs: Input array size mismatch; len(args) = %d; len(values) = %d", len(args), len(values))
	}
	for i, value := range values {
		arg := args[i]
		inputmap[arg] = value
	}
	return inputmap, nil
}

// NewWorker - Helper function for creating a new worker
func NewWorker(name string, goal string,
	starttasks []string,
	tasks ...TaskT) WorkerT {

	w := WorkerT{}
	w.Name = name
	w.Goaltask = goal
	w.Goal = goal
	w.Goalstate = true
	w.Forward = true
	w.Starttasks = starttasks
	w.Tasks = make(map[string]TaskT)
	for _, t := range tasks {
		w.Tasks[t.Key] = t
	}
	w.TSorigin = time.Now().UTC().Format(time.RFC3339)
	w.Lock = &sync.RWMutex{}
	return w
}

// SetHold - sets the worker to hold
func (w *WorkerT) SetHold() {
	w.Hold = true
}

// Release - set the worker to release the hold
func (w *WorkerT) Release() {
	w.Hold = false
}

// SetWorkerInputs - helper function to set a worker's inputs
func (w *WorkerT) SetWorkerInputs(inputs TasktoDataMapT) {
	for key, task := range w.Tasks {
		datamap, ok := inputs[key]
		if ok == true {
			task.Inputs = datamap
			w.Tasks[key] = task
		}
	}
	w.Workerid = nuid.Next()
}

// StartWorker - starts a worker
func (w *WorkerT) StartWorker() {
	event := EventT{}
	event.Timeless = true
	event.Workerid = w.Workerid
	event.Msg = WAKE
	event.Taskkey = ""
	w.Events <- event
}

// Done sends an event to clear a worker that is resident in memory
// but asleep via w.Hold()
func (w *WorkerT) Done() {
	event := EventT{}
	event.Timeless = true
	event.Workerid = w.Workerid
	event.Msg = DONE
	event.Taskkey = ""
	w.Events <- event
}

// FailWorker - function that fails the worker at a task, with an error
func (w *WorkerT) FailWorker(task string, err error) {
	event := EventT{}
	event.Timeless = true
	event.Workerid = w.Workerid
	event.Msg = FAIL
	event.Taskkey = task
	event.Data.Derr = err
	w.Events <- event
}

// RegisterChild - registers a child worker to deliver a given task
func (w *WorkerT) RegisterChild(task string, childid string) {
	t := w.Tasks[task]
	if t.Children == nil {
		t.Children = &TaskWorkerT{}
		t.Children.Results = make(DataMapT)
	}
	t.Children.Workers = append(t.Children.Workers, childid)
	w.Tasks[task] = t
}

// Cancel - Helper method to set a cancel flag for a worker's task based on
// an event.
func (w *WorkerT) Cancel(sase EventT) bool {
	return w.CancelTask(sase.Taskkey)
}

// CancelTask - Helper method to poll during long-running tasks to determinie
// if some external event has flagged them to cancel themselves.
func (w *WorkerT) CancelTask(task string) bool {
	w.Lock.RLock()
	flag := w.Tasks[task].Cancel
	w.Lock.RUnlock()
	return flag
}

// ClearTask - Helper method to directly clear a task's cancel flag
func (w *WorkerT) ClearTask(task string) {
	// lock map for read
	w.Lock.RLock()
	t := w.Tasks[task]
	w.Lock.RUnlock()
	t.Cancel = false
	// Lock map for write
	w.Lock.Lock()
	w.Tasks[task] = t
	w.Lock.Unlock()
}

// FwdGoal - see if a holding worker has met its forward goal
func (w *WorkerT) FwdGoal() bool {
	return w.ForwardGoal
}
