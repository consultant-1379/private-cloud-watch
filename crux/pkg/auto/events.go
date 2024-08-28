// (c) Ericsson Inc. 2016 All Rights Reserved
// Contributors:
//      Christopher W. V. Hogue

// TODO add logging

package auto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/erixzone/crux/pkg/clog"
)

// Debugging and Logging notes
// clog.Log levels used:
// DETAIL_INFO:   event loop start/stop
// DETAIL_MEDIUM: event in, timeouts hit, goals reached, child worker returning results, worker ending, holding, undoing
// DETAIL_FINE:   event cases detail, json worker dumps
// To get very verbose for debugging workers and events:
// Set w.AutoSave to true - combine with log output to DETAIL_FINE - logs every worker to json after applyRules()

// MOST of errors arise from the strings in each worker that declare the list of dependencies:
// E.G. this part.

//        task = task.AddDepends([]string{"reeveapi", "register"})
//        task = task.AddReverseDepends([]string{"ending"})
//        task = task.AddAllDepends([]string{"flock", "muck","selfkeys", "whitelistdb", "stable",
//                                           "watchleader", "watchregistry", "watchsteward", "reeveapi",
//                                           "launchreeve", "registrydb", "launchregistry",
//                                           "stewarddb","launchsteward","register"})

// AddDepends are things that must finish before a task can start, and their results are passed int as params
// ReverseDepends are things that must finish in reverse/inverse diretion before this task will do its
// inverse function
// AllDepends cuts across the entire workflow graph and lists all tasks that must happen before the
// list in AddDepends.  This is essential for reverse workflow to back up properly on concurrent tracks.

// Common mistakes.
// 1 Errors in after Inserting a task.
//  - Check for spelling mistakes or inconsistencies in new task string labels across all lists
//  - Make sure ReverseDepends of the prior task points to the new task
//  - Make sure ReverseDepends of the new task points to the next task
//  - Make sure AddDepends has the tasks that you are going to extract results from AND tasks that must finish first
//    before the new task (even if you don't use its results)
//  - Make sure AllDepends is complete.
// Tips
// Keep a drawing of the workflow graph with labels around - handy for checking while inserting/updating.

// Flawed dependency chains in the reverse direction can lead to a race condition with the worker
// termination:
// If a task is not listed as a ReverseDepends, the worker will not wait for it to finish, and worker may end
// prematurely before seeing events from its completion.  If so the json dump will show the inverse
// task still working. It may have finished, it is just that the worker event loop shut down so it never
// sees the .Inv() event that cleans this up:
//    "result": {
//        "dstring": ".muck/steward/steward.db"
//      },
//      "hasresult": true,
//      "working": true,
//

