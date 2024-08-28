// (c) Ericsson Inc. 2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// WIP - Timeouts/Fallbacks - some placeholder fmt.Printf comments persist

package auto

import (
	"fmt"
	"github.com/erixzone/crux/pkg/clog"
	"os"
	"time"
)

func (w *WorkerT) checkTimeouts() {
	// CheckTimeouts() is called only when no events are incoming
	// So any pending events in the buffer get processed
	// before we check for timeouts across tasks
	// The task timeouts are better defined in terms of behavior
	// than a "whole worker" timeout.
	if !w.Awake {
		// worker is in sleep state
		return
	}
	// The expiration of a task timer is Non-Deterministic
	// So checkTimeouts() here triggers Events (Timeouts)
	// which go back into the ProcessEvents/ApplyRules loop
	Now := time.Now().UTC()
	w.Lock.RLock()
	defer w.Lock.RUnlock()
	if w.Forward {
		for k, t := range w.Tasks {
			if t.Working {
				// See if this task retry has timed out
				if !t.Forward.Overtry {
					taskelapsed := Now.Sub(t.Forward.TsRetry)
					if SkepticalTimeout(t.Forward.To, taskelapsed, t.Forward.Conf, t.Forward.Transit) {
						//fmt.Printf("Forward Task %s OUT OF TIME! Worker-%s\n", k, w.Workerid)
						tevent := EventT{}
						tevent.Msg = TIMEOUT
						tevent.Workerid = w.Workerid
						tevent.Timeless = true
						tevent.Taskkey = k
						tevent.Data.Derr = fmt.Errorf("Timed Out, %s-Forward, Elapsed: %d At: %s",
							k,
							taskelapsed,
							Now.UTC().Format("2006-01-02T15:04:05.00Z07:00"))
						w.Events <- tevent
					}
				} else { // We may be in a fallback
					if t.Fallback.Fn != nil { // Yup
						taskelapsed := Now.Sub(t.Fallback.TsRetry)
						if SkepticalTimeout(t.Forward.To, taskelapsed, t.Forward.Conf, t.Forward.Transit) {
							//fmt.Printf("Fallback Task OUT OF TIME!\n")
							tevent := EventT{}
							tevent.Msg = TIMEOUT
							tevent.Workerid = w.Workerid
							tevent.Timeless = true
							tevent.Taskkey = k
							tevent.Data.Derr = fmt.Errorf("Timed Out, %s-Fallback, Elapsed: %d At: %s",
								k,
								taskelapsed,
								Now.UTC().Format("2006-01-02T15:04:05.00Z07:00"))
							w.Events <- tevent
						}
					} else {
						fmt.Fprintf(os.Stderr, "Bad Timeout State for Worker %s\n", w.Workerid)
						os.Exit(1)
					}
				}
			}
		}
	} else {
		for k, t := range w.Tasks {
			if t.Working {
				// See if it has timed out
				taskelapsed := Now.Sub(t.Inverse.TsRetry)
				if SkepticalTimeout(t.Inverse.To, taskelapsed, t.Inverse.Conf, t.Inverse.Transit) {
					//fmt.Printf("Inverse Task %s OUT OF TIME! %s, %s, %s \n", k, Now, t.Inverse.TsRetry, t.Inverse.Ts)
					tevent := EventT{}
					tevent.Msg = TIMEOUT
					tevent.Workerid = w.Workerid
					tevent.Timeless = true
					tevent.Taskkey = k
					tevent.Data.Derr = fmt.Errorf("Timed Out, %s-Inverse, Elapsed: %d At: %s",
						k,
						taskelapsed,
						Now.UTC().Format("2006-01-02T15:04:05.00Z07:00"))
					w.Events <- tevent
				}
			}
		}
	}
}

