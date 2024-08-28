package alembicpbnew

import (
	"testing"

	//	"github.com/erixzone/stix/pkg/alembic"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type alembicSuite struct{}

func init() {
	Suite(&alembicSuite{})
}

func (s *alembicSuite) TestAlembic(c *C) {
	source := `
	syntax = "proto3";
	//PRE kdkdk

	package stix;

	message AlembicTimestamp {
	    int64 seconds = 1;
	    int32 nanos = 2;
	}
	
	enum Parka {
		goo1 = 3;
		goo7 = 4;
	}

	message AlembicTestEnvRec {
		AlembicTimestamp t = 1;
		int32 temp = 2;
		float humidity = 3;
	}

	message AlembicTestSumRec {
		AlembicTimestamp period = 1;
		int32 n = 2;
		int64 tot = 3;
	}

	message AlembicTestQuit {}

	message AlembicTestReply {
		bool ok = 1;
	}

	service AlembicTestSum {
		rpc Sensor(stream AlembicTestEnvRec) returns (AlembicTestReply) {}
		rpc Quit(AlembicTestQuit) returns (AlembicTestReply) {}
	}

	service AlembicTestSink {
		rpc Sink(AlembicTestSumRec) returns (AlembicTestReply) {}
		rpc Quit(AlembicTestQuit) returns (AlembicTestReply) {}
	}`
	answer := `{
	Pre = 'kdkdk'
	service AlembicTestSum {
		rpc Sensor(stream AlembicTestEnvRec) returns (AlembicTestReply) {}
		rpc Quit(AlembicTestQuit) returns (AlembicTestReply) {}
	}
	service AlembicTestSink {
		rpc Sink(AlembicTestSumRec) returns (AlembicTestReply) {}
		rpc Quit(AlembicTestQuit) returns (AlembicTestReply) {}
	}
}
`

	p, err := New("precompiled", source, false)
	c.Assert(err, IsNil)
	c.Logf("parse returns:\n%s", p.String())
	c.Assert(p.String(), Equals, answer)
}
