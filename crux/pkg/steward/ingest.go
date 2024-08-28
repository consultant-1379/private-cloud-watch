// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package steward

import (
	"container/list"
	"fmt"
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
	rb "github.com/erixzone/crux/pkg/registrydb"
	"github.com/pborman/uuid"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/grpcsig"
)

// Ingest - the steward ingestor internals
type Ingest struct {
	logger    clog.Logger
	clock     rb.StateClock
	timer     chan bool
	clockstop chan bool
	clevents  chan *rb.ClUpdate
	epevents  chan *rb.EpUpdate
	done      chan bool
}

// StartNewIngest - starts a new steward ingestor, at a provided (recalled if necessary from db) state count
func StartNewIngest(statecount int32, duration time.Duration, process FnProcessList, logger clog.Logger) *Ingest {
	ingest := new(Ingest)
	ingest.logger = logger
	ingest.clevents = make(chan *rb.ClUpdate)
	ingest.epevents = make(chan *rb.EpUpdate)
	ingest.done = make(chan bool)
	ingest.timer = make(chan bool)     // timer channel for state updates
	ingest.clockstop = make(chan bool) // done channel for clock goroutine
	ingest.clock.State = statecount
	go ingest.timekeeper(duration)
	go ingest.Events(0, process)
	return ingest
}

// GetState - retrieves the current state count in the Ingest event loop
func (c *Ingest) GetState() int32 {
	return c.clock.State
}

// Quit - stops the clock and Event loop goroutines
func (c *Ingest) Quit() {
	c.clockstop <- true
	c.done <- true
}

// timekeeper - async stoppable go routine for an event loop timer
func (c *Ingest) timekeeper(countdown time.Duration) {
	time.Sleep(countdown)
	for {
		select {
		case <-c.clockstop:
			msg := fmt.Sprintf("steward ingest timekeeper - ending clock")
			pidstr, ts := grpcsig.GetPidTS()
			c.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg)
			return
		default:
			c.timer <- true
			time.Sleep(countdown)
		}
	}
}

// IngestClient - sends event when steward recieves a Client update
//   allocates memory for adding to doubly linked list,
//   which will (in principle) get freed by GC once the list
//   is finished with.
func (c *Ingest) IngestClient(ci *pb.ClientData, reeveid string) string {
	ciu := new(rb.ClUpdate)
	ciu.TxUUID = uuid.NewUUID().String()
	ciu.ReeveKeyID = reeveid
	ciu.Nodeid = ci.Nodeid
	ciu.Keyid = ci.Keyid
	ciu.Keyjson = ci.Keyjson
	ciu.Status = ci.Status
	c.clevents <- ciu
	return ciu.TxUUID
}

// IngestEndpoint - sends event when steward receives an Endpoint Update
func (c *Ingest) IngestEndpoint(ep *pb.EndpointData, reeveid string) string {
	epu := new(rb.EpUpdate)
	epu.TxUUID = uuid.NewUUID().String()
	epu.ReeveKeyID = reeveid
	epu.Nodeid = ep.Nodeid
	epu.Netid = ep.Netid
	epu.Status = ep.Status
	c.epevents <- epu
	return epu.TxUUID
}

// FnProcessList - function callback to process the list of events
// when the timeboxed batch is ready to go (see ingest_test.go for example)
type FnProcessList func(dlhead *list.List, clock rb.StateClock)

// Events - the Ingest event loop.
//   memory buffers incoming client and enpoint updates.
//   allows the database updates and scatter of key/endpoint updates to the flock
//   proceed asynchronously, and be eventually consistent.
//   continuously bundles/buffers inbound client and endpoint updates into a timeboxed list
//   c.clock.State is the "state" number (timebox) here
func (c *Ingest) Events(laststate int32, process FnProcessList) {
	c.clock = rb.StateClock{}
	c.clock.State = laststate
	msg0 := fmt.Sprintf("steward ingest event loop, begin state %d", c.clock.State)
	pidstr, ts := grpcsig.GetPidTS()
	c.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg0)
	dlhead := list.New() // ptr to doubly linked list
	count := 0
	for {
		select {
		case <-c.done:
			close(c.clevents)
			close(c.epevents)
			msg1 := fmt.Sprintf("steward ingest event loop ending")
			pidstr, ts := grpcsig.GetPidTS()
			c.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg1)
			return
			// end this ingestor
			// NB: Anything accumulated since last c.timer is lost...
			// unless we use code from c.timer case to push anything remaining...
		case clevent := <-c.clevents: // ClientUpdate arrival
			// Add to current list
			if count == 0 {
				c.clock.Begin = time.Now()
			}
			dlhead.PushBack(clevent)
			count++
		case epevent := <-c.epevents: // EndpointUpdate arrival
			// Add to current list
			if count == 0 {
				c.clock.Begin = time.Now()
			}
			dlhead.PushBack(epevent)
			count++
		case <-c.timer:
			msg3a := fmt.Sprintf("steward ingest event clock tick")
			msg3b := ""
			if count > 0 { // only update state if something ingested
				msg3b = fmt.Sprintf(" end %d", c.clock.State)
				c.clock.End = time.Now()
				// send list pointer to next stage where it can
				// be inserted into DB, with copy of the clock state

				// CWVH reconsidered whether this should be a goroutine
				// don't want it to dangle so I can shut down db without
				// writes still attempting.
				// go process(dlhead, c.clock)
				process(dlhead, c.clock)

				// start new list, fresh clock state increment
				dlhead = list.New()
				c.clock.State++
				msg3b = msg3b + fmt.Sprintf(" begin %d", c.clock.State)
				c.clock.Begin = time.Time{} // zero out - begin only when data seen
				c.clock.End = time.Time{}   // zero out
				count = 0
			}
			pidstr, ts := grpcsig.GetPidTS()
			c.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg3a+msg3b)
		default:
			// do nothing
		}
	}
}
