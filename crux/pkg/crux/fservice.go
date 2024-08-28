package crux

import (
	"time"

	pb "github.com/erixzone/crux/gen/cruxgen"
)

// Fservice is used for reporting available services.
// when reporting across nodes, FuncName is the url (ip:port) and Image is any key stuff.
type Fservice struct {
	UUID     string                   // primary (per-service unique)
	FuncName string                   // name of function to be invoked
	FileName string                   // key of file containing the func
	Alive    chan<- []pb.HeartbeatReq // health updates
	Quit     chan bool                // true means quit
	T        time.Time                // either "valid until T" or "time of ??"
}

// FserviceReturn is used for reporting back what an Fservice starts
type FserviceReturn struct {
	Err chan *Err
}

// ClientAddr will need to get much better
func ClientAddr(name string, period time.Duration) (string, *Err) {
	return "goo", nil
}

// Timestamp2Time converts the protobuf Timestamp to a Time
func Timestamp2Time(ts *pb.Timestamp) time.Time {
	if ts == nil {
		var t time.Time
		return t
	}
	return time.Unix(0, ts.Nano)
}

// Time2Timestamp converts Time to the protobuf Timestamp
func Time2Timestamp(t time.Time) *pb.Timestamp {
	return &pb.Timestamp{Unix: t.Unix(), Nano: t.UnixNano()}
}
