/*
Package ruck manages the overall architecture for a compute node (Skaar would call this Mind).

In order to run more or less anything at all, ruck needs a bunch of components.

The first set (A) of components support executing (micro)services:
	1) picket (executing a service from a remote)
	2) reeve (per-node side of the strew service)
	3) steward (global: one per flock side of the strew service)
	4) mind
	5) networking (represented by the type Confab)

The second set (B) supports flock-wide services:
	1) proctor (global configurator for per-node services)
	2) pastiche (per-node data manipulator)
	3) healthcheck (per-node health service)

All services are invoked the same way:
	func plugin(quit <-chan bool, alive <-chan []crux.Fservice, net *Confab)
The quit channel says when to quit. Alive are the periodic healthchecks issued by *all* services
started by this plugin. The networking component is described by a *Confab, described below.

The only "trick" in all this is that the healthcare service needs to be run by mind;
it needs the health outputs as an input and so is a special case.

A note on per-component data:
	UUID: a unique uuid denoting the "name" of the this component
	filename: the file from which this component was loaded
	funcname: the function implementing this component
	t: valid until this time
and start/stop sequences also include
	seqno: all actions involving the same seqno are reported as a group. this is problematic
		on stopping components

How stuff starts up:
The goal is to make everything replaceable, but that can't happen. But we can get real close.
Here is our sequence:
	1) generate and start networking (directly via starting the flock service)
	2) start reeve (directly via the reeve service)
	3) if on leader, start steward (directly via the steward service)
	4) start mind (directly via the plugin server)
	5) start pastiche (directly via the pastiche service)

	<< at this point, we can configure the local pastiche to use a default plugin server.
	we can then use that functionality to load any new version of pastiche. >>

	6) send a new list of functions to mind:
		a) proctor
	7) (for testing: wait until things stabilise, and then exit.)
	8) at this point, all nodes are running. At some point, proctor needs to start
	up somewhere, and then it can transition to a newer version of pastiche, flocking (and mind?).

We observe that as long as mind only needs the existing plumbing infrastructure, then we are good to go.
If a new Mind needs a different infrastructure, then they cannot be upgraded in place.

Testing

	1) start up all services (A) above: picket, reeve/steward, mind, networking
		+ test networking, service population
	2) start pastiche by hand on one node:
		+ test discovery of builtin data
	3) start pastiche on another node
		+ test pastiche talking to one another
	4) test overlaying of the services in (A)
		+ for example, a new reeve
	5) add proctor
		+ test a count of 2 for proctor, after second one start, set to {1 + !leader}
*/
package ruck

import (
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
)

// DefaultExecutable shut up verify; TBD dubious -- should be fulcrum-plugin??
const DefaultExecutable = "/crux/bin/plugin_fulcrum"

// Admin define sthe initial horde
const Admin = "Admin"
// HeartQueue sets depth of heartbeat queues
const HeartQueue = 10

// HeartGap is the normal gap between heartbeats
const HeartGap = 2 * time.Second

// heartChan holds local heartbeats inbound
var heartChan chan []pb.HeartbeatReq

// local networking
var network **crux.Confab

// this is a cheat; we need to store the NetIDT for heartbeat and picket. otherwise, dean can't find them.
var heartbeatNID, picketNID idutils.NetIDT

func init() {
	heartChan = make(chan []pb.HeartbeatReq, 1000)
	network = nil
}
