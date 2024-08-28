package crux

import (
	"io"
	"log"
	"os"

	"golang.org/x/crypto/sha3"

	"github.com/erixzone/crux/pkg/begat/common"
)

// Cksum will generate a checksum.
func Cksum(path string, rd io.Reader) string {
	if path != "" {
		var err error
		rd, err = os.Open(path)
		if err != nil {
			log.Fatalf("%s: %s", path, err.Error())
		}
	}
	// get ready to read the input file
	h := sha3.New512()
	// now read it
	buf := make([]byte, 4096) // the number doesn't matter much
	var err error
	for {
		n, err := rd.Read(buf)
		if err != nil {
			break
		}
		h.Write(buf[0:n])
	}
	// make sure it went well
	if (err != nil) && (err != io.EOF) {
		log.Fatalf("whack %s", err.Error())
		return "xx"
	}
	// finally done!
	var rh common.RawHash
	h.Sum(rh[:0])
	hash := common.GetHash(rh).String()
	return hash
}