func (w *WorkerT) checkGoal(eventlog clog.Logger) {
	if !w.Awake {
		// Worker is in sleep state
		return
	}
	var StartTasksGoal = false
	for _, gkey := range w.Starttasks {
		StartTasksGoal = StartTasksGoal || gkey == w.Goaltask
	}
	w.Lock.RLock()
	defer w.Lock.RUnlock()
	if w.Tasks[w.Goaltask].HasResult == w.Goalstate {
		// A Global Goal achieved
		if w.Forward { // Forward Goal, worker is done!
			if !w.Goalstate {
				fmt.Fprintf(os.Stderr, "Assertion Failed in ApplyRules for Worker %s; Contradition in Goalstate Direction variables\n", w.Workerid)
				os.Exit(1)
			}
			if w.Undo {
				fmt.Fprintf(os.Stderr, "Assertion Failed in ApplyRules for Worker %s; Contradition in Undo Direction variables\n", w.Workerid)
				os.Exit(1)
			} else {
				// The goal is real, assertions passed.
				w.Elapsed = time.Now().UTC().Sub(w.Started)
				eventlog.Logm("msg", "worker reached forward goal", "worker", w.Name, "goal", w.Goaltask)
				if w.Hub != nil && w.Parentid != "" {
					eventlog.Logm("msg", "worker at goal is child - sending result to parent", "worker", w.Name, "parent, w.Parentid", "task", w.Parenttask)
					// Inform the Parent, send it
					event := EventT{}
					event.Msg = GOTCHILD
					event.Workerid = w.Parentid
					event.Senderid = w.Workerid
					event.Taskkey = w.Parenttask
					event.Data = *w.Tasks[w.Goaltask].Result
					event.Expires = w.Parentexp
					w.Hub.Family <- event

				}
				if w.Hold == false {
					eventlog.Logm("msg", "worker at goal - ending", "worker", w.Name, "goal", w.Goaltask)
					w.Dienow = true
					w.Awake = false
				} else {
					eventlog.Logm("msg", "worker at goal - holding", "worker", w.Name, "goal", w.Goaltask)
					w.ForwardGoal = true
				}
			}
		} else { // Inverse cases - Global inverse goal achieved
			eventlog.Logm("msg", "worker reached inverse goal", "worker", w.Name, "goal", w.Goaltask)
			if StartTasksGoal {
				// All starttasks must match w.Goalstate and be (false)
				// Meaning everything has been undone - Before we can kill the worker
				allundone := true
				for _, st := range w.Starttasks {
					if w.Tasks[st].HasResult != w.Goalstate {
						allundone = false
						w.Goaltask = st // Move Goaltask up on the Starttask to do list
					}
				}

				if allundone == true { // Start tasks are invalid, meaning worker must end.
					eventlog.Logm("msg", "worker undone back to start - ending", "worker", w.Name, "goal", w.Goaltask)
					w.Awake = false // Sleep worker
					w.Dienow = true // remove from hub
				}
				// otherwise keep going.
			}
			if w.Undo { //  Reverse inverse direction with Undo - Continue processing
				eventlog.Logm("msg", "worker to go foward again", "worker", w.Name, "goal", w.Goaltask, "newgoal", w.Goal)
				w.Forward = true
				w.Undo = false
				w.Goaltask = w.Goal
				w.Goalstate = true
			}
		}
	}
}

