// Merkle tree routines. or more exactly, maintain hash tree data
// within a directory hierarchy. the data for a directory is
// contained in a magic file, and includes the hash for teh directory
// under the name "."

package merkle

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/erixzone/crux/pkg/begat/common"
)

// special file and directory names
const (
	MerkleInfo = ".Merkle_info"
	MerkleGo   = ".Merkle_go"
	Dot        = "."
)

// Object is a file-like thing.
type Object struct {
	Name string
	Hash common.Hash
	Mod  time.Time
}

// EventMerkle describes file system changes coming out of a merkle monitor.
type EventMerkle struct {
	Op   EventOp
	File string
	Hash string
	Ack  chan bool
}

func (o *Object) String() string {
	return fmt.Sprintf("Obj{n=%s mod=%s hash=%s}", o.Name, o.Mod, o.Hash)
}

// UpdateHashTree updates the hash tree starting at dir.
func UpdateHashTree(dir string, notify chan EventMerkle, flare bool, showall bool, ch chan bool) (*Object, error) {
	// read the current hash data
	me, exist := readMagic(dir)
	// run through this dir
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	files, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}
	// look at each entry, and rehash if necessary
	actual := make(map[string]*Object)
	changed := !exist // if magic file didn't exist, always remake it even if empty
	for _, file := range files {
		entry := file.Name()
		name := filepath.Join(dir, entry)
		if file.Mode().IsDir() {
			// first, recursively update its hash
			cur, err1 := UpdateHashTree(name, notify, false, showall, ch)
			if err1 != nil {
				return nil, err1
			}
			cur.Name = entry // it was Dot in that directory
			// did it change (or is it not there)?
			last, ok := me[entry]
			if (ok == false) || (cur.Hash != last.Hash) {
				if ok {
					notify <- EventMerkle{Op: MerkleChange, File: name, Hash: cur.Hash.String(), Ack: nil}
				} else {
					notify <- EventMerkle{Op: MerkleNew, File: name, Hash: cur.Hash.String(), Ack: nil}
				}
				changed = true
			} else {
				if showall {
					notify <- EventMerkle{Op: MerkleNew, File: name, Hash: cur.Hash.String(), Ack: nil}
				}
			}
			actual[entry] = cur
			delete(me, entry)
		} else if file.Mode().IsRegular() {
			if entry == MerkleInfo {
				continue
			}
			last, ok := me[entry]
			actual[entry] = me[entry] // by default, copy
			sent := false
			// if its out of date, recompute hash to see if it is really different
			if (ok == true) && file.ModTime().After(last.Mod) {
				x, err1 := hashFile(name)
				if err1 != nil {
					fmt.Printf("hashfile error %s\n", err1)
					return nil, err1
				}
				if x.Hash != last.Hash {
					notify <- EventMerkle{Op: MerkleChange, File: name, Hash: x.Hash.String(), Ack: nil}
					actual[entry] = x
					changed = true
					sent = true
				}
			}
			if ok == false {
				actual[entry], err = hashFile(name)
				if err != nil {
					fmt.Printf("hashfile error %s\n", err)
					return nil, err
				}
				notify <- EventMerkle{Op: MerkleNew, File: name, Hash: actual[entry].Hash.String(), Ack: nil}
				changed = true
				sent = true
			}
			if showall && !sent {
				notify <- EventMerkle{Op: MerkleNew, File: name, Hash: actual[entry].Hash.String(), Ack: nil}
			}
			delete(me, entry)
		}
	}
	actual[Dot] = me[Dot]
	delete(me, Dot)
	delete(me, MerkleInfo)
	for entry := range me {
		changed = true
		notify <- EventMerkle{Op: MerkleDelete, File: filepath.Join(dir, entry), Hash: "", Ack: nil}
	}
	// done with the scan; do we need to recreat the magic file?
	if changed {
		actual[Dot] = writeMagic(dir, actual)
		notify <- EventMerkle{Op: MerkleChange, File: dir, Hash: actual[Dot].Hash.String(), Ack: nil}
	}
	if flare {
		notify <- EventMerkle{Op: MerkleDone, File: dir, Ack: ch}
	}
	return actual[Dot], nil
}

