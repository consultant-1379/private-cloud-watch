// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package registrydb

import (
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
)

// EndpointRow - a DB row for an Endpoint
type EndpointRow struct {
	EndpointUUID string
	BlocName     string
	HordeName    string
	NodeName     string
	ServiceName  string
	ServiceAPI   string
	ServiceRev   string
	Principal    string
	Address      string
	StartedTS    string
	LastTS       string
	ExpiresTS    string
	RemovedTS    string
	Status       string
}

// ClientRow - A DB row for Client
type ClientRow struct {
	ClientUUID  string
	BlocName    string
	HordeName   string
	ServiceName string
	ServiceAPI  string
	ServiceRev  string
	Principal   string
	KeyID       string
	PubKey      string
	StartedTS   string
	LastTS      string
	ExpiresTS   string
	RemovedTS   string
	Status      string
}

// AllowedRow - A DB row for whitelist allowed info
type AllowedRow struct {
	RuleUUID  string
	RuleID    int
	EPGroupID string
	CLGroupID string
	OwnerID   string
}

// StateTimeRow - A DB row attaching time windows to State Numbers
type StateTimeRow struct {
	StateNo int
	StartTS string
	EndTS   string
}

// BadRequestsRow - a DB row recording errors that arise
// Does not include the offending request data - assumes that gets logged with requestuuid.
// Don't want unvetted crap hitting the database.
// KeyID is that of the reeve or whatever called with the request
type BadRequestsRow struct {
	RequestUUID string
	KeyID       string
	StateNo     int
	RequestTS   string
	error       string
}

// EntryStateRow - A DB row recording Endpoint or Client States
type EntryStateRow struct {
	EntryNo   int
	EntryUUID string
	RuleID    string
	AddState  int
	DelState  int
	CurState  int
}

// FB is a flag block, provides a method for interrogating struct types
// in heterogeneous lists without hitting reflection
type FB interface {
	BMe() byte
}

// AmCLIENT - Label for our list struct types
const AmCLIENT byte = 0x80

// AmENDPOINT - Label for our list struct types
const AmENDPOINT byte = 0x02

// ClUpdate - update from pb.ClientInfo, plus assigned a transaction uuid
type ClUpdate struct {
	TxUUID        string // transaction uuid
	ReeveKeyID    string // updater
	pb.ClientData        // Nodeid, Keyid, Keyjson, Status (KeyStatus)
}

// BMe - retrieves type byte
func (c *ClUpdate) BMe() byte {
	return AmCLIENT
}

// EpUpdate - update from pb.EndpointInfo, plus assigned a transaction uuid
type EpUpdate struct {
	TxUUID          string // transaction uuid
	ReeveKeyID      string // updater
	pb.EndpointData        // Nodeid, Netid, Status, hash (ServiceState)
}

// BMe - retrieves type byte
func (c *EpUpdate) BMe() byte {
	return AmENDPOINT
}

// Because Go ism - sets up the FB interface to both types
var _ FB = (*ClUpdate)(nil)
var _ FB = (*EpUpdate)(nil)

// StateClock - a batch of events bounded by timestamps bounding when they were recieved by Steward
type StateClock struct {
	State int32
	Begin time.Time
	End   time.Time
}
