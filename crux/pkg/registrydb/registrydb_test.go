package registrydb

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	// _ sqlite required
	_ "github.com/mattn/go-sqlite3"
	"github.com/nats-io/nuid"
	"github.com/pborman/uuid"
	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/grpcsig"
)

func TestRegistryDBtest(t *testing.T) { TestingT(t) }

type RegistryDBtestSuite struct {
	dir1 string
	dir2 string
}

func init() {
	Suite(&RegistryDBtestSuite{})
}

func (p *RegistryDBtestSuite) SetUpSuite(c *C) {
	fmt.Printf("Setting up...\n")
	// p.dir1 = "."
	p.dir1 = c.MkDir()
	kerr := grpcsig.SSHKeygenExists()
	c.Assert(kerr, IsNil)
}

func (p *RegistryDBtestSuite) TearDownSuite(c *C) {
	fmt.Printf("Teardown done.\n")
}

func fakenode(i int, portstr string, servicename string, servicerev string) EndpointRow {
	nodenum := i
	nodestr := "node" + strconv.Itoa(nodenum)
	nowts := fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	ep := EndpointRow{
		EndpointUUID: uuid.NewUUID().String(),
		BlocName:     "flock1",
		HordeName:    "horde1",
		NodeName:     nodestr,
		ServiceName:  servicename,
		ServiceAPI:   servicename + "0",
		ServiceRev:   servicerev,
		Principal:    nuid.Next(),
		Address:      nodestr + ":" + portstr,
		StartedTS:    nowts,
		LastTS:       nowts,
		Status:       "up",
	}
	return ep
}

func fakeclient(c *C, servicename string, servicerev string) ClientRow {
	nowts := fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	cl := ClientRow{
		ClientUUID:  uuid.NewUUID().String(),
		BlocName:    "flock1",
		HordeName:   "horde1",
		ServiceName: servicename,
		ServiceAPI:  servicename + "0",
		ServiceRev:  servicerev,
		Principal:   nuid.Next(),
		StartedTS:   nowts,
		LastTS:      nowts,
		Status:      "active",
	}
	pk, err := grpcsig.NewKeyPair("", "junkkey", "", cl.Principal, true)
	fmt.Printf("%v\n", err)
	c.Assert(err, IsNil)
	pk.Service = cl.ServiceRev
	fp := pk.KeyID // raw fingerprint
	pk.KeyID = fmt.Sprintf("/%s/%s/keys/%s", cl.ServiceRev, cl.Principal, fp)
	pkjson, jerr := grpcsig.PubKeyToJSON(pk)
	cl.PubKey = pkjson
	c.Assert(jerr, IsNil)
	cl.KeyID = pk.KeyID
	os.Remove("junkkey")
	os.Remove("junkkey.pub")
	return cl
}

func (p *RegistryDBtestSuite) TestRegistryDB(c *C) {
	fmt.Printf("\nInitialize RegistryDB\n")
	err := InitializeRegistryDB(p.dir1+"/"+"testregistry.db", true)
	c.Assert(err, IsNil)

	endpoints := []EndpointRow{}
	for i := 0; i < 10; i++ {
		ep := fakenode(i, "23456", "Bubbles", "Bubbles1_0_1")
		fmt.Printf("%v\n", ep)
		endpoints = append(endpoints, ep)
	}

	clients := []ClientRow{}
	for j := 0; j < 10; j++ {
		cl := fakeclient(c, "Bubbles", "Bubbles1_0_ 1")
		fmt.Printf("%v\n", cl)
		clients = append(clients, cl)
	}

	// Try some inserts

	db, derr := sql.Open("sqlite3", p.dir1+"/"+"testregistry.db")
	c.Assert(derr, IsNil)
	defer db.Close()

	fmt.Printf("Adding Endpoints\n")

	tx, derr := db.Begin()
	c.Assert(derr, IsNil)
	stmt, derr := tx.Prepare("insert into endpoint(endpointuuid, blocname, hordename, nodename, servicename, serviceapi, servicerev, principal, address, startedts, lastts, status) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? )")
	c.Assert(derr, IsNil)
	defer stmt.Close()
	for _, endpt := range endpoints {
		_, derr = stmt.Exec(endpt.EndpointUUID,
			endpt.BlocName,
			endpt.HordeName,
			endpt.NodeName,
			endpt.ServiceName,
			endpt.ServiceAPI,
			endpt.ServiceRev,
			endpt.Principal,
			endpt.Address,
			endpt.StartedTS,
			endpt.LastTS,
			endpt.Status)
		c.Assert(derr, IsNil)
	}
	txerr := tx.Commit()
	if txerr != nil {
		fmt.Printf("txerr: %v", txerr)
	}

	fmt.Printf("Adding Clients\n")
	tx, derr = db.Begin()
	c.Assert(derr, IsNil)
	stmt, derr = tx.Prepare("insert into client(clientuuid, blocname, hordename, servicename, serviceapi, servicerev, principal, keyid, pubkey, startedts, lastts, status) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	c.Assert(derr, IsNil)
	defer stmt.Close()
	for _, cli := range clients {
		_, derr = stmt.Exec(cli.ClientUUID,
			cli.BlocName,
			cli.HordeName,
			cli.ServiceName,
			cli.ServiceAPI,
			cli.ServiceRev,
			cli.Principal,
			cli.KeyID,
			cli.PubKey,
			cli.StartedTS,
			cli.LastTS,
			cli.Status)
		c.Assert(derr, IsNil)
	}
	txerr = tx.Commit()
	c.Assert(txerr, IsNil)

	rows, derr := db.Query("select clientuuid, principal, keyid from client")
	c.Assert(derr, IsNil)
	for rows.Next() {
		var clientuuid string
		var userid string
		var keyid string
		derr = rows.Scan(&clientuuid, &userid, &keyid)
		c.Assert(derr, IsNil)
		fmt.Println(clientuuid, userid, keyid)
	}
	derr = rows.Err()
	c.Assert(derr, IsNil)
	rows.Close()

	rows, derr = db.Query("select endpointuuid, principal, address from endpoint where servicename = 'Bubbles'")
	c.Assert(derr, IsNil)
	defer rows.Close()
	for rows.Next() {
		var endpointuuid string
		var principal string
		var address string
		derr = rows.Scan(&endpointuuid, &principal, &address)
		c.Assert(derr, IsNil)
		fmt.Println(endpointuuid, principal, address)
	}
	derr = rows.Err()
	c.Assert(derr, IsNil)

	ruleuuid, rerr := MakeServiceGroup(db, "Bubbles", "Bubbles", "horde1", "7", "admin")
	fmt.Printf("MakeBubblesGroup - Errors? [%v], ruleuuid= %s\n", rerr, ruleuuid)
	c.Assert(rerr, IsNil)

	fmt.Printf("\nDone RegistryDB Internals\n")
}
