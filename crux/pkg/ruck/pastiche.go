package ruck

import (
	"fmt"
	"os"

	pb "github.com/erixzone/crux/gen/cruxgen"
	rpb "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/pastiche"
	//"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// Pastiche  is the current versioned name
// TODO: This should move to pkg/pastiche/server.go
const (
//	PasticheName = "pastiche"
//	PasticheAPI  = "pastiche1"
//	PasticheRev  = "pastiche0_2"
)

/* func Dean1_0(
quit <-chan bool,
alive chan<- []pb.HeartbeatReq,
network **crux.Confab,
UUID string,
logger clog.Logger,
nod idutils.NodeIDT,
eRev string,
reeveapi *reeve.StateT) idutils.NetIDT {

*/

// TODO:  Dean needs eRev, does pastiche?
// TODO: nod

// Pastiche0_1 is this release of pastiche.
// Should be started in a new goroutine
func Pastiche0_1(quit <-chan bool, alive chan<- []pb.HeartbeatReq, network **crux.Confab, UUID string, logger clog.Logger, nod idutils.NodeIDT, eRev string, reeveapi *reeve.StateT) idutils.NetIDT {
	// TODO: remove comment.  This is what Dean call looks like.
	// myNID = Dean1_0(nil, heartChan, &cc, "uuid", logboot.With("func", DeanRev), dNOD, DeanRev, reeveapi) // start it

	// Ugh. Bundling is a waste.  All this stuff is just pulled
	// out as args to crux generated grpc code.  There should just
	// be an "api block" struct with the version numbers and use the
	// existing ReeveBlock as-is.
	//myrb := RB(rb.blocName, rb.hordeName, rb.nodeName, PasticheName, PasticheAPI, rb.eRev, rb.reeveapi)

	nod = ReNOD(nod, pastiche.PasticheName, pastiche.PasticheAPI)
	// Setup dirs to use via muck - This will likely change
	blobdir := muck.BlobDir()
	var dirs []string
	clog.Log.With(nil, ">>> Starting PASTICHE log")
	fmt.Printf(">>> Starting PASTICHE printf\n")

	if _, err := os.Stat(blobdir); os.IsNotExist(err) {
		clog.Log.With(nil, fmt.Sprintf("server creations returned  err=%v\n", err))
		crux.Exit(1)
	}
	dirs = append(dirs, blobdir)
	clog.Log.With(nil, fmt.Sprintf("Creating pastiche using dir :%s\n", blobdir))

	pSrv, err := pastiche.NewServer(dirs)

	if err != nil {
		clog.Log.With(nil, fmt.Sprintf("server creations returned  err=%v\n", err))
		crux.Exit(1)
	}

	// Heartbeats for Crux healthcheck

	nid := rpb.StartPasticheSrvServer(&nod, pastiche.PasticheRev, nod.NodeName, 0, pSrv, quit, reeveapi)
	go Afib(alive, quit, UUID, "", nid)
	clog.Log.With(nil, "pastiche server returned / exited")
	fmt.Printf(">>> PASTICHE EXITED printf\n")

	return nid
}
