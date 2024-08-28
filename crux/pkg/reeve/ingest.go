// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// Event loop that forwards Endpoint and Client registration information to Steward

package reeve

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/grpcsig"
)

// Ingest - the reeve ingestor internals
type Ingest struct {
	clevents chan *pb.ClientData
	epevents chan *pb.EndpointData
	done     chan bool
}

// StartIngest - starts the reeve ingest channels and events.
func startIngest() *Ingest {
	ingest := new(Ingest)
	ingest.clevents = make(chan *pb.ClientData)
	ingest.epevents = make(chan *pb.EndpointData)
	ingest.done = make(chan bool)
	go ingest.Events()
	return ingest
}

// Quit - stops the Event loop goroutines
func (c *Ingest) Quit() {
	c.done <- true
}

// IngestClient - sends event when reeve receives a RegisterClient call
func (c *Ingest) IngestClient(cl *pb.ClientData) {
	c.clevents <- cl
}

// IngestEndpoint - sends event when reeve receives an RegisterEndpoint
func (c *Ingest) IngestEndpoint(ep *pb.EndpointData) {
	c.epevents <- ep
}

// Events - the Ingest event loop.
func (c *Ingest) Events() {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "Reeve Events started")
	for {
		select {
		case <-c.done:
			close(c.clevents)
			close(c.epevents)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "Reeve Events stopped")
			return
		case clevent := <-c.clevents: // ClientData arrival
			ack, err := clientUpdateSteward(clevent)
			if err != nil {
				msg1 := fmt.Sprintf("clientUpdateSteward failed : %v", err)
				pidstr, ts := grpcsig.GetPidTS()
				logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg1)
			}
			msg2 := fmt.Sprintf("clientUpdateSteward - Steward processing client %v", ack)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg2)
			// TODO push the competed or fail to local store (completed = steward has it, not state broadcast)
			// here completed means tx to steward (not steward has fan-outed the client)
			//
		case epevent := <-c.epevents: // EndpointData arrival
			ack, err := endpointUpdateSteward(epevent)
			if err != nil {
				msg3 := fmt.Sprintf("endpointUpdateSteward failed : %v", err)
				pidstr, ts := grpcsig.GetPidTS()
				logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg3)
			}
			msg4 := fmt.Sprintf("endpointUpdateSteward - Steward processing endpoint %v", ack)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg4)
			// TODO push the competed or fail  (completed = steward has it, not state broadcast)
			// here completed means tx to steward (not steward has fan-outed the endpoint)
		default:
			// do nothing
		}
	}
}

// PingSleep - blocks, pings with delay intervals, until we get a response from server
// or total time exceeds timeout
func PingSleep(client pb.StewardClient, delay time.Duration, timeout time.Duration) error {
	var total time.Duration
	// Make a ping
	ping := &pb.Ping{Value: pb.Pingu_PING}
	// Ping until we get the server response
	for {
		resp, cerr := client.PingTest(context.Background(), ping)
		if cerr != nil {
			total = total + delay
			time.Sleep(delay)
		} else {
			if resp.Value == pb.Pingu_PONG {
				return nil
			}
		}
		if total > timeout {
			return fmt.Errorf("PingSleep to steward - blocking exceeded timeout")
		}
	}
}

// wakeUpSteward - puts the event loop on hold until PingSleep returns with a working
// authenticated connection to Steward.
func wakeUpSteward(totaltime time.Duration) error {
	if ReeveState == nil {
		return fmt.Errorf("wakeUpSteward - reeve has no state information about steward")
	}
	if ReeveState.stewardsigner == nil {
		return fmt.Errorf("wakeUpSteward - reeve has no signer for steward grpcsignatures set")
	}
	conn, err := ReeveState.stewardsigner.Dial(ReeveState.stewardaddress)
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("wakeUpSteward - dial to steward at %s failed : %v", ReeveState.stewardaddress, err)
	}
	stewcli := pb.NewStewardClient(conn)
	return PingSleep(stewcli, 1*time.Second, totaltime)
}

// clientUpdateSteward - dials steward, sends Client data
func clientUpdateSteward(clidata *pb.ClientData) (*pb.Acknowledgement, error) {
	ack := &pb.Acknowledgement{} // empty Ack
	if ReeveState == nil {
		return ack, fmt.Errorf("reeve has no state information")
	}
	if ReeveState.stewardsigner == nil {
		return ack, fmt.Errorf("no signer for steward grpcsignatures set")
	}
	conn, err := ReeveState.stewardsigner.Dial(ReeveState.stewardaddress)

	defer conn.Close()
	if err != nil {
		return ack, fmt.Errorf("failed to connect to steward at %s : %v", ReeveState.stewardaddress, err)
	}
	stewcli := pb.NewStewardClient(conn)
	return stewcli.ClientUpdate(context.Background(), clidata)
}

// endpointUpdateSteward - dials steward, sends Endpoint data
func endpointUpdateSteward(epdata *pb.EndpointData) (*pb.Acknowledgement, error) {
	ack := &pb.Acknowledgement{} // empty Ack
	if ReeveState == nil {
		return ack, fmt.Errorf("reeve has no state information")
	}
	if ReeveState.stewardsigner == nil {
		return ack, fmt.Errorf("no signer for steward grpcsignatures set")
	}
	conn, err := ReeveState.stewardsigner.Dial(ReeveState.stewardaddress)

	defer conn.Close()
	if err != nil {
		return ack, fmt.Errorf("failed to connect to steward at %s : %v", ReeveState.stewardaddress, err)
	}
	stewcli := pb.NewStewardClient(conn)
	return stewcli.EndpointUpdate(context.Background(), epdata)
}