// EventLoop - The worker's event loop, NeSDA style with event forwarding and checkpoint.
func (w *WorkerT) EventLoop() {
	eventlog := clog.Log.With("focus", "events", "worker", w.Workerid)
	eventlog.Logi("msg", "Starting event loop", "worker", w.Name)
	for {
		select {
		case event := <-w.Events:

			// Event Forwarding to Hub
			if event.Workerid != w.Workerid || event.Broadcast || event.ToHub {
				// fmt.Printf("Not our event! %v\n", event)
				if w.Hub != nil {
					w.Hub.Family <- event
					break
				}
			}

			eventlog.Logm("msg", "EVENT IN", "worker", w.Name, "value", fmt.Sprintf("%v", event))
			err := w.handleEvents(event, eventlog)

			// save state after handleEvents() ?
			if err != nil {
				w.Lasterror = fmt.Sprintf("Worker: %s, UID: %s unrecoverable, after event [%v].", w.Name, w.Workerid, event)
				msg := fmt.Sprintf("%v", err)
				eventlog.Error("handleEvents() failed : " + msg + " worker=" + w.Name + "event= " + fmt.Sprintf("%v", event))
				jsonworker, jerr := w.WorkerToJSON()
				if jerr != nil {
					eventlog.Error(fmt.Sprintf("workerToJSON() failed : %v", jerr) +
						" worker=" + w.Name + " event=" + fmt.Sprintf("%v", event))
				} else {
					eventlog.Error("worker post handleEvents() error state dump" + " worker=" + w.Name +
						"json=\"" + fmt.Sprintf("%s", string(jsonworker[:])) + "\"")
				}

				eventlog.Logi("msg", "Stopping event loop", "worker", w.Name)
				return
			}

			w.applyRules(eventlog) // Checks Goal, runs tasks

			if w.AutoSave || w.Dienow {
				// save worker json state to log, always if w.AutoSave, and on ending
				msg0 := " "
				if w.Dienow { // mark info as 'worker final state'
					msg0 = " final "
				}
				jsonworker, jerr := w.WorkerToJSON()
				if jerr != nil {
					eventlog.Error(fmt.Sprintf("workerToJSON() failed : %v", jerr) +
						" worker=" + w.Name + " event=" + fmt.Sprintf("%v", event))
				} else {
					eventlog.Logf("msg", fmt.Sprintf("worker%sstate", msg0), "worker", w.Name,
						"json", fmt.Sprintf("%s", string(jsonworker[:])))
				}
			}
			if w.Dienow {
				// Send Done to Hub.hub
				// This is a control message that removes its event channel from the Hub
				devent := EventT{}
				devent.Msg = DONE
				devent.Workerid = w.Workerid
				devent.Taskkey = w.Goaltask
				devent.Timeless = true
				w.Hub.hub <- devent
				eventlog.Logi("msg", "Stopping event loop", "worker", w.Name)
				return
			}

		default:
			if w.Awake {
				time.Sleep(time.Second / 10000)
				w.checkTimeouts()
			}
		}
	}
}

