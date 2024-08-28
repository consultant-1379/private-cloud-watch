package lib

import (
	"time"

	"github.com/erixzone/crux/pkg/begat/common"
)

// EventControl is the control message
type EventControl struct {
	T         time.Time
	Op        ECOp
	Progress  bool
	MustBuild bool
	Return    chan EventControl
}

// FSRouterCmd is the control record for the FS router
type FSRouterCmd struct {
	Op    FSRouterOp
	Dest  chan<- EventFS
	Files []string
	ID    string
}

// EventFS is the filesystem event
// for FSEexecstatus, the runID is in Path and the error status is in Err
type EventFS struct {
	Op   FSEType
	Path string
	Hash common.Hash
	Err  string
}

// EventStatus is something.
type EventStatus struct {
	T      time.Time
	ID     string
	Status StatusType
	Err    string
}

// Mount is a mount spec
type Mount struct {
	Where string
	What  string
	Args  []string
}

// Chore is an executable quantum.
type Chore struct {
	Status       StatusType
	Globals      *map[string]Variable
	RunID        string
	D            *Dictum
	Dir          string
	Nexec        int // number of times we have executed this recipe
	Tbegin, Tend time.Time
	Sig          common.Hash
	HistID       string
	Cacheable    bool
	Stepped      bool
	Stored       bool
	Mounts       []Mount
	InEnts       []Ent // for stuff where we know the hash
	OutEnts      []Ent
	RunEnts      []Ent // from when we last executed
	Ctl          chan EventControl
	ret          chan EventControl
}

// Travail is an executed Chore
type Travail struct {
	Chore
}

/*
	the BI (Begat Interface) interfaces detail how begat talks to the outside world.
each interface captures the interactions for that area.
*/

// BIfs is an EventFS pub/sub system
type BIfs interface {
	Pub(string, <-chan EventFS) // register an EventFS publisher with a label on the given channel
	PubClose(<-chan EventFS)    // close that publisher
	Sub(FSRouterCmd)            // manipulate a subscriber
	Quit()                      // bye bye
}

// BIexec is how begat talks to execution points
type BIexec interface {
	InitN(int) // init n nodes
	//RouterCtl(FSRouterCmd) // send a command to the internal router
	PrimeFS(*Chore)    // prime the EventFS pump
	Exec(*Chore) error // exec one chore somewhere
	Quit()             // bye bye
}

// BIhistory talks to the history database
type BIhistory interface {
	Clear()                              // clear the history
	GetTravail(*Chore) (*Travail, error) // get a travail using the chore as a key
	PutTravail(*Travail) error           // store a travail (using its internal chore as a key)
	Quit()                               // bye bye
}
