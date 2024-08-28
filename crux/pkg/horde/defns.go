package horde

import (
	"time"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/kv"
)

// horde service-related timing constants
const (
	HeartbeatPeriod = 10 // HeartbeatPeriod is the recommended period for heartbeat
	HeartbeatMargin = 5  // HeartbeatMargin is the recommendd safety margin
	// the horde will consider a service to be dead after heartbeat.t + (HeartbeatPeriod+HeartbeatMargin) secs
)

/*

// Horder is the interface to a distributed process group.
type Horder interface {
	UniqueID() string    // unique identifier for this horde
	Description() string // horde description
	Board() Boarder      // underlying blackboard
	SetSpec(spec string) // set the string retuned by Spec()
	Spec() string        // return a command-line flag that will create this type of horde
	// node stuff
	RegisterNode(name string, tags []string) error // register a node
	Nodes() ([]Node, error)                        // get a list of current nodes
	// service stuff
	RegisterService(uuid, role, node, addr, pkey string) (string, error) // register a service; return uuid
	HeartbeatService(uuid string, stage StageStr) error                  // heartbeat a service; set stage
	Stage(uuid string) StageStr                                          // returns the stage for this service
	Services() ([]Service, error)                                        // get a list of current services
	// governor stuff
	Govern() Governor                                        // underlying process governor
	RegisterGov(node, ip, hostname, addr, pkey string) error // register a governor
	Govs() []Gov                                             // get a list of current governors
	// these next methods proxy the underlying Boarder methods applying the horde-related prefix
	List(prefix string) ([]string, error) // return list of keys with prefix (use "" for none)
	Get(key string) (string, error)       // return value for key
	Put(key, value string) error          // put value
	Delete(key string) error              // delete key
	CAS(key, ovalue, nvalue string) error // swap values
	Leader() string                       // leader
}

// Boarder is the interface to a distributed blackboard. (think consul, etcd or gyre)
type Boarder interface {
	Watcher(chatty bool, output chan string) error // monitor blackboard events to output
	List(prefix string) ([]string, error)          // return list of keys with prefix (use "" for none)
	Get(key string) (string, error)                // return value for key
	Put(key, value string) error                   // put value
	Delete(key string) error                       // delete key
	CAS(key, ovalue, nvalue string) error          // swap values
	Leader() string                                // leader
}

// Governor is the interface to process creation/destruction
type Governor interface {
	Service2cmd(uuid, node, ip, service string) string // map a service name (like segp) into a real command invocation
	Start(node, service string, count int)             // start count instances of the service on the given node
	Stop(node, service string, count int)              // stop count instances of the service on the given node
}
*/

// Administer is the administrative interface for the horde, and for the nodes in the horde.
type Administer interface {
	// horde stuff
	UniqueID() string    // unique identifier for this horde
	Description() string // horde description
	// node stuff
	RegisterNode(name string, tags []string) *crux.Err // register a node
	Nodes() ([]Node, *crux.Err)                        // get a list of current nodes
}

// Servicer is the interface for dealing with Services.
type Servicer interface {
	// service stuff
	RegisterService(uuid, role, node, addr, pkey string) (string, *crux.Err) // register a service; return uuid
	HeartbeatService(uuid string, stage StageStr) *crux.Err                  // heartbeat a service; set stage
	Stage(uuid string) StageStr                                              // returns the stage for this service
	Services() ([]Service, *crux.Err)                                        // get a list of current services
}

// Action captures what you need to start and stop services.
type Action interface {
	Start(node, service string, count int)
	Stop(node, service string, count int)
	What() []Service
	Start1(node, service, addr string) // mainly for testing
	Reset()
}

// Horde struct makes it convenient to hold a bundle of individual interfaces.
type Horde struct {
	Adm Administer
	Act Action
	KV  kv.KV
}

// structs supporting above interfaces

// Service describes an instance of a service.
type Service struct {
	UniqueID string    // unique identifier
	Name     string    // service identifier
	Node     string    // node where it runs
	Addr     string    // address suitable for client/server routines
	Key      []byte    // public key for client/server
	Delete   bool      // pending delete (if not, add)
	Expire   time.Time // when this service entry expires
	Stage    StageStr  // what stage the service thinks it's at
}

// StageStr encourages use of a limited set of stages
type StageStr string

// values for StageStr
const (
	StageStart   StageStr = "start"
	StageReady   StageStr = "ready"
	StageDone    StageStr = "done"
	StageUnknown StageStr = "unknown"
)

// Node describes a node and its tags
type Node struct {
	Name string // unique within a horde
	Tags []string
}

// Gov describes a registered governor.
type Gov struct {
	UniqueID string // unique identifier
	Node     string // given node name
	IP       string // ip address
	Hostname string // symbolic name for the IP
	Addr     string // dialstring for zmq.NewServer
	Key      []byte // public key for server
}

// KhanWho is where khan publishes its list of what it's trying to start
const KhanWho = "khan/who"
