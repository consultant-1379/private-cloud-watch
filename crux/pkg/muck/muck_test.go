package muck

// This tests the muck system internals

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func TestMucktest(t *testing.T) { TestingT(t) }

type MucktestSuite struct {
	dir1 string
	dir2 string
}

func init() {
	Suite(&MucktestSuite{})
}

func (p *MucktestSuite) SetUpSuite(c *C) {
	p.dir1 = c.MkDir()
	p.dir2 = c.MkDir()
}

func (p *MucktestSuite) TearDownSuite(c *C) {
	fmt.Printf("Teardown done.\n")
}

func resetMuckMem() {
	muckdir = ""
	principal = ""
	muckkeys = false
	alldir = ""
	currentdir = ""
	deprecdir = ""
	killeddir = ""
}

func (p *MucktestSuite) TestMuckInternals(c *C) {

	fmt.Printf("\nTesting Muck Internals\n")

	fmt.Printf("\n==== Test CheckName\n")

	goodname := "chicken-lickin_good.time.0.1.3"
	bademailname := "../chicken-lickin   "
	badcharsname := "chicken|lickin"
	funkygoodname := "Rönnbär"

	nerr := CheckName(goodname)
	fmt.Printf("CheckName(%s) Errors?: [%v]\n", goodname, nerr)
	c.Assert(nerr, IsNil)

	nerr = CheckName(funkygoodname)
	fmt.Printf("CheckName(%s) Errors?: [%v]\n", funkygoodname, nerr)
	c.Assert(nerr, IsNil)

	nerr = CheckName(bademailname)
	fmt.Printf("CheckName(%s) Errors?: [%v]\n", bademailname, nerr)
	c.Assert(nerr, Not(IsNil))

	nerr = CheckName(badcharsname)
	fmt.Printf("CheckName(%s) Errors?: [%v]\n", badcharsname, nerr)
	c.Assert(nerr, Not(IsNil))

	fmt.Printf("\n==== Test InitMuck\n")

	fmt.Printf("\n== Preflight check, should be clean\n")
	c.Assert(IsMuckInited(), Equals, false)

	sid, werr := Principal()
	fmt.Printf("Principal() Value: %s Errors?: [%v]\n", sid, werr)
	c.Assert(werr, Not(IsNil))

	fmt.Printf("\n== InitMuck with no provided name - gets NUID, in directory:\n%s\n", p.dir1)
	kerr := InitMuck(p.dir1+"/"+".muck", "")
	fmt.Printf("InitMuck() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	md := Dir()
	fmt.Printf("MuckDir:%s\n", md)

	c.Assert(IsMuckInited(), Equals, true)

	who, werr := Principal()
	fmt.Printf("Principal() Value: %s Errors?: [%v]\n", who, werr)
	c.Assert(werr, IsNil)

	fmt.Printf("\n== Init with no name - picks up previous name\n")
	kerr = InitMuck(p.dir1+"/"+".muck", "")
	fmt.Printf("InitMuck() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	who2, verr := Principal()
	c.Assert(verr, IsNil)
	c.Assert(who, Equals, who2)
	c.Assert(md, Equals, Dir())

	fmt.Printf("== Init with provided name - ignored on mismatch\n")
	kerr = InitMuck(p.dir1+"/"+".muck", "bentobox")
	fmt.Printf("InitMuck() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	who2, verr = Principal()
	c.Assert(verr, IsNil)
	c.Assert(who, Equals, who2)
	c.Assert(md, Equals, Dir())

	fmt.Printf("\n== Init with provided name, different directory should also ignore:\n%s\n", p.dir2)
	kerr = InitMuck(p.dir2+"/"+".muck", "bentobox")
	fmt.Printf("InitMuck() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	who2, verr = Principal()
	fmt.Printf("Principal() Value: %s Errors?: [%v]\n", who2, werr)
	c.Assert(verr, IsNil)
	c.Assert(who, Equals, who2)
	c.Assert(md, Equals, Dir())

	fmt.Printf("IsMuckInitied?\n")
	c.Assert(IsMuckInited(), Equals, true)

	fmt.Printf("MuckDir:%s\n", Dir())
	fmt.Printf("AllKeysDir:%s\n", AllKeysDir())
	fmt.Printf("CurrentKeysDir:%s\n", CurrentKeysDir())
	fmt.Printf("DeprecKeysDir:%s\n", DeprecKeysDir())
	fmt.Printf("KilledKeysDir:%s\n", KilledKeysDir())

	fmt.Printf("\n== Force reset, then Init with provided dir, bad principal, should error:\n%s\n", p.dir2)
	resetMuckMem()
	kerr = InitMuck(p.dir2+"/"+".muck", "ben/to/box")
	fmt.Printf("InitMuck() Errors?: [%v]\n", kerr)
	c.Assert(kerr, Not(IsNil))
	who2, verr = Principal()
	fmt.Printf("Principal() Value: %s Errors?: [%v]\n", who2, werr)
	c.Assert(verr, Not(IsNil))

	fmt.Printf("IsMuckInitied?\n")
	c.Assert(IsMuckInited(), Equals, false)

	fmt.Printf("MuckDir:%s\n", Dir())
	fmt.Printf("AllKeysDir:%s\n", AllKeysDir())
	fmt.Printf("CurrentKeysDir:%s\n", CurrentKeysDir())
	fmt.Printf("DeprecKeysDir:%s\n", DeprecKeysDir())
	fmt.Printf("KilledKeysDir:%s\n", KilledKeysDir())

	fmt.Printf("\n== Force reset, then Init with provided dir, principal, should be ok:\n%s\n", p.dir2)
	resetMuckMem()
	kerr = InitMuck(p.dir2+"/"+".muck", "bentobox")
	fmt.Printf("InitMuck() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)
	who2, verr = Principal()
	fmt.Printf("Principal() Value: %s Errors?: [%v]\n", who2, werr)
	c.Assert(verr, IsNil)

	fmt.Printf("IsMuckInitied?\n")
	c.Assert(IsMuckInited(), Equals, true)

	fmt.Printf("MuckDir:%s\n", Dir())
	fmt.Printf("AllKeysDir:%s\n", AllKeysDir())
	fmt.Printf("CurrentKeysDir:%s\n", CurrentKeysDir())
	fmt.Printf("DeprecKeysDir:%s\n", DeprecKeysDir())
	fmt.Printf("KilledKeysDir:%s\n", KilledKeysDir())
	fmt.Printf("BlobDir:%s\n", BlobDir())
	fmt.Printf("StewardDir:%s\n", StewardDir())
	fmt.Printf("RegistryDir:%s\n", RegistryDir())

	hn1 := "myhorde"
	hn2 := "yourhorde"
	fmt.Printf("HordeName stored is (not set) %s\n", HordeName(""))
	fmt.Printf("HordeName set to %s\n", HordeName(hn1))
	hn := HordeName("")
	c.Assert(hn, Equals, hn1)
	fmt.Printf("HordeName stored is %s\n", HordeName(""))
	fmt.Printf("HordeName set to %s\n", HordeName(hn2))
	hn = HordeName("")
	c.Assert(hn, Equals, hn2)
	fmt.Printf("HordeName stored is %s\n", HordeName(""))

	fn1 := "/flock/horde/node/servicename/serviceapi"
	fn2 := "/flock/horde2/node2/servicename/serviceapi"
	fmt.Printf("StewardNodeID stored is (not set) %s\n", StewardNodeID(""))
	fmt.Printf("StewardNodeID set to %s\n", StewardNodeID(fn1))
	fn := StewardNodeID("")
	c.Assert(fn, Equals, fn1)
	fmt.Printf("StewardNodeID stored is %s\n", StewardNodeID(""))
	fmt.Printf("StewardNodeID set to %s\n", StewardNodeID(fn2))
	fn = StewardNodeID("")
	c.Assert(fn, Equals, fn2)
	fmt.Printf("StewardNodeID stored is %s\n", StewardNodeID(""))

	kn1 := "/servicename/principal/keys/fingerprint"
	kn2 := "/servicename/principal2/keys/fingerprint"
	fmt.Printf("StewardKeyID stored is (not set) %s\n", StewardKeyID(""))
	fmt.Printf("StewardKeyID set to %s\n", StewardKeyID(kn1))
	kn := StewardKeyID("")
	c.Assert(kn, Equals, kn1)
	fmt.Printf("StewardKeyID stored is %s\n", StewardKeyID(""))
	fmt.Printf("StewardKeyID set to %s\n", StewardKeyID(kn2))
	kn = StewardKeyID("")
	c.Assert(kn, Equals, kn2)
	fmt.Printf("StewardKeyID stored is %s\n", StewardKeyID(""))

	nn1 := "/servicename/principal/net/address"
	nn2 := "/servicename/principal2/net/address2"
	fmt.Printf("StewardNetID stored is (not set) %s\n", StewardNetID(""))
	fmt.Printf("StewardNetID set to %s\n", StewardNetID(nn1))
	nn := StewardNetID("")
	c.Assert(nn, Equals, nn1)
	fmt.Printf("StewardNetID stored is %s\n", StewardNetID(""))
	fmt.Printf("StewardNetID set to %s\n", StewardNetID(nn2))
	nn = StewardNetID("")
	c.Assert(nn, Equals, nn2)
	fmt.Printf("StewardNetID stored is %s\n", StewardNetID(""))

	fmt.Printf("\nDone Muck Internals\n")

}
