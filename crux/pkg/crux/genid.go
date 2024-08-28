package crux

import (
	"crypto/rand"
	"fmt"
)

// SmallID returns an ID with roughly 64 bits of entropy
func SmallID() string {
	return gen1() + "_" + gen1()
}

// LargeID returns an ID with roughly 128 bits of entropy
func LargeID() string {
	return gen1() + "_" + gen1() + "_" + gen1() + "_" + gen1()
}

// if we ever do a large number of IDs, replace the rand.Read's with a PRNG
// but reseed it with rand.READ every 100 or so IDs.
func gen1() string {
	code := "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890"
	base := int64(len(code))
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("Failed to read random bytes: %v", err))
	}
	i := int64(buf[3])
	i |= (int64(buf[2]) << 8) | (int64(buf[1]) << 16) | (int64(buf[0]) << 24)
	var s string
	for i > 0 {
		n := i % base
		s += code[n : n+1]
		i /= base
	}
	if s == "" {
		s = "0"
	}
	return s
}
