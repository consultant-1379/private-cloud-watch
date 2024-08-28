package steward

// This tests the steward server with the self-key system
// within a single process

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nuid"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
)

func TestStewardtest(t *testing.T) { TestingT(t) }

type StewardtestSuite struct {
	dir string
}

func init() {
	Suite(&StewardtestSuite{})
}

const (
	port1i   = 50065
	port1    = ":50065"
	dbname   = "whitelist.db"
	service1 = StewardRev
)

var testdb string

var imp1 grpcsig.ImplementationT

func (p *StewardtestSuite) SetUpSuite(c *C) {
	fmt.Printf("Setting up...\n")
	p.dir = c.MkDir()
	// p.dir = "."
	serr := grpcsig.SSHKeygenExists()
	c.Assert(serr, IsNil)
	fmt.Printf("Starting Muck\n")
	merr := muck.InitMuck(p.dir+"/.muck", "")
	fmt.Printf("InitMuck() Errors?: [%v]\n", merr)
	c.Assert(merr, IsNil)

	fmt.Printf("\nStarting Self Keys\n")

	//-------------------
	// Bootstrap "self-keys" for this process, so it can authenticate itself
	// with ssh-keygen, and start ssh-agent for signing
	kerr := grpcsig.InitSelfSSHKeys(true)
	fmt.Printf("InitSelfSSHKeys() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// Double check that we can read keys from ssh-agent
	_, kerr = grpcsig.ListKeysFromAgent(true) // Does ssh-agent have it?
	fmt.Printf("ListKeysFromAgent() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// A fulcrum will use one whitelist DB per process - for all services
	// Provide our test public key lookup BoltDB (initally empty) as
	testdb = p.dir + "/.muck/" + dbname

	oerr := grpcsig.StartNewPubKeyDB(testdb)
	c.Assert(oerr, IsNil)
	logger := clog.Log.With("focus", "steward_test")
	derr := grpcsig.InitPubKeyLookup(testdb, logger)
	c.Assert(derr, IsNil)

	// Add self public key to whitelist DB (i.e. this test talks to itself!)
	kerr = grpcsig.AddPubKeyToDB(grpcsig.GetSelfPubKey())
	fmt.Printf("AddNewPubKeyToDB() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	grpcsig.FiniPubKeyLookup() //

	//------------------
	// START steward demo server 1 "stewardsrv"

	fmt.Printf("\nStarting steward server %s\n", service1)
	var cerr *crux.Err

	// The Steward server is grpcsig protected (from above, we have a self-key in the db)
	// don't watch
	imp1, cerr = grpcsig.InitDefaultService(testdb, service1, nil, logger, 300, false)
	if cerr != nil {
		fmt.Printf("InitDefaultService() Errors?: [%v]\n", cerr)
	}
	c.Assert(cerr, IsNil)
	impif := &imp1
	nod, ferr := idutils.NewNodeID("flock", "horde", "node", StewardName, StewardAPI)
	if ferr != nil {
		fmt.Printf("NewNodeID() Errors?: [%v]\n", ferr)
	}
	c.Assert(ferr, IsNil)

	nid, nerr := idutils.NewNetID(StewardRev, "principal", "localhost", port1i)
	if nerr != nil {
		fmt.Printf("NewNetID() Errors?: [%v]\n", nerr)
	}
	c.Assert(nerr, IsNil)

	zerr := StartSteward(nod, nid, &impif)
	if zerr != nil {
		fmt.Printf("StartSteward() Errors?: [%v]\n", zerr)
	}
	c.Assert(zerr, IsNil)

	/*
		// Start up the Steward database and Ingestor, clear the database if junk left in it
		zerr := StartStewardDB(p.dir+"/.muck/steward.db", logger, true)
		if zerr != nil {
			fmt.Printf("StartStewardDatabase() Errors?: [%v]\n", zerr)
		}
		c.Assert(zerr, IsNil)

		// Start up the Steward service
		serr = StartStewardServer(imp1, port1)
		if serr != nil {
			fmt.Printf("StartStewardServer() Errors?: [%v]\n", serr)
		}
		c.Assert(serr, IsNil)

		fmt.Printf("Serving '%s' on port%s\n", service1, port1)
	*/
	// Pause for handy MacOS X firewall dialogue to appear & clicky-clicky
	if runtime.GOOS == "darwin" {
		time.Sleep(4 * time.Second)
	} else {
		time.Sleep(4 * time.Millisecond)
	}
}

func (p *StewardtestSuite) TearDownSuite(c *C) {
	fmt.Printf("\nTearing Down Suite\n")

	// Remove that key from "pubkeys.db", even though it is a transient /tmp dir..
	derr := grpcsig.RemoveSelfPubKeysFromDB()
	fmt.Printf("RemoveSelfPubKeysFromDB() %s Errors?: [%v]\n", testdb, derr)
	c.Assert(derr, IsNil)

	// Remove the private key from ssh-agent
	err := grpcsig.FiniSelfSSHKeys(true)
	fmt.Printf("FiniSelfSSHKeys() Errors?: [%v]\n", err)
	c.Assert(err, IsNil)

	// Stop the steward DB and Ingestor
	StopStewardDB(true)

	// Stop the lookup service
	grpcsig.FiniDefaultService()

	fmt.Printf("Teardown done.\n")
}

func (p *StewardtestSuite) TestStewardClient(c *C) {

	// Start up client
	fmt.Printf("\nStarting Steward Client 1\n")

	// It is going to talk to the Steward server with the self-key signature...
	selfSigner, err := grpcsig.SelfSigner(imp1.Certificate)
	if err != nil {
		fmt.Printf("grpcsig.SelfSigner() Errors?: [%v]\n", err)
	}
	c.Assert(err, IsNil)

	conn1, err := selfSigner.Dial(port1)
	defer conn1.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Did not connect to server1: %v\n", err)
	}
	c.Assert(err, IsNil)
	stewardcli := pb.NewStewardClient(conn1)

	// PingSleep until server is up
	cerr := PingSleep(stewardcli, 2*time.Second, 300*time.Second)
	c.Assert(cerr, IsNil)

	// Make Fake Clients/Endpoints

	loopcount := 40

	// Make a pile of clients and endpoints and send them to server
	for i := 0; i < loopcount; i++ {
		cli, err := fakeClient(p.dir, "reeve", "reeve1.0", false, false)
		c.Assert(err, IsNil)
		ack, err := stewardcli.ClientUpdate(context.Background(), cli)
		c.Assert(err, IsNil)
		fmt.Printf("CLI Ack:%v\n", ack)
		ep := fakeEndpoint(i, ":42069", "reeve", "reeve1.0", false)
		ack, err = stewardcli.EndpointUpdate(context.Background(), ep)
		c.Assert(err, IsNil)
		fmt.Printf("EP Ack:%v\n", ack)
		// send some bad ones
		if i%10 == 0 {
			fmt.Println("BAD ONES INJECTED")
			cli, err := fakeClient(p.dir, "reeve", "reeve1.0", false, true)
			c.Assert(err, IsNil)
			ack, err := stewardcli.ClientUpdate(context.Background(), cli)
			c.Assert(err, IsNil)
			fmt.Printf("CLI Ack:%v\n", ack)
			ep := fakeEndpoint(i, ":42069", "reeve", "reeve1.0", true)
			ack, err = stewardcli.EndpointUpdate(context.Background(), ep)
			c.Assert(err, IsNil)
			fmt.Printf("EP Ack:%v\n", ack)
		}
	}

	time.Sleep(6 * time.Second) // Give server some time to catch up with the buffer..
	fmt.Printf("Done!\n")
}

func fakeClient(path string, servicename string, servicerev string, debug bool, bad bool) (*pb.ClientData, error) {
	principalID := nuid.Next()
	pk, err := grpcsig.NewKeyPair("rsa", path+"/junkkey", "", principalID, false)
	if err != nil {
		return nil, err
	}
	pk.Service = servicename
	fp := pk.KeyID // raw fingerprint
	pk.KeyID = fmt.Sprintf("/%s/%s/keys/%s", servicerev, principalID, fp)
	pkjson, jerr := grpcsig.PubKeyToJSON(pk)
	if jerr != nil {
		return nil, err
	}
	status := pb.KeyStatus_CURRENT
	ci := pb.ClientData{}
	if !bad {
		ci = pb.ClientData{
			Nodeid:  "////reeve/reeve1",
			Keyid:   pk.KeyID,
			Keyjson: pkjson,
			Status:  status,
		}
	} else {
		ci = pb.ClientData{
			Nodeid:  "/BAD///",
			Keyid:   pk.KeyID,
			Keyjson: pkjson,
			Status:  status,
		}
	}
	os.Remove(path + "/junkkey")
	os.Remove(path + "/junkkey.pub")
	return &ci, nil
}

func fakeEndpoint(nodenum int, portstr string, servicename string, servicerev string, bad bool) *pb.EndpointData {
	principalID := nuid.Next()
	status := pb.ServiceState_UP
	nodeid := "/flock1/horde1/node" + strconv.Itoa(nodenum) + "/" + servicename + "/" + "reeve1"
	netid := ""
	if !bad {
		netid = "/" + servicerev + "/" + principalID + "/net/" + "localhost" + portstr
	} else {
		netid = "/" + servicerev + "/" + principalID + "/BAD/" + "localhost" + portstr
	}
	ep := pb.EndpointData{
		Nodeid: nodeid,
		Netid:  netid,
		Status: status,
	}
	return &ep
}
