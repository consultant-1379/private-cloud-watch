package kv

import (
	"github.com/erixzone/crux/pkg/crux"
)

// KV is our interface to the kv store
type KV interface {
	Get(key string) (string, *crux.Err)
	GetMissingOK(key string) (string, *crux.Err)
	Put(key, val string) *crux.Err
	PutUnique(dir, val string) *crux.Err // generate a unique key within that dir
	GetKeys(prefix string) []string
	CAS(key, ovalue, nvalue string) *crux.Err // set val(key) to nvalue ONLY IF val(key) is ovalue
	PopQueue(key string) (string, *crux.Err)
	Leader() string
}
