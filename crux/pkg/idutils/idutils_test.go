package idutils

// This tests the netID, keyID, flocID  internals

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func TestIdUtilstest(t *testing.T) { TestingT(t) }

type IDUtilstestSuite struct {
	dir1 string
	dir2 string
}

func init() {
	Suite(&IDUtilstestSuite{})
}

func (p *IDUtilstestSuite) SetUpSuite(c *C) {
	p.dir1 = "."
	// p.dir1 = c.MkDir()
	p.dir2 = c.MkDir()
}

func (p *IDUtilstestSuite) TearDownSuite(c *C) {
	fmt.Printf("Teardown done.\n")
}

func (p *IDUtilstestSuite) TestIdUtils(c *C) {

	fmt.Printf("\nTesting IdUtils Internals\n")

	nid1 := "/Myservice/Myid/net/10.0.0.1:5555"
	ns1, err := NetIDParse(nid1)
	fmt.Printf("\nnid1 NetIDParse() Errors?: [%v]\n", err)
	c.Assert(err, IsNil)
	fmt.Printf("[%v]\n", ns1)
	fmt.Printf("in:  %s\nout: %s\n", nid1, ns1.String())

	nid2 := "/Myservice/Myid/net/localhost:5555"
	ns2, terr := NetIDParse(nid2)
	fmt.Printf("\nnid2 NetIDParse() Errors?: [%v]\n", terr)
	c.Assert(terr, IsNil)
	fmt.Printf("[%v]\n", ns2)
	fmt.Printf("in:  %s\nout: %s\n", nid2, ns2.String())

	nid3 := "/Myservice/Myid/not/localhost:5555"
	ns3, uerr := NetIDParse(nid3)
	fmt.Printf("\nnid3 NetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, Not(IsNil))
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "/Myservice/Myid/net/http://www.bob.com/testing:123456"
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\nNetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, IsNil)
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "/Myservice/Myid/net/http://www.bob.com/testing:12345O"
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\nNetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, Not(IsNil))
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "/Myservice/Myid/net/bob:123456"
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\nNetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, IsNil)
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "/Myservice/Myid/net/:123456"
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\nNetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, Not(IsNil))
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "/Myservice/*/net/"
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\nNetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, IsNil)
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "///net/*" // query form
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\n NetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, IsNil)
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	nid3 = "///net/" // query form
	ns3, uerr = NetIDParse(nid3)
	fmt.Printf("\n NetIDParse() Errors?: [%v]\n", uerr)
	c.Assert(uerr, IsNil)
	fmt.Printf("[%v]\n", ns3)
	fmt.Printf("in:  %s\nout: %s\n", nid3, ns3.String())

	kid1 := "/Jettison/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22"
	k1, kerr := KeyIDParse(kid1)
	fmt.Printf("\n KeyIDParse() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	fmt.Printf("[%v]\n", k1)
	fmt.Printf("in:  %s\nout: %s\n", kid1, k1.String())

	kid1 = "//maude/keys/*"
	k1, kerr = KeyIDParse(kid1)
	fmt.Printf("\n KeyIDParse() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	fmt.Printf("[%v]\n", k1)
	fmt.Printf("in:  %s\nout: %s\n", kid1, k1.String())

	kid1 = "//maude/keys/"
	k1, kerr = KeyIDParse(kid1)
	fmt.Printf("\n KeyIDParse() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	fmt.Printf("[%v]\n", k1)
	fmt.Printf("in:  %s\nout: %s\n", kid1, k1.String())

	kid1 = "/Jettison/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26"
	k1, kerr = KeyIDParse(kid1)
	fmt.Printf("\n KeyIDParse() Errors?: [%v]\n", kerr)
	c.Assert(kerr, Not(IsNil))
	fmt.Printf("[%v]\n", k1)
	fmt.Printf("in:  %s\nout: %s\n", kid1, k1.String())

	kid1 = "/Jettison/maude//keys/"
	k1, kerr = KeyIDParse(kid1)
	fmt.Printf("\n KeyIDParse() Errors?: [%v]\n", kerr)
	c.Assert(kerr, Not(IsNil))
	fmt.Printf("[%v]\n", k1)
	fmt.Printf("in:  %s\nout: %s\n", kid1, k1.String())

	kid1 = "/Jettison/maude/keys//"
	k1, kerr = KeyIDParse(kid1)
	fmt.Printf("\n KeyIDParse() Errors?: [%v]\n", kerr)
	c.Assert(kerr, Not(IsNil))
	fmt.Printf("[%v]\n", k1)
	fmt.Printf("in:  %s\nout: %s\n", kid1, k1.String())

	fid1 := "/flock/horde/node/Service/Api1"
	f1, ferr := NodeIDParse(fid1)
	fmt.Printf("\n NodeIDParse() Errors?: [%v]\n", ferr)
	c.Assert(ferr, IsNil)
	fmt.Printf("[%v]\n", f1)
	fmt.Printf("in:  %s\nout: %s\n", fid1, f1.String())

	fid1 = "flock/horde/node/Service/Api1"
	f1, ferr = NodeIDParse(fid1)
	fmt.Printf("\n NodeIDParse() Errors?: [%v]\n", ferr)
	c.Assert(ferr, IsNil)
	fmt.Printf("[%v]\n", f1)
	fmt.Printf("in:  %s\nout: %s\n", fid1, f1.String())

	fid1 = "flock/horde/node/Service/"
	f1, ferr = NodeIDParse(fid1)
	fmt.Printf("\n NodeIDParse() Errors?: [%v]\n", ferr)
	c.Assert(ferr, IsNil)
	fmt.Printf("[%v]\n", f1)
	fmt.Printf("in:  %s\nout: %s\n", fid1, f1.String())

	fid1 = "/flock/horde/no/de/Service/Api1"
	f1, ferr = NodeIDParse(fid1)
	fmt.Printf("\n NodeIDParse() Errors?: [%v]\n", ferr)
	c.Assert(ferr, Not(IsNil))
	fmt.Printf("[%v]\n", f1)
	fmt.Printf("in:  %s\nout: %s\n", fid1, f1.String())

	fid1 = "/flock/horde/node/Service"
	f1, ferr = NodeIDParse(fid1)
	fmt.Printf("\n NodeIDParse() Errors?: [%v]\n", ferr)
	c.Assert(ferr, Not(IsNil))
	fmt.Printf("[%v]\n", f1)
	fmt.Printf("in:  %s\nout: %s\n", fid1, f1.String())

	fmt.Printf("\nDone IdUtils Internals\n")

}