func (w *WorkerT) handleEvents(e EventT, eventlog clog.Logger) error {

	// Can a RmSize event happen before GotSize event?
	// SO - In applyRules()
	//  - we don't dispatch t.Inverse.Fn when t.Working == true
	//  - it doesn't care about direction.
	//  - it will never send a size .Inverse.Fn while GotSize event pending.
	//  - meaning no .Inverse.Fn call possible until timeout hit or data returned.
	//     a lagging RmSize event may happen in an oscillating worker.

	// check the taskkey - if it doesn't match our list, ignore it
	w.Lock.RLock()
	t, ok := w.Tasks[e.Taskkey] // t is a copy of the task
	w.Lock.RUnlock()
	if !ok {
		if e.Msg != WAKE && e.Msg != DONE { // Wake/Done do not need a task
			msg := fmt.Sprintf("handleEvents() : bad task name specified; task not found : %s", e.Taskkey)
			eventlog.Error(msg + " worker=" + w.Name)
			return nil
		}
	}
	// check for EXPIRED events.
	if !e.Timeless && !e.Broadcast {
		// Check expiry time in event
		if time.Now().UTC().Sub(e.Expires) > 0 {
			eventlog.Logf("msg", "handleEvents() : expired event ignored", "worker", w.Name, "val", fmt.Sprintf("%v", e))
			// expired events are logged and ignored
			return nil
		}
	}
	switch e.Msg {
	case WAKE:
		eventlog.Logf("msg", "handleEvents() worker set to awake", "worker", w.Name)
		w.Awake = true // Wake up this worker
		// Reset the clock for worker timeout
		w.Started = time.Now().UTC()
		// expired case not considered here
	case DONE:
		msg1 := fmt.Sprintf("DONE  %s %s", e.Taskkey, e.Senderid)
		eventlog.Logf("msg", "handleEvents() worker done, finishing event loop", "worker", w.Name, msg1)
		if !w.Awake { // If worker is in a completed state
			w.Dienow = true // State that finishes event loop.
		}
	case ADDCHILD: // Adds a child worker nuid
		msg2 := fmt.Sprintf("ADD CHILD %s %s", e.Taskkey, e.Senderid)
		eventlog.Logf("msg", "handleEvents() add child worker", "worker", w.Name, msg2)
		w.RegisterChild(e.Taskkey, e.Senderid)
	case GOTCHILD: // Result inbound from a child worker
		msg3 := fmt.Sprintf("GOT CHILD %s %s", e.Taskkey, w.Name)
		eventlog.Logf("msg", "handleEvents() child worker done", "worker", w.Name, msg3)
		// Attach event Data to Child struct
		t.Children.Results[e.Senderid] = e.Data

		// Do we have all results in?
		var allresults = true
		var partial int
		for _, child := range t.Children.Workers {
			_, ok = t.Children.Results[child]
			allresults = allresults && ok
			if ok {
				partial = partial + 1
			}
		}
		if allresults {
			// Bundle results into a DataT.Djson as a json array of DataT
			result := DataT{}
			var merr error
			msg4 := fmt.Sprintf("Json marshaling child worker results %s", e.Taskkey)
			eventlog.Logf("msg", "handleEvents() bundling results", "worker", w.Name, msg4)
			result.Djson, merr = json.Marshal(t.Children.Results)
			if merr != nil {
				msg5 := fmt.Sprintf("handleEvents() : cannot JSON marshal child worker results %s : %v", e.Taskkey, merr)
				eventlog.Error(msg5 + " worker=" + w.Name)
			}
			result.Denc = JSONDataMap
			t.Result = &result
			t.HasResult = true
			t.Working = false
			t.Children.Partial = false
			t.Children.Results = nil // Remove redundant copy of Child Results
		} else {
			if float64(partial) > (float64(len(t.Children.Workers)) * PartialFraction) {
				t.Children.Partial = true
			}
			//fmt.Printf("All Results NOT IN YET\n")
		}
		if !t.Forward.Overtry { // Got from Forward
			t.Forward.Elapsed = time.Now().UTC().Sub(t.Forward.Ts)
		} else { // Got from Fallback
			if t.Fallback.Fn != nil {
				t.Fallback.Elapsed = time.Now().UTC().Sub(t.Fallback.Ts)
			}
		}
		w.Lock.Lock()
		w.Tasks[e.Taskkey] = t
		w.Lock.Unlock()
	case GOT:
		msg6 := fmt.Sprintf("GOT %s", e.Taskkey)
		eventlog.Logf("msg", fmt.Sprintf("handleEvents() task done %s", msg6), "worker", w.Name)
		t.Result = &e.Data
		t.HasResult = true
		t.Working = false
		if !t.Forward.Overtry { // Got from Forward
			t.Forward.Elapsed = time.Now().UTC().Sub(t.Forward.Ts)
		} else { // Got from Fallback
			if t.Fallback.Fn != nil {
				t.Fallback.Elapsed = time.Now().UTC().Sub(t.Fallback.Ts)
			}
		}
		w.Lock.Lock()
		w.Tasks[e.Taskkey] = t
		w.Lock.Unlock()
	case INV:
		msg7 := fmt.Sprintf("INV %s", e.Taskkey)
		eventlog.Logf("msg", "handleEvents() inverse task done", "worker", w.Name, msg7)
		t.Result = nil
		t.HasResult = false
		t.Working = false
		t.Inverse.Elapsed = time.Now().UTC().Sub(t.Inverse.Ts)
		w.Lock.Lock()
		w.Tasks[e.Taskkey] = t
		w.Lock.Unlock()
	case TIMEOUT: // Nondeterministic Event generated by CheckTimeouts()
		t.Working = false

		if w.Forward {
			if !t.Forward.Overtry {
				t.Forward.Errs = append(t.Forward.Errs, e.Data.Derr.Error())
				t.Forward.Overtime = true
				if t.Forward.Rt >= t.Forward.Maxrt {
					t.Forward.Overtry = true
					if t.Fallback == nil {
						msg9 := fmt.Sprintf("Excess forward retries at task %s", e.Taskkey)
						t.Forward.Failure = true
						w.Awake = false // Sleep until Rmv event wakes up the worker, or hard fail
						if w.UndoOnFail {
							msg10 := fmt.Sprintf("reversing workflow, via Rmv of starttask %s", w.Starttasks[0])
							// Sends Rmv event that will unwind all steps, which will fail after unwinding
							fevent := EventT{}
							fevent.Msg = RMV
							fevent.Workerid = w.Workerid
							fevent.Timeless = true
							fevent.Taskkey = w.Starttasks[0] // First starttask is invalidated
							fevent.Data.Derr = fmt.Errorf("%s, no fallback : %s", msg9, msg10)
							w.Events <- fevent
							eventlog.Logm("msg", "handleEvents() task timeout, unwinding",
								"worker", w.Name, fmt.Sprintf("%v", fevent.Data.Derr))

						} else {
							// Hard Fail Exit point... No rules processed...
							eventlog.Error(fmt.Sprintf("handleEvents() task %s hard timeout, no fallback : %v", e.Taskkey, e.Data.Derr) + " worker=" + w.Name)
							return e.Data.Derr
						}
					} else {
						eventlog.Logm("msg", "handleEvents() task timeout, try fallback", "worker", w.Name)
						t.Forward.Failure = true
						t.Fallback.Ts = time.Now().UTC()   // Fallback Task Start TS
						t.Fallback.TsRetry = t.Fallback.Ts // 1st Try Start TS
					}
				} else {
					t.Forward.Rt = t.Forward.Rt + 1 // one retry is counted
					eventlog.Logm("msg", "handleEvents() task timeout, retry", "worker", w.Name)
				}
			} else { // In Forward Overtry
				if t.Fallback.Fn != nil { // Timeout is in a Fallback call
					t.Fallback.Errs = append(t.Fallback.Errs, e.Data.Derr.Error())
					t.Fallback.Overtime = true
					if t.Fallback.Rt >= t.Fallback.Maxrt {
						t.Fallback.Overtry = true
						t.Fallback.Failure = true
						w.Awake = false // Sleep until Rmv event wakes up the worker, or hard fail
						w.Lock.Lock()
						w.Tasks[e.Taskkey] = t
						w.Lock.Unlock()
						if w.UndoOnFail {
							// Sends Rmv event that will unwind all steps, which will fail after unwinding
							fevent := EventT{}
							fevent.Msg = RMV
							fevent.Workerid = w.Workerid
							fevent.Timeless = true
							fevent.Taskkey = w.Starttasks[0] // First starttask is invalidated
							msg12 := fmt.Sprintf("reversing workflow, via Rmv of starttask %s", w.Starttasks[0])
							fevent.Data.Derr = fmt.Errorf("Excess fallback retries at task %s triggered worker reversal : %s", e.Taskkey, msg12)
							w.Events <- fevent
							eventlog.Logm("msg", "handleEvents() fallback task timeout, unwinding : "+
								fmt.Sprintf("%v", fevent.Data.Derr),
								"worker", w.Name)
						} else {
							// Hard Fail Exit point... No rules processed...
							eventlog.Error("handleEvents() fallback task hard timeout : " +
								fmt.Sprintf("Excess fallback retries at task %s : %v", e.Taskkey, e.Data.Derr) +
								" worker=" + w.Name)
							return e.Data.Derr
						}
					} else {
						t.Fallback.Rt = t.Fallback.Rt + 1 // one Fallback retry is counted
					}
				}
			}
		} else { // Inverse Fn Timeout
			t.Inverse.Errs = append(t.Inverse.Errs, e.Data.Derr.Error())
			t.Inverse.Overtime = true
			if t.Inverse.Rt >= t.Inverse.Maxrt {
				eventlog.Error(fmt.Sprintf("handleEvents() inverse task function timeout : %v\"", e.Data.Derr) +
					" worker=" + w.Name)
				t.Inverse.Overtry = true
				t.Inverse.Failure = true
				w.Lock.Lock()
				w.Tasks[e.Taskkey] = t
				w.Lock.Unlock()
				return e.Data.Derr
			}
			t.Inverse.Rt = t.Inverse.Rt + 1 // one retry is counted
		}
		w.Lock.Lock()
		w.Tasks[e.Taskkey] = t
		w.Lock.Unlock()
	case CANCEL:
		eventlog.Logf("msg", "handleEvents() cancel task boolean set in", "worker", w.Name, "task", e.Taskkey)
		t.Cancel = true
		w.Lock.Lock()
		w.Tasks[e.Taskkey] = t
		w.Lock.Unlock()
		// This is a bool for any user Fn to pick up in any long running loop in which it holds a pointer to the worker
		// then it sends a Fail event.
	case FAIL:
		// fmt.Printf("FAIL EVENT From %s %s.\n", e.Taskkey, w.Name)
		// 4xx Client error - our bad
		eventlog.Error(fmt.Sprintf("handleEvents() fail event in %s on task %s", w.Name, e.Taskkey))
		t.Working = false
		if w.Forward {
			if !t.Forward.Overtry {
				if e.Data.Derr != nil {
					t.Forward.Errs = append(t.Forward.Errs, e.Data.Derr.Error())
				}
				t.Forward.Elapsed = time.Now().UTC().Sub(t.Forward.Ts)
			} else {
				if t.Fallback.Fn != nil { // Fail is in a Fallback call
					if e.Data.Derr != nil {
						t.Fallback.Errs = append(t.Fallback.Errs, e.Data.Derr.Error())
					}
					t.Fallback.Elapsed = time.Now().UTC().Sub(t.Fallback.Ts)
				}
			}
			w.Awake = false // Sleep until Rmv event wakes up worker, or hard fail
			w.Lock.Lock()
			w.Tasks[e.Taskkey] = t
			w.Lock.Unlock()
			if w.UndoOnFail {
				// Sends Rmv event that will unwind all steps, which will fail after unwinding
				fevent := EventT{}
				fevent.Msg = RMV
				fevent.Workerid = w.Workerid
				fevent.Timeless = true
				fevent.Taskkey = w.Starttasks[0] // First starttask is invalidated
				msg14 := fmt.Sprintf("handleEvents() fail event at %s triggered workflow reversal, via Rmv of starttask %s", e.Taskkey, w.Starttasks[0])
				fevent.Data.Derr = fmt.Errorf(msg14)
				eventlog.Error(msg14+" worker=", w.Name)
				w.Events <- fevent
			} else {
				eventlog.Error(fmt.Sprintf("handleEvents() hard fail evnt - no rules processed : %v ", e.Data.Derr) +
					" worker=" + w.Name)
				return e.Data.Derr
			}
		} else { // FAIL while going in reverse direction. Keep going backward, save the error
			if e.Data.Derr != nil {
				t.Inverse.Errs = append(t.Inverse.Errs, e.Data.Derr.Error())
			}
			t.Result = nil
			t.HasResult = false
			t.Inverse.Elapsed = time.Now().UTC().Sub(t.Inverse.Ts)
			w.Lock.Lock()
			w.Tasks[e.Taskkey] = t
			w.Lock.Unlock()
		}
	case RMV:
		eventlog.Logi("msg", "handleEvents() remove event", "worker", w.Name, "task", e.Taskkey)
		// This part reverses direction when necessary
		// If an earlier fail event set the goal
		// upstream in the dependencies of the current event's task,
		// don't change the goal settings again!
		// or it won't finish going all the way backwards
		// (i.e. it was already going backwards!)
		upstream := false
		for _, dep := range t.Alldeps {
			upstream = upstream || dep == w.Goaltask
		}
		if !upstream {
			eventlog.Logi("msg", "handleEvents() remove event reversing worker direction", "worker", w.Name, "task", e.Taskkey)
			w.Goaltask = e.Taskkey // Make this event the goal
			w.Goalstate = false    // Make hasResult = false the goal
			w.ForwardGoal = false  // invalidate any ForwardGoal achieved.
			w.Forward = false      // Use backwards applyRules()
			w.Lasterror = fmt.Sprintf("Worker: %s, UID: %s Rmv event [%v].", w.Name, w.Workerid, e.Data.Derr)
			// If this is a start task, there is NO U-turn,
			// as base worker inputs are invalidated, so when
			// at goal, applyRules() will terminate the worker
			isstart := false
			for _, start := range w.Starttasks {
				isstart = isstart || e.Taskkey == start
			}
			w.Undo = !isstart
			w.Awake = true // Wake up this worker
		}
	}
	return nil
}
