/*
Package flock implements a low-level cluster. Basically, it maintains a set of flocks, each with a
leader and a set of members. If two flocks learn of each other, they will merge into a single flock.
This package is meant to be used as a very low level clustering tool.

How do flocks work? Asymptotically, this is the activitity:
	a) nodes in a flock heartbeat to the leader
	b) the leader heartbeats to all the flock member nodes
	c) less frequently, the leader sends a full membership list to all the flock member nodes (TBD)
	d) leadership is stable (that is, a leader remains the leader until it goes away)
	e) flocks have a flock (UUID-like) name, shared by all the member nodes.
	f) periodically, each node probes for potential members

Flock building works simply:
	a) nodes start as a flock of one node
	b) when a node gets a packet from another node, it is either
		1) in the same flock, in which case we do leader-electing
		2) a different flock; we merge one flock (alphabetically lower flock name) into
			the other flock.

We have constructed our notions of identity and discovery to be fairly generic, but easily adapted
to the common cases of physical hardware, VMs and containers. An ID is a tuple of (name, address).
The name can be discovered (say by DNS) or assigned (by the user). If neither, then the package assigns
a random UUID-like name. The address is just a string (mostly, this will be an IP:port).

Discovery is part of the periodic probing mentioned above. There are three sources of addresses to probe:
	a) a specific address given in an admin command
	b) nodes we have seen in the past but are not currently members (these are forgotten after a while)
	c) a random node chosen from a given range of addresses (mostly, the range will be a CIDR)

Pivotal to all the above are encryption keys; all packets are encrypted by an AES-style symmetric key.
Obviously, key management is important. Here's how we do it:
	1) all nodes started with a so-called "secondary" key; we'll call it sec0
	2) rarely, an operator will change the secondary key (to newS):
		a) the operator will cause any new (or restarted) nodes to sec0==newS
		b) the operator will tell the existing flock leader to change the secondary key to newS
		c) the leader sets sec1 = sec0, sec0 = newS
		d) after the next two heartbeats, sec1 = nil
	3) the leader has a so-called "primary" key; we'll call it prim0
	4) every so often (think a minute), the leader chooses a new primary key. This means that
	the next heartbeat will
		a) set prim1 = prim0, and choose a new prim0
		b) the heartbeat will be sent using prim1 (because that is what all the members will be using)
	5) the leader heartbeat always advertises prim0 and sec0.
	6) by default, we encrypt packets with prim0. we decrypt with prim0, prim1, sec0, sec1 until we succeed.
	any packets that don't decrypt are discarded.
	7) when we get probes from other clusters, we know their prim0 so we can reply using that key.
	8) when we probe other (stranger) nodes, we need to use sec0 (because we don't know their primary keys)

The main consequence of this scheme is that if you change the secondary key, it is possible that nodes that are down
or in the process of flocking might not see that change to the secondary key. These nodes will then be
effectively marooned; they won't know any key that our flock uses. An operator will need to inject
the new secondary key changes into these nodes, or simply cause them to be restarted
(we presume the restarted nodes will have the new secondary key anyway).

You might ask how to send admin commands to the cluster (for example, update the secondary key).
You need a process that probes (using the old secondary key) the flock until it gets a response. Once you have a response,
you have prim0 and the leader. Send your command to that leader using prim0. You should
remain in the flock for a while; if the leader changes before your command takes effect, you may need to
resend it to the new leader.

Flocking is governed by two interfaces; one is node-oriented, the other is flock-oriented.
The node interface is
	type Fnode interface {
		Recv() *FlockInfo                      // receive a packet sent to me (via SetMe)
		Send(*FlockInfo, bool)                 // send a FlockInfo on prim0(true) or sec0(false)
		SetMe(*NodeID)                         // fill in unset fields
		SetKeys(prim0, prim1, sec0, sec1 *Key) // use nil for unused keys
		Logf(string, ...interface{})           // how we log stuff
		Monitor() chan crux.MonInfo                 // monitoring channel; nil is off
	}
SetMe fills blank parts of the ID, and remembers the resulting ID. Send sends a FlockInfo packet to its "dest" field.
(For Send, if the arg is true but prim0 is not set, it uses sec0.)
Recv yields the next packet addressed to the remembered ID. Keys are described below.

The flock interface governs overall cluster behaviour:
	type Fflock interface {
		Heartbeat() time.Duration	// period for sending heartbeats
		NodePrune() time.Duration	// lose flock membership if no heartbeats
		HistoryPrune() time.Duration	// forget old nodes after this
		Checkpoint() time.Duration	// drop a checkpoint after this period
		KeyPeriod() time.Duration	// period for rotating prim0
		Probebeat() time.Duration	// period between rework probes
		ProbeN() int			// how many probes per rework
		Probe() Address,bool		// return a random address within a given range
	}
This returns a number of timing "constants". These may change over time, and the new values are
expected to used within a couple of uses. The code supports periodic checkpoints; these are intended
to speed up reintegration into teh flock after a node restart. Checkpoints are a work in progress
and not yet tested. Probe simply returns a random address from the current address range (which
is currently set outside this interface).

TBD: there needs to be a flocking level function to list nodes in the cluster. this is currently
being worked, but the answer is likely to be a reeve function.
*/
package flock

import (
	"time"

	"github.com/erixzone/crux/pkg/crux"
)

// Key represents the symmetric key used for encoding packets
type Key [32]byte

// NodeID identifies a node. Must be unique across flock.
type NodeID struct {
	Moniker string
	Addr    string
	Horde	string
}

// Info is the data packet sent for heartbeats and probes. in the fullness of time,
// it will be supplemented with member lists and registries.
// if vote==0, then this is an admin command.
// the Beacon is the way out of the cluster; the exact protocol will be named later on.
type Info struct {
	Flock       string // name of my flock
	Dest        NodeID
	Me          NodeID // src TBD
	Leader      NodeID
	Lvote       int32
	Vote        int32
	Expire      time.Time
	EpochID     Nonce
	Beacon      string
	Steward     string
	Registry    string
	RegistryKey string
}

// Fnode is the interface for node-specific functions.
type Fnode interface {
	Recv() *Info                             // receive a packet sent to me (via SetMe)
	Send(*Info)                              // send a FlockInfo
	SetMe(*NodeID)                           // fill in unset fields
	SetKeys(epochID *Nonce, sec0, sec1 *Key) // use nil for unused
	Logf(string, ...interface{})             // how we log stuff
	GetCertificate() *crux.TLSCert           // cert chain for TLS
	Monitor() chan crux.MonInfo              // monitoring channel; nil is off
	Quit()                                   // for shutting down
}

// Fflock is the interface for flock-wide functions. It is especially important that the timing functions
// give identical answers across the flock.
type Fflock interface {
	Heartbeat() time.Duration    // period for sending heartbeats
	KeyPeriod() time.Duration    // period for rotating prim0
	NodePrune() time.Duration    // lose flock membership if no heartbeats
	HistoryPrune() time.Duration // forget old nodes after this
	Checkpoint() time.Duration   // drop a checkpoint after this period
	Probebeat() time.Duration    // period between rework probes
	ProbeN() int                 // how many probes per rework
	Probe() (string, bool)       // return a random address within a given range
}

// Status is used for reporting overall flock status
type Status struct {
	T      time.Time
	Period time.Duration
	Stable bool
	Name   string
	N      int
}
