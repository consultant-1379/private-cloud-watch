// (c) Ericsson Inc. 2016 All Rights Reserved
// Contributors:
//      Christopher W. V. Hogue

package auto

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nuid"
)

// The Tomaton Hub
// ===============
//
// NOTE, This part of the code is still evolving.
//
// (A) Allocates/manages all the event channels for a family of workers,
// facilitating  routing of child-parent, broadcast events, peer-to-peer
// events (using Go channels carrying EventT), and
//
// (B) Collects user-defined status/progress/monitoring events using the
// same routing system, funneling them into user-supplied functions
// called Nablas.
//
// In (A), for example,  a worker may have child workers launched by user tasks.
// The Hub provides the means for worker-worker event passing within the group.
// Workers are identifed by w.Workerid which is an NUID
// string like so: "OTWW0GS08JMUQ7O9DBMZ5O"
//
// In (B) The Hub has its own event loop, which can be hooked into via Nabla
// functions. A Nabla function may collect progress updates from some
// concurrent group of workers, it may provide a central log for the group,
// or it may provide higher-ordered operations (e.g. gradient operators)
// that monitor or control a family of workers. This functionality has
// a strong dependence on (A), which is why Nabla functions reside in the Hub,
// rather than outside. Nabla EventT are forwarded when event.ToHub = true,
// and event.NCode is used to specify a specific Nabla function.
//
// For (A) a Hub employs a map - (like this, but sharded)  map[nuid]chan EventT
// as a switchboard to send events to any worker in the family.
//
// Internal events related to a worker traversing its own graph of tasks,
// are not forwarded to the Hub.
//
// For (A) a Hub's routing of EventT is an in-memory system that is not
// serializable for persistant storage or retrieval, as it only holds
// channels and worker NUIDs in a family group.
//
// For (B) the user-defined Nabla can hook data into the HubT structure
// via the NablaSetT interface. When Nabla functions are present, that
// portion of (B) becomes serializable to JSON.
//
// The Hub does not hold pointers to Workers, so there is no way for
// the actions of the Hub to meddle with a worker's state outside
// of that worker's running event loop and message passing..
// i.e. It can only communicate with a worker in its group via an
// event on the HubT.Family channel.
//
// The only exception to this is when the Launch() command is issued
// with a pointer to the worker, which is provided so that:
// 1. The worker's event channel can be installed/managed by the Hub.
// 2. The Hub's pointer can be installed in the worker for routing.
// Note that 1 & 2 happen before the worker's event loop is started,
// and the Launch command polls the Hub for the appearance of the
// worker's event channel before starting the worker, so that the
// worker cannot not race ahead of the Hub.
//

// Hub Starting, Launching and Stopping
//
// Briefly:
// myhub := auto.StartNewHub(0) - Starts a Hub (0 = no sharding, see sharding below)
// myhub.Launch(&Worker, false) - Launches a worker, no caching of its event channel.
// myhub.DelWorker(&Worker) - Removes a worker. Workers do this by themsleves when done.
// myhub.WorkerCount() - Returns the number of workers left, Poll unitl 0 - all are finished.
// myhub.Stop(false) - Stops the Hub, false - don't serialize HubT to JSON.
//
// A Bit More:
//
// Hub Shard Control
// -----------------
// myhub := StartNewHub(0) = 1 shard (as in - not sharded)
// myhub := StartNewHub(1) = 36 shards
// myhub := StartNewHub(2) = 1296 shards
//             numbers >2 or <0 are adjusted to 2 or 0 respectively.
//
// Hub with Nablas
// ---------------
// Add in optional ...NablaSetT arguments like so:
// myhub := StartNewHub(0, usercode1.UserNabla1(), usercode2.UserNabla2())
// Starts a hub, hooking in 2 user-defined NablaSetT to its event loop and struct.
// See tomaton/nablademo/grabrace/grabrace.go and tomaton/nablademo/grabber/nabla.go
// for an example.
//
// Worker Launching
// ----------------
// myhub.Launch(&Worker, true) registers a new worker in the hub, notifies parent
// worker (if exists) by passing an event to it, and starts the worker. The boolean
// indicates to the Hub that it cache the worker's event channel, avoiding the
// map for lookups. Presumed useful when a Parent worker has large numbers of child
// workers - tested out to over 100K child workers.
//
// Worker Removal
// --------------
// myhub.DelWorker(&Worker) removes a worker from the hub.
// This is automatically sent when the worker hits it forward goal or when
// w.Dienow = true, e.g. when a Worker meets some Inv goal after a Cancel event.
//
// When is a Hub Done?
// -------------------
// myhub.WorkerCount() returns the number of worker event channels in the hub,
// and may be used to determine when a family of workers has finished, before
// calling myhub.Stop(false).