// Loop is a daemon monitoring a filetree.
func Loop(dir string, period int, notify chan EventMerkle, ping chan chan bool, shutdown chan bool) {
	_, err := UpdateHashTree(dir, notify, true, true, nil)
	if err != nil {
		panic(err)
	}
	t := time.NewTicker(time.Duration(period) * time.Second)
loop:
	for {
		select {
		case <-shutdown:
			notify <- EventMerkle{Op: MerkleExit}
			break loop
		case <-t.C:
			_, err := UpdateHashTree(dir, notify, false, false, nil)
			if err != nil {
				panic(err)
			}
		case ch := <-ping:
			_, err := UpdateHashTree(dir, notify, true, false, ch)
			if err != nil {
				panic(err)
			}
		}
	}
}

// consider changing read|writeMagic to use json
// read a directory's magic file and return as a map
func readMagic(dir string) (map[string]*Object, bool) {
	m := make(map[string]*Object)
	path := filepath.Join(dir, MerkleInfo)
	file, err := os.Open(path)
	if err != nil {
		// only print this if really pedantic; it is harmless
		// fmt.Fprintf(os.Stderr, "warning: %s: %s\n", path, err)
		return m, false
	}
	defer file.Close()
	rd := bufio.NewReader(file)
	for {
		var s string
		var t int64
		var buf common.RawHash
		n, err := fmt.Fscanf(rd, "%s %d ", &s, &t)
		if err == io.EOF {
			break
		}
		if (n != 2) || (err != nil) {
			fmt.Fprintf(os.Stderr, "bad format in %s\n", path)
			return m, false
		}
		o := Object{Name: s, Mod: time.Unix(0, t)}
		for i := range buf {
			var j int
			fmt.Fscanf(rd, "%2x", &j)
			buf[i] = byte(j)
		}
		o.Hash = common.GetHash(buf)
		fmt.Fscanf(rd, " ") // eat newline
		m[o.Name] = &o
	}
	return m, true
}

// sorting stuff
type objects []*Object

func (oo objects) Len() int           { return len(oo) }
func (oo objects) Swap(i, j int)      { oo[i], oo[j] = oo[j], oo[i] }
func (oo objects) Less(i, j int) bool { return oo[i].Name < oo[j].Name }

//write out the magic file
func writeMagic(dir string, m map[string]*Object) *Object {
	dot := Object{Name: Dot, Mod: time.Now()}
	temp := make([]*Object, 0, len(m))
	for f, o := range m {
		if (f != Dot) && (f != MerkleInfo) {
			temp = append(temp, o)
		}
	}
	// canonical order
	sort.Sort(objects(temp))
	path := filepath.Join(dir, MerkleInfo)
	file, err := os.Create(path)
	if err != nil {
		// should do something here
		panic(err)
	}
	defer file.Close()
	contents := make([]byte, 100*len(temp)) // plausible guess; it really doesn't matter
	wr := bufio.NewWriter(file)
	for _, o := range temp {
		o.write(wr)
		contents = append(contents, o.Name...)
		buf := o.Hash.Bytes()
		contents = append(contents, buf[:64]...)
	}
	dot.Hash = common.GetHash(sha3.Sum512(contents))
	dot.write(wr)
	wr.Flush()
	return &dot
}

// write out a single object
func (o *Object) write(wr io.Writer) {
	fmt.Fprintf(wr, "%s %d %s\n", o.Name, o.Mod.UnixNano(), o.Hash)
}

// hash a file
func hashFile(path string) (o *Object, e error) {
	// first, snag the mod time. do it first so that if the file updates, we'll know
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	o = &Object{Mod: fi.ModTime()}
	h := sha3.New512()
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	rdr := bufio.NewReader(file)
	buf := make([]byte, 4096) // the number doesn't matter much
	for {
		n, err1 := rdr.Read(buf)
		if err1 != nil {
			break
		}
		h.Write(buf[0:n])
	}
	if (err != nil) && (err != io.EOF) {
		return nil, err
	}
	// finally done!
	o.Name = filepath.Base(path)
	var rh common.RawHash
	h.Sum(rh[:0])
	o.Hash = common.GetHash(rh)
	//		fmt.Printf("%s: %s %d\n", path, o.Hash, o.Hash)
	return o, nil
}

// RmTree removes the whole directory tree under and including dir.
func RmTree(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		fmt.Printf("Warning: couldn't remove tree %s\n", err)
	}
}
