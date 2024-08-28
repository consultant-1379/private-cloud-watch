/*
	one thing we do a lot of in Merkle-like schemes is to compare cryptohashes.
i mean, a LOT. yet, these hashes don't change much. so as an efficiency thing, we introduce
a new type (Hash) such that comparing Hash's is cheap. And of course, we need a way to convert
a Hash to and from a cryptohash. this also largely insulates us from the actual choice of cryptohash
as very little code needs to know the cryptohash details. but for the record, we use SHA3 512 bit.

	to be specific, the underlying cryptohash is 512 bits, or a 64 byte type (RawHash). and
Hash is a simple int (basically an index into a list of distinct RawHash's). we never record
the Hash, and so these are purely private local equivalences. in order to record hashes, we
also have routines that map Hash to string (hex encoded) versions of the RawHash.
*/

package common

import (
	"encoding/hex"
	"sync"
)

// random constants
const (
	HashLen = 512 / 8
	nhash   = 100 // initial guess
)

// RawHash is the checksum proper.
type RawHash [HashLen]byte

// Hash is what we crave.
type Hash int

var me struct {
	sync.Mutex
	hashmap map[RawHash]Hash
	hashvec []RawHash
	hashi   Hash
}

func init() {
	me.hashmap = make(map[RawHash]Hash, nhash)
	me.hashi = 1
	me.hashvec = make([]RawHash, 0, nhash)
	me.hashvec = append(me.hashvec, RawHash{}) // put something in at 0
}

// GetHash returns the Hash (integer) for the hash.
func GetHash(h RawHash) Hash {
	me.Lock()
	defer me.Unlock()
	x, ok := me.hashmap[h]
	if ok {
		return x
	}
	me.hashmap[h] = me.hashi
	me.hashvec = append(me.hashvec, h)
	x = me.hashi
	me.hashi++
	return x
}

// GetHashS returns the Hash for a byte slice.
func GetHashS(h []byte) Hash {
	var x RawHash
	copy(x[0:HashLen], h[0:HashLen])
	return GetHash(x)
}

// GetHashString returns the Hash for a string.
func GetHashString(s string) Hash {
	var h RawHash
	b, err := hex.DecodeString(s)
	if err != nil {
		// log.Warn(....)
	} else if len(b) != HashLen {
		// log.Warn(...)
		n := len(b)
		if n > HashLen {
			n = HashLen
		}
		copy(h[0:], b[0:n])
	} else {
		copy(h[0:], b[0:HashLen])
	}
	return GetHash(h)
}

// Bytes returns the byte slice for a given Hash.
func (h Hash) Bytes() []byte {
	me.Lock()
	defer me.Unlock()
	return me.hashvec[h][:]
}

func (h Hash) String() string {
	return hex.EncodeToString(h.Bytes())
}