// Hub lock contention problem Go 1.4 - 1.6:
// =========================================
//
// In Go 1.4 torture testing was yielding 2200 child workers/sec in a hub,
// on tests of over 100K child workers.
//
// Go 1.6 hit a fatal error when large numbers of fast running children are being
// added & removed from the hub's first map[] implementation.
//
// So - Go 1.6 forces a mutex lock around map[string] to avoid unsafe access to the map.
//
// Solution is a apparently a RW mutex and if required -  a sharded map system.
// A single RW mutex is sufficient to fix the fatal-error problem, together with
// a sleep added to checkTimeouts(), which helped fix the Go 1.6 slowdown issue.
//
// I have layered in sharding and a cache, possibly code  overkill, but flame-graphing
// suggests this does break up things.  When it is all turned on, profiling shows
// worker stacks increasing in sampling frequency compared to all the system stacks.
// While it appears to do better with sharding, it does not affect the total
// runtime:
//
//                        w.eventLoop()  time (no significant differences!)
// RW mutex only                          93.7 s
// 1     shard  - cache    4.36%%         94.2 s
// 1     shard  + cache    6.96%          94.5 s
// 36    shards + cache   11.53%          93.5 s
// 36*36 shards + cache   15.50%          93.3 s
//
// I haven't devised a runtime test that can yet tell the difference when the
// system is saturated with workers whether the shard/caching is an improvement in
// throughput.
//
// So - at Go 1.6 I am  hitting 700 child workers/sec on the same system
// after adding the single mutex.
// It may be my timeouts are governing the throughput here, or it is a Go wall that
// doesn't matter how many mutexes you deploy.

// Go 1.4 to 1.6 Outcome:
//
// Despite flame graphing and exploring parameter space,
// I can only recover about 700 child workers/sec, at GOMAXPROCS = 8,
// it is only hitting about 1.3 CPUs.  So at least it is not running hot.
// I have not gone back and re-explored timeout settings, so I am not
// 100% sure whether it can ever match Go 1.4 again.

// Sharding implementation:
//
// I shard the Hub's event channel map with an outermost RO map to
// group operations on innermost RW map. The outermost map
// is a preallocated map with all the shard keys combinations in it.
//
// Conveniently, the NUID rightmost characters are suitable for shard keys.
// That is I don't need to make a new hash, I just use the last characters of the
// NUID for the shard space.
//
// For example, the last character of NUID as shard key string supports 36 shards
// and the last 2 chars yeids 1296 shards.
//
// For most workflows without children or workflows with
// number of children < GOMAXPROCS * 10 , sharding does not make much difference.
// For minimal shard initialization overhead, the hub can be
// instantiated with a single shard accessed internally with string "0" as key.

// EventShardT - Innermost shard with map to EventT channels and its mutex lock
type EventShardT struct {
	fork map[string]chan EventT
	lock *sync.RWMutex
}

// EventMapT -  Outermost map of shards
type EventMapT map[string]*EventShardT

// HubT - The Hub struc - not serialized.
type HubT struct {
	Hubid      string      `json:"hubid"`    // nuid for this hub
	NablaSets  []NablaSetT `json:"nablaset"` // User Defined Nablas & data
	NablaCodes map[int]int `json:"nablamap"` // User Defined Nabla Routing Code -> NablaSets index
	shards     EventMapT   // Sharded up event switchboard map[NUID]chan EventT
	schars     int         // Number of trailing characters in an NUID
	// for a shard. 1 gives 36 shards, 2 is 36x36 = 1296 shards
	// 0 gives 1 shard with key string "0"
	Family chan EventT       `json:"-"` // inbound events for the hub to route
	hub    chan EventT       // management events add, remove workers
	stop   chan int          // stop this hub's event loop
	cache  []WorkerChannelsT // Cache of parent channels
}

