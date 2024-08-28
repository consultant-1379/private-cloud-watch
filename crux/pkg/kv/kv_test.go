package kv

import (
	"testing"

	. "gopkg.in/check.v1"
)

func TestKV(t *testing.T) { TestingT(t) }

type KVTestSuite struct {
	root string
}

var _ = Suite(&KVTestSuite{})

// KV interface
var kv *LocalKV

func (s *KVTestSuite) SetUpSuite(c *C) {
	kv = NewLocalKV()
}

func (s *KVTestSuite) TearDownSuite(c *C) {
	// nothing
}

func (s *KVTestSuite) TestPutGet(c *C) {
	const key = "key3"
	const val = "fkjhfkjhfkjhfkhj"
	err := kv.Put(key, val)
	c.Assert(err, IsNil)

	v, err := kv.Get(key)
	c.Assert(err, IsNil)
	c.Assert(v, Equals, val)

	v, err = kv.Get("crap")
	c.Assert(err, Not(IsNil))
}

func (s *KVTestSuite) TestGetKeys(c *C) {
	const val = "stuff"
	err := kv.Put("foo1", val)
	c.Assert(err, IsNil)
	err = kv.Put("foo2", val)
	c.Assert(err, IsNil)
	err = kv.Put("foo13", val)
	c.Assert(err, IsNil)

	keys := kv.GetKeys("foo")
	c.Assert(keys, DeepEquals, []string{"foo1", "foo13", "foo2"})
	keys = kv.GetKeys("foo1")
	c.Assert(keys, DeepEquals, []string{"foo1", "foo13"})
	keys = kv.GetKeys("foo2")
	c.Assert(keys, DeepEquals, []string{"foo2"})
}

func (s *KVTestSuite) TestPutUnique(c *C) {
	const val = "stuff"
	const dir = "/barf/"
	err := kv.PutUnique(dir, val)
	c.Assert(err, IsNil)
	err = kv.PutUnique(dir, val)
	c.Assert(err, IsNil)
	err = kv.PutUnique(dir, val)
	c.Assert(err, IsNil)

	keys := kv.GetKeys(dir)
	c.Assert(keys, HasLen, 3)
}

func (s *KVTestSuite) TestCAS(c *C) {
	const key = "key13"
	const nval = "ababababa"
	const oval = "fkjhfkjhfkjhfkhj"
	err := kv.Put(key, oval)
	c.Assert(err, IsNil)
	err = kv.CAS(key, oval, nval)
	c.Assert(err, IsNil)

	v, err := kv.Get(key)
	c.Assert(err, IsNil)
	c.Assert(v, Equals, nval)

	err = kv.CAS(key, oval, nval)
	c.Assert(err, Not(IsNil))
}

func (s *KVTestSuite) TestPopQueue(c *C) {
	const val1 = "stuff1"
	const val2 = "stuff2"
	const val3 = "stuff3"
	const dir = "/barfx/"
	err := kv.PutUnique(dir, val1)
	c.Assert(err, IsNil)
	err = kv.PutUnique(dir, val2)
	c.Assert(err, IsNil)
	err = kv.PutUnique(dir, val3)
	c.Assert(err, IsNil)

	v, err := kv.PopQueue(dir)
	c.Assert(err, IsNil)
	c.Assert(v, Equals, val1)
	v, err = kv.PopQueue(dir)
	c.Assert(err, IsNil)
	c.Assert(v, Equals, val2)
	v, err = kv.PopQueue(dir)
	c.Assert(err, IsNil)
	c.Assert(v, Equals, val3)
	v, err = kv.PopQueue(dir)
	c.Assert(err, IsNil)
	c.Assert(v, Equals, "")
}
