package flock

import (
	"bytes"
	"math/rand"
	"os"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/crux"
)

func TestCrypt(t *testing.T) { TestingT(t) }

type cryptSuite struct {
	x int // not used
}

func init() {
	Suite(&cryptSuite{})
}

func (k *cryptSuite) SetUpSuite(c *C) {
}

func (k *cryptSuite) TearDownSuite(c *C) {
}

func (k *cryptSuite) TestBasic(c *C) {
	const blen = 1500.0 // ave length of packet
	const nblocks = 10  // number of packets to encrypt
	const nkeys = 5     // number of keys to test
	var blocks [][]byte
	testlog()

	// initialise blocks
	for i := 0; i < nblocks; i++ {
		k := int(0.8*blen + 0.4*rand.Float32()*blen)
		b := make([]byte, k)
		rand.Read(b)
		blocks = append(blocks, b)
	}

	// now test them
	for i := 0; i < nkeys; i++ {
		key, err := NewEncryptionKey()
		c.Assert(err, IsNil)
		for j := range blocks {
			enc, err := Encrypt(blocks[j], key)
			c.Assert(err, IsNil)
			dec, err := Decrypt(enc, key)
			c.Assert(err, IsNil)
			c.Assert(bytes.Equal(blocks[j], dec), Equals, true)
		}
	}
}

func testlog() {
	logf, err := os.Create("junk.log")
	crux.Assert(err == nil)
	log := crux.GetLoggerW(logf)
	log.Log("field1", "val1", "field2", 34, "test log")
}
