package ruck

import (
	"fmt"
	"time"

	"github.com/erixzone/crux/pkg/auto"
	"github.com/erixzone/crux/pkg/clog"
)

// Start with a simple nabla function that logs

// Nog - pushes the string in any event it sees to a clog.Log
func Nog(event auto.EventT, hub *auto.HubT) {
	// Fish out our logger from the hub
	noginfo := hub.GetNablaData(LOGEVENT).(nogNablaT)
	if event.Data.Derr != nil {
		ermsg := fmt.Sprintf("%v", event.Data.Derr)
		// Log an error
		noginfo.NablaLogger.Log("error", ermsg)
	}
	// Log info
	msg := fmt.Sprintf("%s", event.Data.Dstring)
	noginfo.NablaLogger.Log("info", msg)
}

// LOGEVENT A unique number for this Nabla
const LOGEVENT int = 9886

// nogNablaT Define a minimal struct to fulfil the NablaSetT interface
// by including Name(), Function(), Code() methods.
type nogNablaT struct {
	NablaName   string        `json:"nablaname"` // Name of this nabla
	NablaCode   int           `json:"nablacode"` // Unique Decoding int for this nabla's events
	NablaFn     auto.NablaFnT `json:"-"`         // Nabla Function
	NablaLogger clog.Logger   `json:"-"`         // Logger Function
}

// Name - returns the nabla name
func (n nogNablaT) Name() string {
	return n.NablaName
}

// Function - returns the nabla function
func (n nogNablaT) Function() auto.NablaFnT {
	return n.NablaFn
}

// Code - returns the nabla code
func (n nogNablaT) Code() int {
	return n.NablaCode
}

// nogger() - This is what goes in auto.StartNewHub()
func nogger() auto.NablaSetT {
	// allocate the Interface
	n := nogNablaT{}
	// set its values
	n.NablaName = "Nog"
	n.NablaFn = Nog
	n.NablaCode = LOGEVENT
	n.NablaLogger = clog.Log.With("focus", "nabla")
	return n
}

// BootstrapOrganzaX - is the follow-on fulcrum integration tester after Ripstop
// Uses the tomaton NeSDa event loop and reversible workflow
// Args
// port :  the port for flocking UDP
// skey :  the flocking key (cmd argument --key)
// ipname :  is the hostname of the node we are on
// ip is : ip address of the host or resolvable hostname of the host we are on
// beacon : is an Address (ip:port) intended as the flock leader (cmd argument --beacon)
// horde : is the name of the horde on which this process's endpoints are running.
// networks: a list of CIDR networks to probe. If this is blank, we probe the local network of the given ip.
func BootstrapOrganzaX(block string, port int, skey, ipname, ip, horde, beacon, networks, certdir string) {
	// Start up a Hub with a Nabla function that logs events
	// 0 arg means no sharding of Hub map
	Hub := auto.StartNewHub(0, nogger())

	// Make a Booter worker - the "Battle Royale" variant
	Booter := NewBooterWorker("Booter-Worker", "")

	// Apportion Booter's initial arguments
	flockargs, _ := auto.MapArgData([]string{"block", "port", "skey", "ipname", "ip", "horde", "beacon", "networks", "certdir"},
		auto.DataT{Dstring: block},
		auto.DataT{Dint: port},
		auto.DataT{Dstring: skey},
		auto.DataT{Dstring: ipname},
		auto.DataT{Dstring: ip},
		auto.DataT{Dstring: horde},
		auto.DataT{Dstring: beacon},
		auto.DataT{Dstring: networks},
		auto.DataT{Dstring: certdir})

	bInputs := make(auto.TasktoDataMapT)
	bInputs["flock"] = flockargs // Name of start task is "flock"

	// N.B. Any other start task args are made and apportioned as above,
	// e.g. "muck" could get a dir argument passed in..

	// Set Booter's Input arguments
	Booter.SetWorkerInputs(bInputs)

	// Booter.AutoSave = true  || SET to see json workers in logs
	// Launch the Booter
	Hub.Launch(&Booter, false)

	// wait for Booter.ForwardGoal == true

	// Extract params from Booter to send into level 2 workflow...
	// hmm. Maybe the "ending" task makes a struct that collects
	// the reeveapi, flockname, hordename, nodeName, principal
	// into at TasktoDataMapT and set that as the w.GoalResults
	// Maybe the ending task returns a list of tasksnames and
	// the event loop itslelf does the bundling when the
	// goal is reached. Same place as where w.ForwardGoal = true
	// is set. Drop the TasktoDataMak into w.GoalResults
	// so it can be daisy-chained to next worker.

	// Set up the foobar worker and launch in the same hub.

	// If the Booter worker goes all the way back to
	// "whitelist" where it will shut off grpcsig injectors
	// Broadcast a message to the other workers via the Hub
	// so that they shut down.

	// May be able to send an event into Booter identifying the
	// worker upstream. That could be the replacement for "ending"

	// Poll the Hub until it has no more workers running
	for {
		time.Sleep(2 * time.Second)
		if Hub.WorkerCount() == 0 {
			break
		}
	}
	// Stop the Hub, serialize it to json file in local directory for inspection.
	Hub.Stop2("boothub.json") // Saves Hub
}
