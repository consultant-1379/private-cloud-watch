package kv

import (
	"fmt"
	"sort"

	"github.com/erixzone/crux/pkg/crux"
)

// LocalKV is a local implementation of the KV interface.
type LocalKV struct {
	index int64
	m     map[string]string
}

// NewLocalKV returns a local KV.
func NewLocalKV() *LocalKV {
	s := LocalKV{index: 1, m: make(map[string]string, 0)}
	return &s
}

// Get returns the value for the given key.
func (db *LocalKV) Get(key string) (string, *crux.Err) {
	v, ok := db.m[key]
	if !ok {
		return "", crux.ErrF("key %s not found", key)
	}
	return v, nil
}

// GetMissingOK is like Get, but a missing key retuns the empty string rather than an error.
func (db *LocalKV) GetMissingOK(key string) (string, *crux.Err) {
	v, ok := db.m[key]
	if !ok {
		return "", nil
	}
	return v, nil
}

// Put sets the value for the given key.
func (db *LocalKV) Put(key, val string) *crux.Err {
	db.index++
	db.m[key] = val
	return nil
}

// PutUnique chooses a nonexistent key and sets the value to that key.
func (db *LocalKV) PutUnique(dir, val string) *crux.Err {
	if dir[len(dir)-1:] != "/" {
		dir += "/"
	}
	for {
		db.index++
		key := fmt.Sprintf("%s%012d", dir, db.index)
		if _, ok := db.m[key]; !ok {
			db.m[key] = val
			return nil
		}
	}
}

// GetKeys returns a list of all teh keys that start with the given prefix.
func (db *LocalKV) GetKeys(prefix string) []string {
	var res []string
	if prefix == "" {
		for k := range db.m {
			res = append(res, k)
		}
	} else {
		plen := len(prefix)
		for k := range db.m {
			if (len(k) >= plen) && (k[:plen] == prefix) {
				res = append(res, k)
			}
		}
	}
	sort.Strings(res)
	return res
}

// CAS is a test and set; set the new value for a key if and only if that key has the given old value.
func (db *LocalKV) CAS(key, ovalue, nvalue string) *crux.Err {
	v, ok := db.m[key]
	if !ok {
		return crux.ErrF("key %s not found", key)
	}
	if v != ovalue {
		return crux.ErrF("key %s: value mismatch: expected %s, got %s", key, ovalue, v)
	}
	db.index++
	db.m[key] = nvalue
	return nil
}

// PopQueue picks a key from a "directory", deletes it, and returns its value.
func (db *LocalKV) PopQueue(dir string) (string, *crux.Err) {
	// ensure dir is wellformed
	if dir[len(dir)-1:] != "/" {
		dir += "/"
	}
	// get entries
	ents := db.GetKeys(dir)
	if len(ents) == 0 {
		return "", nil
	}
	head := ents[0] // this is the one we'll return
	v, ok := db.m[head]
	if !ok {
		return "", crux.ErrF("internal error: popqueue(%s) failed on %s", dir, head)
	}
	delete(db.m, head)
	return v, nil
}

// Leader returns the leader of the KV store. Does this even make sense?? TBD
func (db *LocalKV) Leader() string {
	return "huh?" // TBD
}