// NablaFnT - the nabla function type
type NablaFnT func(event EventT, hub *HubT) // A Nabla Function Type

// NablaSetT - the nabla set type
type NablaSetT interface {
	Name() string
	Function() NablaFnT
	Code() int
}

// WorkerChannelsT - used to get all worker/channel pairs for broacast events,
// and for the HubT.cache
type WorkerChannelsT struct {
	Workerid     string
	Workerevents *chan EventT
}

// These constants derive from NUID specifications:
const shardchars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const shardbase = 36
const nuidlen = 22

// NewEventMap - Populates the precomputed shards for the EventMapT,
// Write once, so any outer shard may be accessed without mutex locking
// schars -> only 0, 1, 2 allowed.
func NewEventMap(schars int) (EventMapT, int) {
	var count = 1
	if schars >= 2 {
		schars = 2 // Capped for combinatorical exposion (3 = 46556 shards!)
		count = 1296
	}
	if schars == 1 {
		count = 36
	}
	if schars <= 0 {
		schars = 0
	}

	// count is either 1, 36, or 1296
	events := make(EventMapT, count)

	switch schars {
	case 0: // Only 1 shard to pre-allocate
		//fmt.Printf("Single Shard\n")
		events["0"] = &EventShardT{
			fork: make(map[string]chan EventT),
			lock: new(sync.RWMutex),
		}
	case 1: // 36 shards to pre-allocate
		//fmt.Printf("36 Shards\n")
		for j := 0; j < shardbase; j++ { // total number of shards
			events[string(shardchars[j])] = &EventShardT{
				fork: make(map[string]chan EventT),
				lock: new(sync.RWMutex),
			}
		}
	case 2: // 36 * 36 = 1296 Shards to pre-allocate
		//fmt.Printf("1296 Shards\n")
		var mkey [2]byte
		for k := 0; k < shardbase; k++ {
			mkey[0] = shardchars[k]
			for l := 0; l < shardbase; l++ {
				mkey[1] = shardchars[l]
				events[string(mkey[:])] = &EventShardT{
					fork: make(map[string]chan EventT),
					lock: new(sync.RWMutex),
				}
			}
		}
	}
	return events, schars
}

// init - Initializes a Hub.
// When a Hub is started, the entire EventShard is instantiated
// based on the value of hub.schars * shardbase, and the
// outer shard is initialized with all possible strings of size
// hub.schars from shardchars in one pass.
// When hub.schars = 0, there is only a single shard initialized
// with the string "0", as its key, to make sharding effectively
// "optional".
// for h.init(), possible values are 0, 1 or 2. for shardsz.
func (h *HubT) init(shardsz int) {
	h.Hubid = nuid.Next() // Id of this hub - for hub/group count control purposes
	h.shards, h.schars = NewEventMap(shardsz)
	h.Family = make(chan EventT, 200)
	h.hub = make(chan EventT, 20) // Events adding/removing workers
	h.stop = make(chan int)       // Channel to shut down the hub's event loop
}

// manage - Hub management channel - inbound on h.HubEvents() -  Add/Remove workers
func (h *HubT) manage(change EventT) error {
	if change.Workerid != "" {
		switch change.Msg {
		case ADD:
			//fmt.Printf("Add Worker %s\n",change.Workerid)
			eventchan := h.emAdd(change.Workerid, change.Data.Dint)
			if change.Data.Dstring == "cache" {
				h.cache = append(h.cache, WorkerChannelsT{
					change.Workerid,
					eventchan,
				})
			}
			// change.Data.Dint holds the buffer size for the channel
			return nil
		case DEL, DONE:
			// fmt.Printf("Done Worker - Removing %s\n",change.Workerid)
			err := h.emRmv(change.Workerid)
			// fmt.Printf("Workers Remaining = %d\n", h.WorkerCount())
			return err
		default:
			return fmt.Errorf("Malformed manageHub event \"%s\" in handleWorkers, Worker: %s event:[%v]",
				change.Msg, change.Workerid, change)
		}
	}
	return fmt.Errorf("Malformed manageHub event \"%s\" in handleWorkers, Worker: %s event:[%v]",
		change.Msg, change.Workerid, change)
}

