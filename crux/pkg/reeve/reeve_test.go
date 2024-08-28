package reeve

// This tests the reeve client and server with the self-key system
// within a single process

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/grpcsig"
)

func TestSigtest(t *testing.T) { TestingT(t) }

type SigtestSuite struct {
	dir string
}

func init() {
	Suite(&SigtestSuite{})
}

const (
	port1    = ":50055"
	dbname   = "pubkeys.db"
	service1 = "reevesrv"
)

var testdb string

var imp1 grpcsig.ImplementationT

func (p *SigtestSuite) SetUpSuite(c *C) {
	p.dir = c.MkDir()
	// p.dir = "./"
	fmt.Printf("Starting Reeve Server\n")
	logger := clog.Log.With("focus", "reeve_test")
	reevestate, err := StartReeve(p.dir, "flock", "horde", "node", 50055, "", nil, logger)
	c.Assert(err, IsNil)
	fmt.Printf("Reeve Server Started\n")
	imp1 = *reevestate.imp
}

func (p *SigtestSuite) TearDownSuite(c *C) {
	fmt.Printf("\nTearing Down Suite\n")
	Fini()
	fmt.Printf("Teardown done.\n")
}

func (p *SigtestSuite) TestReeveEndpointStore(c *C) {
	fmt.Printf("Testing Endpoint Storage\n")
	eerr := endpointsLocalIni()
	c.Assert(eerr, IsNil)
	ReeveEvents = startIngest()
	testEp := pb.EndpointInfo{}

	// Pending, then success
	testEp.Filename = "good"
	EndpointPending(testEp, "abc")
	// fmt.Printf("eps.completed: [%v]\n,eps.pending [%v]\n,eps.failed [%v]\n", eps.completed, eps.pending, eps.failed)
	EndpointCompleted(testEp, "abc", "xyz")
	// fmt.Printf("eps.completed: [%v]\n,eps.pending [%v]\n,eps.failed [%v]\n", eps.completed, eps.pending, eps.failed)

	// Event Loop
	epd := pb.EndpointData{}
	epd.Hash = "hash"
	ReeveEvents.IngestEndpoint(&epd)

	ReeveEvents.Quit()

	// Pending, then fail
	testEp.Filename = "bad"
	EndpointPending(testEp, "efg")
	// fmt.Printf("eps.completed: [%v]\n,eps.pending [%v]\n,eps.failed [%v]\n", eps.completed, eps.pending, eps.failed)
	EndpointFailed(testEp, "efg", "jkl")
	// fmt.Printf("eps.completed: [%v]\n,eps.pending [%v]\n,eps.failed [%v]\n", eps.completed, eps.pending, eps.failed)

	fmt.Printf("Save Endpoints\n")
	serr := SaveEndpoints()
	c.Assert(serr, IsNil)
	fmt.Printf("Clear Endpoints in Memory And Reload from storage\n")
	lerr := endpointsLocalIni()
	// fmt.Printf("eps.completed: [%v]\n,eps.pending [%v]\n,eps.failed [%v]\n", eps.completed, eps.pending, eps.failed)
	c.Assert(lerr, IsNil)
}

func (p *SigtestSuite) TestReeveClientStore(c *C) {
	fmt.Printf("Testing Client Storage\n")
	cerr := clientsLocalIni()
	ReeveEvents = startIngest()
	c.Assert(cerr, IsNil)

	//fmt.Printf("cls.completed: [%v]\n,cls.pending [%v]\n,cls.failed [%v]\n", cls.completed, cls.pending, cls.failed)

	testCl := pb.ClientInfo{}

	// Pending, then success
	testCl.Nodeid = "/flock/horde/me/service1/serviceapi1"
	ClientPending(testCl, "abc")
	//fmt.Printf("cls.completed: [%v]\n,cls.pending [%v]\n,cls.failed [%v]\n", cls.completed, cls.pending, cls.failed)
	ClientCompleted(testCl, "abc", "wxyz")
	//fmt.Printf("cls.completed: [%v]\n,cls.pending [%v]\n,cls.failed [%v]\n", cls.completed, cls.pending, cls.failed)

	// Event Loop
	cld := pb.ClientData{}
	cld.Nodeid = "/flock/horde/me/service1/serviceapi1"
	ReeveEvents.IngestClient(&cld)

	ReeveEvents.Quit()

	// Pending, then fail
	testCl.Nodeid = "bad"
	ClientPending(testCl, "fgh")
	//fmt.Printf("cls.completed: [%v]\n,cls.pending [%v]\n,cls.failed [%v]\n", cls.completed, cls.pending, cls.failed)
	ClientFailed(testCl, "fgh", "ijkl")
	//fmt.Printf("cls.completed: [%v]\n,cls.pending [%v]\n,cls.failed [%v]\n", cls.completed, cls.pending, cls.failed)

	fmt.Printf("Save Clients\n")
	serr := SaveClients()
	c.Assert(serr, IsNil)
	fmt.Printf("Clear Clients in Memory and Reload Clients from Storage\n")
	lerr := clientsLocalIni()
	//fmt.Printf("cls.completed: [%v]\n,cls.pending [%v]\n,cls.failed [%v]\n", cls.completed, cls.pending, cls.failed)
	c.Assert(lerr, IsNil)

}

func (p *SigtestSuite) TestReeveClient(c *C) {

	// Start up client
	fmt.Printf("\nStarting Reeve Client 1\n")

	// It is going to talk to the reeve server with the self-key signature...
	selfSigner, err := grpcsig.SelfSigner(imp1.Certificate)
	if err != nil {
		fmt.Printf("grpcsig.SelfSigner() Errors?: [%v]\n", err)
	}
	c.Assert(err, IsNil)

	// mess with the implementation to trigger errors
	junkimp := imp1
	junkimp.Algorithms = []string{"bobscrypto", "nullcrypto"}

	// Now - This is a one-shot (dial, call gRPC getReeveUpdate(), close)
	// encapsulated call to the Reeve Server.
	cerr2 := ClientUpdate(port1, grpcsig.GetSelfPubKey(), selfSigner, &junkimp)
	if cerr2 != nil {
		fmt.Printf("ClientUpdate() Errors?: [%v]\n", cerr2)
	}
	c.Assert(cerr2, Not(IsNil))

	fmt.Printf("\nStarting Reeve Client 2\n")

	// Now - This is a one-shot (dial, call gRPC getReeveUpdate(), close)
	// encapsulated call to the Reeve Server.
	cerr := ClientUpdate(port1, grpcsig.GetSelfPubKey(), selfSigner, &imp1)
	if cerr != nil {
		fmt.Printf("ClientUpdate() Errors?: [%v]\n", cerr)
	}
	c.Assert(cerr, IsNil)

}
