package walrus

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type walrusSuite struct {
	file string
}

func init() {
	Suite(&walrusSuite{})
}

func (s *walrusSuite) SetUpSuite(c *C) {

}

func (s *walrusSuite) TearDownSuite(c *C) {

}

// should verify logging is right, vs just checking server responses
func (s *walrusSuite) TestSendRecv(c *C) {
	// Server  loops with timer outputting message from each level. A printf for time interval to make no-loggin obvious
	//    Two different modules
	// s:= xxx.New()
	// go s

	// Client 1. turn on all levels/modules.   2. 1 module     3.  info level, etc.
	// c:= yyy.New()
	// go c

	// Shut down go-routines.  channel wrapper?

	c.Assert(true, Equals, true)
}