// WorkerEventChan - Get the Event channel for worker with given NUID
func (h *HubT) WorkerEventChan(workerid string) (chan EventT, error) {
	return h.emGet(workerid)
}

// hubEvents - A Hub's Event loop
//
// for - select loop provides processing for
// 3 inbound channels of each Hub
//
//   h.Family - Worker Family Events
//   h.hub    - Hub Management Events Add/Remove Workers
//   h.stop   - Stop Hub
//
// The design has all workers handling their own events
// with matching event.Workerid except when event.ToHub is
// flagged, for Nabla events.
//
// Events from user task functions are sent to its own worker
// on the 'done' channel, and the hub is used internally to forward
// any events destined for other workers in the group.
//
// Events may be sent from Nabla functions to h.Family to
// workers.
//
// If a worker receives an event with a different Workerid, it
// fowards it to the Hub without handling it, e.g. event from child
// worker addressed to its parent worker.
//
// If an event with Broadcast = true is seen by a worker,
// it is forwarded to the Hub via the HubT.Family channel,
// and the forwarding worker ignores it.
// Inside the hub, that event is copied and distributed to all workers
// in the group after setting Broadcast = false and personalizing the
// recipient Workerid. These events are then recognized and handled by
// each of the workers that recieve them.
func (h *HubT) hubEvents() {

	for {
		done := false
		select {
		case event := <-h.Family:
			//fmt.Printf("Event In hubEvent h.Family\n")

			if event.Broadcast {
				// fmt.Printf("Broadcasting Event...\n")
				event.Broadcast = false // Strip off Broadcast
				workers := h.emGetAll()
				for _, worker := range workers {
					// Send the Broadcast Event
					event.Workerid = worker.Workerid
					//fmt.Printf("%s\n",event.Workerid)
					// Personalize it so it is handled
					// Send
					*worker.Workerevents <- event
					//fmt.Printf("Sent Broadcast Event\n")

					// Note that only RMV or WAKE events wake up a worker.
					// If a worker is at GOAL, WAKE won't persist to the next event.
				}
			} else {
				if event.ToHub {
					// fmt.Printf("%s\n",event.Data.Dstring)

					// Nabla Event Hooks are called here
					// Check if Event has a user-registered NablaFn
					nablaidx, ok := h.NablaCodes[event.NCode]
					if ok {
						//Call that Nabla's function, passing the event
						call := h.NablaSets[nablaidx].Function()
						if call != nil {
							call(event, h)
						}
					}
				} else {
					workerevents, err := h.emGet(event.Workerid)
					// fmt.Printf("FORWARDING [%v]\n",event)
					if err == nil {
						workerevents <- event
					} else { // LOG THIS
						fmt.Fprintf(os.Stderr, "No such worker in hubEvents, error [%v]\n", err)
					}
				}
			}
		case change := <-h.hub:
			err := h.manage(change)
			if err != nil { // LOG THIS
				fmt.Fprintf(os.Stderr, "Worker change error in hubEvents, workerid: %s, error: [%v]", change.Workerid, err)
			}
		case halt := <-h.stop:
			_ = halt

			// should broadcast message to all workers - hmm...

			done = true
		}
		if done {
			break
		}
	}
	//fmt.Printf("Hub %s Stopped\n", h.Hubid)
}

// Stop2 - stops hub saves to a provided filename
func (h *HubT) Stop2(filename string) {

	if len(filename) > 0 && len(h.NablaSets) != 0 { // Nothing to save if no NablaFn hook)
		err := h.HubToFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR writing %s [%v]\n", filename, err)
		}
	}
	// Shut down the Hub's Event Loop
	h.stop <- 1
}

