package registrydb

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	//	"testing"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

// func TestFanouttest(t *testing.T) { TestingT(t) }

type FanouttestSuite struct {
	dir1 string
	dir2 string
}

func init() {
	Suite(&FanouttestSuite{})
}

func (p *FanouttestSuite) SetUpSuite(c *C) {
	fmt.Printf("Setting up...\n")
	// p.dir1 = "."
	p.dir1 = c.MkDir()
}

func (p *FanouttestSuite) TearDownSuite(c *C) {
	fmt.Printf("Teardown done.\n")
}

func copy(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func (p *FanouttestSuite) TestFanout(c *C) {
	fmt.Printf("\nTesting Fanout Queries\n")
	// DB - our sqlite database
	var DB *sql.DB
	var derr error
	e := copy("testdata/steward_test.db", p.dir1+"/test.db")
	c.Assert(e, IsNil)

	err := InitializeRegistryDB(p.dir1+"/test.db", false)
	c.Assert(err, IsNil)

	terr := RulesInit(p.dir1+"/test.db", "")
	c.Assert(terr, IsNil)

	fmt.Printf("\nTesting Test Database Dump Queries\n")
	DB, derr = sql.Open("sqlite3", p.dir1+"/test.db")
	c.Assert(derr, IsNil)

	cerr := DumpClients(DB)
	c.Assert(cerr, IsNil)
	cerr = DumpClientStates(DB)
	c.Assert(cerr, IsNil)
	cerr = DumpEndpoints(DB)
	c.Assert(cerr, IsNil)
	cerr = DumpEndpointStates(DB)
	c.Assert(cerr, IsNil)
	ptr1, berr := GatherReeves(DB)
	c.Assert(ptr1, Not(IsNil))
	c.Assert(berr, IsNil)
	ptr, aerr := GatherCatalog(DB)
	c.Assert(aerr, IsNil)
	c.Assert(ptr, Not(IsNil))
	cerr = DumpDBErrors(DB)
	c.Assert(cerr, IsNil)
	cerr = DumpDBStateTime(DB)
	c.Assert(cerr, IsNil)
	fmt.Printf("\nTesting Test Database Dump Finished\n")

	tu, yerr := UpdateOnTick(DB, 0)
	fmt.Printf("UpdateonTick 0 Errors? [%v]?\n", yerr)
	c.Assert(yerr, IsNil)
	c.Assert(tu, Not(IsNil))

	tu, yerr = UpdateOnTick(DB, 1)
	fmt.Printf("UpdateonTick 1 Errors? [%v]?\n", yerr)
	c.Assert(yerr, IsNil)
	c.Assert(tu, Not(IsNil))

	tu, yerr = UpdateOnTick(DB, 2)
	fmt.Printf("UpdateonTick 2 Errors? [%v]?\n", yerr)
	c.Assert(yerr, IsNil)
	c.Assert(tu, Not(IsNil))

	tu, yerr = UpdateOnTick(DB, 3)
	fmt.Printf("UpdateonTick 3 Errors? [%v]?\n", yerr)
	c.Assert(yerr, IsNil)
	c.Assert(tu, Not(IsNil))

}