func (w *WorkerT) applyRules(eventlog clog.Logger) {
	// Are we at our Goal?
	w.checkGoal(eventlog)
	// Are we sleeping?
	if !w.Awake {
		// Worker is in sleep state
		return
	}
	Now := time.Now().UTC()
	// Are we moving Forward?
	w.Lock.Lock()
	defer w.Lock.Unlock()
	if w.Forward {
		// Dispatch any task whose input/dependencies are ready to go
		for k, t := range w.Tasks {
			if t.Working {
				continue
			}
			if !t.HasResult {
				if len(t.Depends) != 0 {
					// Task has listed dependencies on other tasks
					var depsok = true
					for _, dep := range t.Depends {
						depsok = depsok && w.Tasks[dep].HasResult
					}
					if depsok {
						// all dependencies are ready
						// populate t.Inputs from dependent results
						t.Inputs = make(DataMapT)
						for _, depstr := range t.Depends {
							if w.Tasks[depstr].Result != nil {
								t.Inputs[w.Tasks[depstr].Key] = *(w.Tasks[depstr].Result)
							}
						}
						// save state on the worker, not the local t. copy:
						task := w.Tasks[k]
						task.Working = true
						if !t.Forward.Overtry {
							if !task.Forward.Overtime {
								task.Forward.Ts = Now
							}
							task.Forward.TsRetry = Now
							expires := Now.Add(t.Forward.To)
							w.Tasks[k] = task
							sase := EventT{}
							sase = sase.To(w.Workerid).Expiry(expires).Task(task.Key)
							go t.Forward.Fn(&w.Events, t.Inputs, sase, w)
						} else {
							if t.Fallback.Fn != nil {
								if !task.Fallback.Overtime {
									task.Fallback.Ts = Now
								}
								task.Fallback.TsRetry = Now
								expires := Now.Add(t.Fallback.To)
								w.Tasks[k] = task
								sase := EventT{}
								sase = sase.To(w.Workerid).Expiry(expires).Task(task.Key)
								go t.Fallback.Fn(&w.Events, t.Inputs, sase, w)
							}
						}
					}
				} else {
					// Reminder - t.Input is supplied by the
					// worker constructor for any task with external
					// inputs and no dependencies on other tasks,
					//  even if nil values
					// save state on the worker, not the t. transient copy:
					task := w.Tasks[k]
					task.Working = true
					if !task.Forward.Overtime {
						task.Forward.Ts = Now
					}
					expires := Now.Add(t.Forward.To)
					if !t.Forward.Overtry {
						task.Forward.TsRetry = Now
						w.Tasks[k] = task
						sase := EventT{}
						sase = sase.To(w.Workerid).Expiry(expires).Task(task.Key)
						go t.Forward.Fn(&w.Events, t.Inputs, sase, w)
					} else {
						if t.Fallback.Fn != nil {
							task.Fallback.TsRetry = Now
							w.Tasks[k] = task
							sase := EventT{}
							sase = sase.To(w.Workerid).Expiry(expires).Task(task.Key)
							go t.Fallback.Fn(&w.Events, t.Inputs, sase, w)
						}
					}
				}
			}
		}
	} else {
		// We are moving Backwards.
		// Backwards Rules Dispatcher
		// Theory here is that the inputs to a reverse function should be all the inputs to
		// the forward function (i.e. the same) plus the result of the forward function.
		for k, t := range w.Tasks {
			if t.Working {
				continue
			}
			hasgoaldep := false
			for _, gdep := range t.Alldeps {
				hasgoaldep = hasgoaldep || gdep == w.Goaltask
			}
			if w.Goaltask == t.Key {
				hasgoaldep = true
			}
			if t.HasResult && hasgoaldep {
				// Candidate for inv dispatching
				// recall t is a copy of the task, not on the worker
				invinput := make(DataMapT)
				invinput["result"] = *t.Result // Results from foward function
				if t.Depends != nil {
					// populate t.Inputs from Dependent results
					t.Inputs = make(DataMapT)
					for _, deps := range t.Depends {
						if w.Tasks[deps].Result != nil {
							t.Inputs[w.Tasks[deps].Key] = *(w.Tasks[deps].Result)
						}
					}
					// append these to the inputs
					for k, v := range t.Inputs {
						invinput[k] = v
					}
				}
				// inputs are assempled,
				if len(t.Revdeps) != 0 {
					// Task has reverse dependencies, should we wait?
					var revdepsok = true
					for _, revdep := range t.Revdeps {
						revdepsok = revdepsok && !w.Tasks[revdep].HasResult
					}
					if revdepsok {
						// Results & Inputs appended together, launch!
						task := w.Tasks[k]
						task.Working = true
						if !task.Inverse.Overtime {
							task.Inverse.Ts = Now
						}
						task.Inverse.TsRetry = Now
						expires := Now.Add(t.Inverse.To)
						w.Tasks[k] = task
						sase := EventT{}
						sase = sase.To(w.Workerid).Expiry(expires).Task(task.Key)
						go t.Inverse.Fn(&w.Events, invinput, sase, w)
					}
				} else {
					// Task has no reverse dependencies, no waiting...
					task := w.Tasks[k]
					task.Working = true
					if !task.Inverse.Overtime {
						task.Inverse.Ts = Now
					}
					task.Inverse.TsRetry = Now
					expires := Now.Add(t.Inverse.To)
					w.Tasks[k] = task
					sase := EventT{}
					sase = sase.To(w.Workerid).Expiry(expires).Task(task.Key)
					go t.Inverse.Fn(&w.Events, invinput, sase, w)
				}
			} // else task is excluded
		}
	}
}