// Stop  - stops hub, saves to a ID based filename
func (h *HubT) Stop(save bool) {

	if save && len(h.NablaSets) != 0 { // Nothing to save if no NablaFn hook)
		filename := fmt.Sprintf("./%s_hub.json", h.Hubid)
		err := h.HubToFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR writing %s [%v]\n", filename, err)
		}
	}
	// Shut down the Hub's Event Loop
	h.stop <- 1
}

// Launch  takes a Worker, registers it in the Hub,
// waits for the Hub to present its event channel,
// connects that channel + and the Hub to the worker,
// then starts the worker.
//
// The chache parameter is for parent workers with
// large numbers of child workers being launched,
// This caches its event channel for fast lookup.
// For normal workers, cache should be false.
func (h *HubT) Launch(w *WorkerT, cache bool) {

	//fmt.Printf("In Hub %s Launch\n", h.Hubid)

	if w.Parentid != "" {
		// Send event to Register Worker with Parent
		childevent := EventT{}
		childevent.Msg = ADDCHILD
		childevent.Timeless = true
		childevent.Workerid = w.Parentid  // Event to Parent Worker
		childevent.Senderid = w.Workerid  // Child's nuid
		childevent.Taskkey = w.Parenttask // Parent Task
		h.Family <- childevent
	}

	// Send event to Add this Worker to the Hub via h.hub channel
	addevent := EventT{}
	addevent.Msg = ADD
	addevent.Timeless = true
	addevent.Workerid = w.Workerid
	addevent.Data = DataT{}
	addevent.Data.Dint = 10 // Channel Buffer Size
	if cache == true {
		addevent.Data.Dstring = "cache"
	}
	h.hub <- addevent

	//fmt.Printf("Add Worker %s Event Sent to Hub %s \n", w.Workerid, h.Hubid)

	// Poll Hub for worker event channel, so we don't race ahead
	// This is a sensitive optimization point - too short and it polls too often
	// - too long and it delays the hoarde of workers
	var eventchan chan EventT
	var err error

	Sleepytime := time.Second / 10000

	//fmt.Printf("Polling for Worker %s Event Channel in Hub %s\n", w.Workerid,h.Hubid)
	for {
		time.Sleep(Sleepytime)
		eventchan, err = h.WorkerEventChan(w.Workerid)
		if err == nil {
			break
		}
	}

	w.Events = eventchan
	w.Hub = h

	// Plumbed in, Now we can start the worker
	//fmt.Printf("Starting Worker Event Loop %s\n", w.Name)
	go w.EventLoop()

	// Send the start event
	//fmt.Printf("Starting Worker %s\n", w.Name)
	w.StartWorker()
}

// DelWorker - Sends itself a delete worker event
func (h *HubT) DelWorker(w *WorkerT) {
	delevent := EventT{}
	delevent.Msg = DEL
	delevent.Timeless = true
	delevent.Workerid = w.Workerid
	h.hub <- delevent
}

// StartNewHub - Spins up a Hub, Returns a pointer to it
// shardsz indicates sharding regime to use
// nablas are user-defined interfaces of NablaSetT that hook
// a function into the Hub's event loop.
// The Hub dispatches an appropriate NablaFn
// when the boolean event.ToHub is set true, and there
// is a matching NablaCode registered by the user code.
// NablaFns are not gorouties.
func StartNewHub(shardsz int, nablas ...NablaSetT) *HubT {
	Hub := HubT{}
	Hub.init(shardsz)
	// initialize map
	Hub.NablaCodes = make(map[int]int)
	// set up Nablas
	for n, nabla := range nablas {
		// Is this code already in use??
		i, present := Hub.NablaCodes[nabla.Code()]
		if present {
			fmt.Fprintf(os.Stderr, "ERROR: In StartNewHub() nabla %s has overlapping code %d, Cannot install nabla %s\n", Hub.NablaSets[i].Name(), nabla.Code(), nabla.Name())
			os.Exit(1)
		}
		Hub.NablaCodes[nabla.Code()] = n
		Hub.NablaSets = append(Hub.NablaSets, nabla)
	}
	//fmt.Printf("Hub %s Initialized\n", Hub.Hubid)
	go Hub.hubEvents()
	//fmt.Printf("Hub %s Running\n", Hub.Hubid)
	return &Hub
}

