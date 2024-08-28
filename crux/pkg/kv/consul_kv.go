package kv

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/hashicorp/consul/api"
)

// some known names
const (
	Unique     = "_unique"              // key used to store the number of the next unique name
	RetryCount = 70                     // max times we'll retry some k/v store operations
	RetryGap   = 100 * time.Millisecond // gap between retries
)

// ConsulKV is the local proxy structure
type ConsulKV struct {
	client *api.Client
	kv     *api.KV
}

// NewConsulKV returns the proxy
func NewConsulKV(addr string) (*ConsulKV, error) {
	// TBD: use addr?
	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}
	for i := 0; i < RetryCount; i++ {
		s := client.Status()
		leader, err := s.Leader()
		if (leader != "") && (err == nil) {
			peers, _ := s.Peers()
			fmt.Printf("=== elected leader=%s (%d loops); %d peers\n", leader, i, len(peers))
			break
		}
		time.Sleep(RetryGap)
	}
	ckv := ConsulKV{client: client, kv: client.KV()}
	// lastly, set the Unique variable up if its not there (harmless if multiple sets)
	pair, _, err1 := ckv.kv.Get(Unique, nil)
	if (pair == nil) || (err1 != nil) {
		ckv.Put(Unique, "xx") // value doesn't matter as long as everyone does the same value
	}
	return &ckv, nil
}

// Leader returns the leader.
func (ckv *ConsulKV) Leader() string {
	for i := 0; i < RetryCount; i++ {
		s := ckv.client.Status()
		leader, err := s.Leader()
		if (leader != "") && (err == nil) {
			// if teh leader string has the port, get rid of that
			if i := strings.Index(leader, ":"); i >= 0 {
				leader = leader[0:i]
			}
			return leader
		}
		time.Sleep(RetryGap)
	}
	return ""
}

// Get returns the value for the given key
func (ckv *ConsulKV) Get(key string) (string, *crux.Err) {
	var err error
	var pair *api.KVPair
	qopt := api.QueryOptions{RequireConsistent: true}
	pair, _, err = ckv.kv.Get(key, &qopt)
	if (pair != nil) && (err == nil) {
		//			fmt.Printf("++Get(%s) succeeded [%v]\n", key, ckv)
		return string(pair.Value), nil
	}
	//fmt.Printf("++Get(%s) failed %s\n", key, err.Error())
	return "", crux.ErrE(err)
}

// GetMissingOK returns the value for the given key, but a missing var
// returns as a "", not an error
func (ckv *ConsulKV) GetMissingOK(key string) (string, *crux.Err) {
	var err error
	var pair *api.KVPair
	pair, _, err = ckv.kv.Get(key, nil)
	if (pair != nil) && (err == nil) {
		return string(pair.Value), nil
	}
	if pair == nil { // this is apparently a missing key
		return "", nil
	}
	return "", crux.ErrE(err)
}

// Put sets the value for the given key
func (ckv *ConsulKV) Put(key, val string) *crux.Err {
	p := &api.KVPair{Key: key, Value: []byte(val)}
	_, err := ckv.kv.Put(p, nil)
	if err != nil {
		return crux.ErrF("put(%s); failed to set kv: %s\n", key, err.Error())
	}
	//	fmt.Printf("++Put(%s) succeeded [%v]\n", key, ckv)
	return nil
}

// PutUnique generates a unique key in the given dir and sets its value
func (ckv *ConsulKV) PutUnique(dir, val string) *crux.Err {
	pair, meta, err := ckv.kv.Get(Unique, nil)
	if (pair == nil) || (err != nil) {
		return crux.ErrF("putunique: can't get Unique, possible error is %s", err.Error())
	}
	if dir[len(dir)-1:] != "/" {
		dir += "/"
	}
	key := fmt.Sprintf("%012d", meta.LastIndex)
	if cerr := ckv.CAS(Unique, string(pair.Value), key); cerr != nil {
		return cerr
	}
	if meta != nil {
		return crux.ErrF("putunique: meta is nil, possible error is %s", err.Error())
	}
	return ckv.Put(dir+key, val)
}

// GetKeys returns all the keys matching the prefix
func (ckv *ConsulKV) GetKeys(prefix string) []string {
	pairs, _, err := ckv.kv.List(prefix, nil)
	//	fmt.Printf("List('%s') returns %+v [%v]\n", prefix, pairs, ckv)
	if (pairs != nil) && (err == nil) {
		var ret []string
		for _, kp := range pairs {
			ret = append(ret, string(kp.Key))
		}
		sort.Strings(ret)
		return ret
	}
	return nil
}

// CAS sets the new value for the given key only if its current value matches ovalue
func (ckv *ConsulKV) CAS(key, ovalue, nvalue string) *crux.Err {
	pair, meta, err := ckv.kv.Get(key, nil)
	if err != nil {
		return crux.ErrE(err)
	}
	if pair == nil {
		return crux.ErrF("cas('%s'): expected value", key)
	}
	if meta.LastIndex == 0 {
		return crux.ErrF("cas('%s'): expected meta: %+v", key, meta)
	}
	if string(pair.Value) != ovalue {
		return crux.ErrF("cas('%s'): expected value '%s', got '%s'", key, ovalue, pair.Value)
	}
	// do the update to the new value
	p := &api.KVPair{Key: key, Value: []byte(nvalue), ModifyIndex: meta.LastIndex}
	if work, _, err := ckv.kv.CAS(p, nil); err != nil {
		return crux.ErrE(err)
	} else if !work {
		return crux.ErrF("cas('%s'): unexpected failure", key)
	}
	return nil
}

// PopQueue returns teh value of the "next" key in that dir and deletes that key
func (ckv *ConsulKV) PopQueue(dir string) (string, *crux.Err) {
	// ensure dir is wellformed
	if dir[len(dir)-1:] != "/" {
		dir += "/"
	}
	/*
		this is a little risky. consul doesn't have a reliable way to do locks.
		so we'll do a loop, getting a candidate, and returning it IF we
		were able to delete it.
		i am assuming that only one client gets to delete an entry.
	*/
	for {
		// get entries
		ents := ckv.GetKeys(dir)
		if len(ents) == 0 {
			return "", nil
		}
		head := ents[0] // this is the one we'll return
		// get its value
		pair, _, err := ckv.kv.Get(head, nil)
		if (pair == nil) || (err != nil) {
			continue // try again
		}
		if _, err = ckv.kv.Delete(head, nil); err != nil {
			continue // try again
		}
		return string(pair.Value), nil
	}
}
