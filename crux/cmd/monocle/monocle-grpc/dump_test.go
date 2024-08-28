package main

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/erixzone/crux/gen/cruxgen"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type DumpTest struct {
	foo string
}

func init() {
	fmt.Printf("Monocle test init\n")
	dt := DumpTest{}
	Suite(&dt)
}

func (dt *DumpTest) TestBasic(c *C) {
	// Testing monocle without the networking. Calling the grpc
	// functions directly.
	serv := Server{}
	// send dump request to "ping" subsystem

	req := &pb.GetDumpRequest{Subsystems: "ping", Level: 1}
	resp, err := serv.GetDump(context.Background(), req)
	c.Assert(err, IsNil)

	// check success bool
	c.Assert(resp.GetSuccess(), Equals, true)
	// check text is right for ping (global var)
	c.Assert(resp.GetData(), Equals, PingTestResp)
}