// emGet - and  emGetAll Concurrency-safe event channel access
func (h *HubT) emGet(key string) (chan EventT, error) {

	// check h.cache array first, before map
	for _, frequent := range h.cache {
		if key == frequent.Workerid {
			//fmt.Printf("CACHE HIT in emGet\n")
			return *frequent.Workerevents, nil
		}
	}

	var shardkey string
	if h.schars == 0 {
		shardkey = "0"
	} else {
		shardkey = key[nuidlen-h.schars : nuidlen]
	}
	//fmt.Printf("GET SHARD-KEY: %s\n",shardkey)
	shard := h.emGetShard(shardkey)
	shard.lock.RLock()
	//	fmt.Printf("SHARD: [%v]\n",shard)
	eventchan, ok := shard.fork[key]
	shard.lock.RUnlock()
	if ok {
		//fmt.Printf("OK!\n")
		return eventchan, nil
	}
	//fmt.Printf("NOT FOUND!\n")
	return nil, fmt.Errorf("Not found: %s", key)
}

// emGetAll Concurrency-safe event channel access
func (h *HubT) emGetAll() []WorkerChannelsT {
	wca := []WorkerChannelsT{}
	for _, shard := range h.shards {
		shard.lock.RLock()
		for nuid, eventchannel := range shard.fork {
			wc := WorkerChannelsT{}
			wc.Workerid = nuid
			wc.Workerevents = &eventchannel
			wca = append(wca, wc)
		}
		shard.lock.RUnlock()
	}
	return wca
}

// WorkerCount - returns number of event channels and workers
// that remain in the hub. Can be used to poll a hub to
// determine when all the workers are completed, and have
// removed themselves.
func (h *HubT) WorkerCount() int {
	count := 0
	for _, shard := range h.shards {
		count = count + len(shard.fork)
	}
	return count
}

// emAdd - emRmv  Add/Remove Hub event channels by Workerid
func (h *HubT) emAdd(key string, bufsize int) *chan EventT {
	var shardkey string
	if h.schars == 0 {
		shardkey = "0"
	} else {
		shardkey = key[nuidlen-h.schars : nuidlen]
	}
	//fmt.Printf("Add SHARD-KEY: %s\n",shardkey)
	shard := h.emGetShard(shardkey)
	eventchan := make(chan EventT, bufsize)
	shard.lock.Lock()
	shard.fork[key] = eventchan
	shard.lock.Unlock()
	return &eventchan
}

// emRmv  Add/Remove Hub event channels by Workerid
func (h *HubT) emRmv(key string) error {
	var shardkey string
	if h.schars == 0 {
		shardkey = "0"
	} else {
		shardkey = key[nuidlen-h.schars : nuidlen]
	}
	//fmt.Printf("Rmv SHARD-KEY: %s\n",shardkey)
	shard := h.emGetShard(shardkey)
	shard.lock.Lock()
	defer shard.lock.Unlock()
	_, ok := shard.fork[key]
	if ok {
		delete(shard.fork, key)
		return nil
	}
	return fmt.Errorf("Not found:  %s", key)
}

// emGetShard - Outermost shard (no mutex - it is preallocated preventing simultaneous R-W)
func (h *HubT) emGetShard(shardkey string) (shard *EventShardT) {
	return h.shards[shardkey]
}

// GetNablaData - returns the Nabla data
func (h *HubT) GetNablaData(code int) NablaSetT {
	i, ok := h.NablaCodes[code]
	if ok {
		return h.NablaSets[i]
	}
	return nil
}

// PutNablaData - Puts some Nabla Data
func (h *HubT) PutNablaData(code int, nablaset NablaSetT) {
	i, ok := h.NablaCodes[code]
	if ok {
		h.NablaSets[i] = nablaset
	}
}
