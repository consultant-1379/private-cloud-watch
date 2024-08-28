// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/radovskyb/watcher"

	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
)

// PubKeyT - The public key information as json with username, keyid and public key one-liner
// KeyID is the KV-store key
// will use StateAdded <0 (negative) values to mark entries not managed by Reeve/Steward
type PubKeyT struct {
	Service    string `json:"service"`
	Name       string `json:"name"`
	KeyID      string `json:"keyid"`
	PubKey     string `json:"pubkey"`
	StateAdded int32  `json:"stateadded"`
}

// EndPointT - The local cache of endpoint information held by reeve
// KV-store key is NodeID + NetID concatenated together ...
// will use StateAdded <0 (negative) values to mark entries not managed by Reeve/Steward
type EndPointT struct {
	NodeID     string `json:"nodeid"`
	NetID      string `json:"netid"`
	Priority   string `json:"priority,omitempty"`
	Rank       int32  `json:"rank,omitempty"`
	StateAdded int32  `json:"stateadded"`
}

const pubkeysbucket = "PubKeys"
const endpointsbucket = "EndPoints"

// KeyIDfromPubKeyJSON - extract keyid string from a json pubkey
func KeyIDfromPubKeyJSON(pkjson string) (string, *c.Err) {
	jsonbytes := []byte(pkjson)
	pk, err := PubKeyFromJSON(jsonbytes)
	if err != nil {
		return "", c.ErrF("cannot read pubkey json - %v", err)
	}
	return pk.KeyID, nil
}

// PubKeyToJSONBytes - json marshal PubKeyT
func PubKeyToJSONBytes(pk *PubKeyT) ([]byte, *c.Err) {
	if pk == nil {
		return nil, c.ErrF("cannot json marshal nil public key")
	}
	// marshal pubkey
	pkjson, err := json.Marshal(pk)
	if err != nil {
		return nil, c.ErrF("cannot json marshal public key - %v", err)
	}
	return pkjson, nil
}

// PubKeyToJSON - json marshal PubKeyT
func PubKeyToJSON(pk *PubKeyT) (string, *c.Err) {
	pkjson, err := PubKeyToJSONBytes(pk)
	if err != nil {
		return "", err
	}
	return string(pkjson), nil
}

// EndPointToJSON - json marshal EndPoint
func EndPointToJSON(ep *EndPointT) (string, *c.Err) {
	if ep == nil {
		return "", c.ErrF("cannot json marshal nil endpoint")
	}
	// marshal endpoint
	epjson, err := json.Marshal(ep)
	if err != nil {
		return "", c.ErrF("cannot json marshal endpoint - %v", err)
	}
	return string(epjson), nil
}

// EndPointFromJSON - json unmarshall EndPointT
func EndPointFromJSON(jsonbytes []byte) (*EndPointT, *c.Err) {
	if len(jsonbytes) == 0 {
		return nil, c.ErrF("cannot unmarshal empty endpoint json")
	}
	var endpoint EndPointT
	err := json.Unmarshal(jsonbytes, &endpoint)
	if err != nil {
		return nil, c.ErrF("endpoint unmarshal failed, malformed json: %v", err)
	}
	return &endpoint, nil
}

// PubKeyFromJSON - json unmarshall PubKeyT
func PubKeyFromJSON(jsonbytes []byte) (*PubKeyT, *c.Err) {
	if len(jsonbytes) == 0 {
		return nil, c.ErrF("cannot unmarshal empty json public key")
	}
	var pubkey PubKeyT
	err := json.Unmarshal(jsonbytes, &pubkey)
	if err != nil {
		return nil, c.ErrF("public key unmarshal failed, malformed json: %v", err)
	}
	return &pubkey, nil
}

// Db - the BoltDB of public keys - the local whitelist and endpoint database
var Db *bolt.DB

// dblogger - local logger for BoltDB operations
var dblogger clog.Logger

// PubKeyDBExists - returns true if the whitelist database exists
func PubKeyDBExists(dbname string) bool {
	// stat the file
	if dbname == "" {
		return false
	}
	if _, err := os.Stat(dbname); err == nil {
		return true
	}
	return false
}

// InitPubKeyLookup - takes the filename of the database, initializes boltdb.
// if verbose, dumps the json list of database public keys on startup
func InitPubKeyLookup(dbname string, dblog clog.Logger) *c.Err {
	if dblog == nil {
		msg0 := fmt.Sprintf("InitPubKeyLookup - missing logger")
		return c.ErrF("%s", msg0)
	}
	pidstr, ts := GetPidTS()
	if !PubKeyDBExists(dbname) {
		msg1 := fmt.Sprintf("error - missing BoltDB %s", dbname)
		dblog.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
		return c.ErrF("%s", msg1)
	}
	// Open it read only - UMM read the BoltDB fine print - no such thing actually
	// It always opens the file in read/write mode, no matter what you insist
	var derr error
	Db, derr = bolt.Open(dbname, 0600, nil)
	if derr != nil {
		msg2 := fmt.Sprintf("error - cannot open grpcsig BoltDB for reading from file %s [%v]", dbname, derr)
		dblog.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg2)
		return c.ErrF("%s", msg2)
	}
	dblogger = dblog
	dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig BoltDB %s opened", dbname))
	return nil
}

// FiniPubKeyLookup - closes the BoldDB database
func FiniPubKeyLookup() {
	if Db != nil {
		Db.Close()
	}
	pidstr, ts := GetPidTS()
	dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig BoltDB closed"))
}

// StartNewPubKeyDB - starts a new file with buckets initialized
func StartNewPubKeyDB(dbname string) *c.Err {
	// Open it read write
	db, derr := bolt.Open(dbname, 0600, nil)
	defer db.Close()
	if derr != nil {
		return c.ErrF("cannot open grpcsig BoltDB for read/write to file %s [%v]", dbname, derr)
	}
	// Make the "PubKeys" bucket
	derr = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(pubkeysbucket))
		if err != nil {
			return err
		}
		return nil
	})
	if derr != nil {
		return c.ErrF("cannot create Bucket [%s] in grpcsig BoltDB file: %s\n", pubkeysbucket, dbname)
	}
	// Make the "EndPoints" bucket
	derr = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(endpointsbucket))
		if err != nil {
			return err
		}
		return nil
	})
	if derr != nil {
		return c.ErrF("cannot create Bucket [%s] in grpcsig BoltDB file: %s\n", endpointsbucket, dbname)
	}
	return nil
}

// RemoveSelfPubKeysFromDB - removes all transient public keys starting with the
// self keyId pattern "/self/self" from the database.
func RemoveSelfPubKeysFromDB() *c.Err {
	if Db == nil {
		return c.ErrF("RemoveSelfPubKeysFromDb - grpcsig BoltDB not open")
	}
	// Since we don't have a bucket for anything but KeyID, scan the whole file
	// Dump the key db, collect lines into an array
	Dbjson := []string{}
	// Collect all the keys with the selfkey prefix using a seek and loop
	selfkeyprefix := []byte("/self/self")
	err := Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(pubkeysbucket))
		if b == nil {
			return fmt.Errorf("grpcsig BoltDB bucket %s not found", pubkeysbucket)
		}
		c := b.Cursor()
		for k, _ := c.Seek(selfkeyprefix); k != nil && bytes.HasPrefix(k, selfkeyprefix); k, _ = c.Next() {
			Dbjson = append(Dbjson, string(k[:]))
		}
		return nil
	})
	if err != nil {
		pidstr, ts := GetPidTS()
		msg := fmt.Sprintf("grpcsig BoltDB self key json extract failed: %v", err)
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%s", msg)
	}
	for _, key := range Dbjson {
		derr := Db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(pubkeysbucket))
			if bucket == nil {
				return fmt.Errorf("bucket %s not found", pubkeysbucket)
			}
			return bucket.Delete([]byte(key))
		})
		if derr != nil {
			pidstr, ts := GetPidTS()
			msg1 := fmt.Sprintf("cannot delete key from grpcsig BoltDB database %v", derr)
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
			return c.ErrF("%s", msg1)
		}
	}
	pidstr, ts := GetPidTS()
	dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "self keys removed from grpcsig BoltDB database")
	return nil
}

// RemoveServiceRevPubKeysFromDB - removes all transient public keys starting with the
// keyId pattern "servicerev" from the database.
func RemoveServiceRevPubKeysFromDB(servicerev string) *c.Err {
	if Db == nil {
		return c.ErrF("RemoveServiceRevPubKeysFromDb - grpcsig BoltDB not open")
	}
	// Ensure the string provided is delimited - so it must only search the first field of key
	delimrev := fmt.Sprintf("/%s/", servicerev)

	// Since we don't have a bucket for anything but KeyID prefix, scan the whole file
	// Dump the key db, collect lines into an array
	Dbjson := []string{}
	// Collect all the keys with the selfkey prefix using a seek and loop
	keyprefix := []byte(delimrev)
	err := Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(pubkeysbucket))
		if b == nil {
			return fmt.Errorf("grpcsig BoltDB bucket %s not found", pubkeysbucket)
		}
		c := b.Cursor()
		for k, _ := c.Seek(keyprefix); k != nil && bytes.HasPrefix(k, keyprefix); k, _ = c.Next() {
			Dbjson = append(Dbjson, string(k[:]))
		}
		return nil
	})
	if err != nil {
		pidstr, ts := GetPidTS()
		msg := fmt.Sprintf("grpcsig BoltDB prefixf key json extract failed on %s: %v", delimrev, err)
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%s", msg)
	}
	for _, key := range Dbjson {
		derr := Db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(pubkeysbucket))
			if bucket == nil {
				return fmt.Errorf("bucket %s not found", pubkeysbucket)
			}
			return bucket.Delete([]byte(key))
		})
		if derr != nil {
			pidstr, ts := GetPidTS()
			msg1 := fmt.Sprintf("cannot delete key from grpcsig BoltDB database : %s - %v", key, derr)
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
			return c.ErrF("%s", msg1)
		}
	}
	pidstr, ts := GetPidTS()
	dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("keys %s removed from grpcsig BoltDB database", delimrev))
	return nil
}

// RemovePubKeyFromDB - removes PubKey from db
func RemovePubKeyFromDB(pk *PubKeyT) *c.Err {
	if Db == nil {
		return c.ErrF("RemovePubKeyFromDB - grpcsig BoltDB not open")
	}
	exists, err := pubKeyExists(pk.KeyID)
	if err != nil {
		return err
	}
	if exists {
		derr := Db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(pubkeysbucket))
			if bucket == nil {
				return fmt.Errorf("bucket %s not found", pubkeysbucket)
			}
			return bucket.Delete([]byte(pk.KeyID))
		})
		if derr != nil {
			pidstr, ts := GetPidTS()
			msg1 := fmt.Sprintf("cannot delete key %s from grpcsig BoltDB database %v", pk.KeyID, derr)
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
			return c.ErrF("%s", msg1)
		}
	}
	return nil
}

// RemoveEndPointFromDB - removes EndPoint from db
func RemoveEndPointFromDB(ep *EndPointT) *c.Err {
	if Db == nil {
		return c.ErrF("RemoveEndPointFromDB - grpcsig BoltDB not open")
	}
	key, kerr := KeyFromEndPoint(ep)
	if kerr != nil {
		msg := fmt.Sprintf("%v", kerr)
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}
	exists, err := endpointExists(key)
	if err != nil {
		return err
	}
	if exists {
		derr := Db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(endpointsbucket))
			if bucket == nil {
				return fmt.Errorf("bucket %s not found", endpointsbucket)
			}
			return bucket.Delete([]byte(key))
		})
		if derr != nil {
			pidstr, ts := GetPidTS()
			msg1 := fmt.Sprintf("cannot delete endpoint %s from grpcsig BoltDB database %v", key, derr)
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
			return c.ErrF("%s", msg1)
		}
	}
	return nil
}

// AddPubKeyBlockUpdateToDB - inserts a set of public keys (obtained from reeve/steward) into the database.
func AddPubKeyBlockUpdateToDB(pubkeys []PubKeyT) *c.Err {
	for _, pubkey := range pubkeys {
		exists, err := pubKeyExists(pubkey.KeyID)
		if err != nil {
			return err
		}
		if !exists {
			err = AddPubKeyToDB(&pubkey)
			if err != nil {
				msg := fmt.Sprintf("AddPubKeyBlockUpdateToDB error writing to grpcsig BoltDB, pubkey [%v]: %v", pubkey, err)
				pidstr, ts := GetPidTS()
				dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
				return c.ErrF("%v", msg)
			}
		}
	}
	return nil
}

// KeyFromEndPoint - makes the key used in the Bolt database for the endpoints table
func KeyFromEndPoint(ep *EndPointT) (string, *c.Err) {
	if ep == nil {
		return "", c.ErrF("no key from nil EndPoint")
	}
	nid, nerr := idutils.NetIDParse(ep.NetID)
	if nerr != nil {
		return "", c.ErrF("grpcsig KeyFromEndPoint - could not parse NetID %s : %v", ep.NetID, nerr)
	}
	nod, ferr := idutils.NodeIDParse(ep.NodeID)
	if ferr != nil {
		return "", c.ErrF("grpcsig KeyFromEndPoint - could not parse NodeID %s : %v", ep.NetID, ferr)
	}
	key := nod.ServiceName + "/" + nid.ServiceRev + "/" + nod.NodeName + "/" + nid.Principal
	return key, nil
}

// AddEndPointBlockUpdateToDB - inserts a set of endpoints (obtained from reeve/steward) into the database.
func AddEndPointBlockUpdateToDB(endpoints []EndPointT) *c.Err {
	for _, endpoint := range endpoints {
		key, kerr := KeyFromEndPoint(&endpoint)
		if kerr != nil {
			msg := fmt.Sprintf("%v", kerr)
			pidstr, ts := GetPidTS()
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
			return c.ErrF("%v", msg)
		}
		exists, err := endpointExists(key)
		if err != nil {
			msg2 := fmt.Sprintf("%v", err)
			pidstr, ts := GetPidTS()
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg2)
			return c.ErrF("%v", msg2)
		}
		if !exists {
			err = AddEndPointToDB(key, &endpoint)
			if err != nil {
				msg3 := fmt.Sprintf("AddEndPointBlockUpdateToDB error writing endpoint [%v]: %v", endpoint, err)
				pidstr, ts := GetPidTS()
				dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg3)
				return c.ErrF("%v", msg3)
			}
		}
	}
	return nil
}

// RemovePubKeys - removes a set of public keys from the pubkey database (reeve - deprecated keys)
func RemovePubKeys(keyIDs []string) *c.Err {
	if Db == nil {
		return c.ErrF("RemovePubKeys - grpcsig BoltDB not open")
	}
	if len(keyIDs) == 0 {
		// nothing to do
		return nil
	}
	for _, keyID := range keyIDs {
		derr := Db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(pubkeysbucket))
			if bucket == nil {
				return fmt.Errorf("bucket %s not found", pubkeysbucket)
			}
			return bucket.Delete([]byte(keyID))
		})
		if derr != nil {
			msg := fmt.Sprintf("RemovePubKeys cannot delete public key %s from grpcsig BoltDB : %v", keyID, derr)
			pidstr, ts := GetPidTS()
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
			return c.ErrF("%v", msg)
		}
	}
	return nil
}

// AddPubKeyToDB - Adds a public key
func AddPubKeyToDB(pubkey *PubKeyT) *c.Err {
	if Db == nil {
		return c.ErrF("AddPubKeyToDB - grpcsig BoltDB not open")
	}
	pkjson, jerr := PubKeyToJSON(pubkey)
	if jerr != nil {
		return jerr
	}
	derr := Db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(pubkeysbucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", pubkeysbucket)
		}
		return bucket.Put([]byte(fmt.Sprintf("%s", pubkey.KeyID)), []byte(pkjson))
	})
	if derr != nil {
		msg := fmt.Sprintf("AddPubKeyToDB cannot add public key to grpcsig BoltDB : %v", derr)
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}
	return nil
}

// AddEndPointToDB - Adds an endpoint
func AddEndPointToDB(key string, endpoint *EndPointT) *c.Err {
	if Db == nil {
		return c.ErrF("AddEndPointToDB grpcsig BoltDB not open")
	}
	epjson, jerr := EndPointToJSON(endpoint)
	if jerr != nil {
		msg0 := fmt.Sprintf("AddEndPointToDB failed : %v", jerr)
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg0)
		return c.ErrF("%v", msg0)
	}
	derr := Db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(endpointsbucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", endpointsbucket)
		}
		// fmt.Printf("--add(%s) %s\n", key, epjson)
		return bucket.Put([]byte(key), []byte(epjson))
	})
	if derr != nil {
		msg := fmt.Sprintf("AddEndPointToDB cannot add endpoint to grpcsig BoltDB : %v", derr)
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}
	return nil
}

// pubKeyExists - returns true if key exists in database
// does not compare value for equality.  In our case the
// trailing fingerprint of the KeyId is a hash of the database value,
// so finding the KeyId as a key is sufficient.
func pubKeyExists(keyid string) (bool, *c.Err) {
	if Db == nil {
		return false, c.ErrF("grpcsig BoltDB not open")
	}
	derr := Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(pubkeysbucket))
		if b == nil {
			return fmt.Errorf("grpcsig BoltDB bucket %s not found", pubkeysbucket)
		}
		val := b.Get([]byte(keyid))
		if val == nil {
			return fmt.Errorf("no key")
		}
		return nil
	})
	if derr != nil {
		if derr.Error() == "no key" {
			return false, nil
		}
		return false, c.ErrF("%v", derr)
	}
	return true, nil
}

// endpointExists - returns true if key exists in database
// does not compare value for equality.  In our case the
// trailing fingerprint of the KeyId is a hash of the database value,
// so finding the KeyId as a key is sufficient.
func endpointExists(endpointkey string) (bool, *c.Err) {
	if Db == nil {
		return false, c.ErrF("grpcsig BoltDB not open")
	}
	derr := Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(endpointsbucket))
		if b == nil {
			return fmt.Errorf("grpcsig BoltDb bucket %s not found", endpointsbucket)
		}
		val := b.Get([]byte(endpointkey))
		if val == nil {
			return fmt.Errorf("no key")
		}
		return nil
	})
	if derr != nil {
		if derr.Error() == "no key" {
			return false, nil
		}
		return false, c.ErrF("%v", derr)
	}
	return true, nil
}

// EndpointScan - grpcsig BoltDB function servicing EndpointsUP reeve query
func EndpointScan(toservices []string, hordename, servicerev string, limit int) ([]EndPointT, *c.Err) {
	if Db == nil {
		return nil, c.ErrF("grpcsig DB not open for EndpointScan query")
	}
	endpoints := []EndPointT{}
	for _, toservice := range toservices {
		// Do a prefix query on toservice, servicerev
		derr := Db.View(func(tx *bolt.Tx) error {
			prefix := []byte(toservice + "/" + servicerev)
			/*
				fmt.Printf("-->prefix '%s'\n", string(prefix))
				cc := tx.Bucket([]byte(endpointsbucket)).Cursor()
				for k, v := cc.First(); k != nil; k, v = cc.Next() {
					_ = v
					fmt.Printf("--k = %s :: %s\n", k, v)
				}
			*/
			c := tx.Bucket([]byte(endpointsbucket)).Cursor()
			for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
				ep, jerr := EndPointFromJSON(v)
				if jerr != nil {
					return fmt.Errorf("%v", jerr)
				}
				endpoints = append(endpoints, *ep)
				// fmt.Printf("\tadding %+v\n", *ep)
			}
			return nil
		})
		if derr != nil {
			msg := fmt.Sprintf("EndPointScan failed on grpcsig BoltDB : %v", derr)
			pidstr, ts := GetPidTS()
			dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
			return nil, c.ErrF("%v", msg)
		}
	}
	if (limit <= 0) || (limit > len(endpoints)) {
		limit = len(endpoints)
	}
	return endpoints[0:limit], nil
}

// PubKeysDBLookup - given a keyID string, returns the public key information
// from the BoltDB database, resource is ignored.
func PubKeysDBLookup(resource string, keyid string) (string, *c.Err) {
	if Db == nil {
		return "", c.ErrF("grpcsig BoltDB not open for query: %s", resource)
	}
	var JSONBytes []byte
	derr := Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(pubkeysbucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", pubkeysbucket)
		}
		JSONBytes = b.Get([]byte(keyid))
		if JSONBytes == nil {
			return fmt.Errorf("no match for KeyID %s", keyid)
		}
		return nil
	})
	if derr != nil {
		msg := fmt.Sprintf("PubKeysDBLookup grpcsig BoltDB item not found : %v", derr)
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg)
		return "", c.ErrF("%v", msg)
	}
	pubkey, err := PubKeyFromJSON(JSONBytes)
	if err != nil {
		msg2 := fmt.Sprintf("PubKeysDBLookup failed, malformed json : %v", err)
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg2)
		return "", c.ErrF("%v", msg2)
	}
	return pubkey.PubKey, nil
}

// DBWatcher - uses package github.com/radovskyb/watcher - which maintains
// os.Stat information (file timestamp, attributes) - to poll the
// status of the BoltDB public key database file every 3 seconds.
// It starts a go-routine to monitor the whitelist db, logs events,
// and restarts the db access.
// Provided so that an administrator or script can replace the whitelist db file,
// while service(s) are running.
// If db is deleted, the watcher re-checks every 20s to see if a viable
// replacement is provided, and logs an error on each failed attempt.
// Meanwhile client-side requests get bounced with unauthenticated
// messages while the server whitelist database is missing;
// e.g.
// rpc error: code = NotFound desc = Error - : unauthenticated - server cannot
// find public key with provided http-signatures header keyId '...' :
// public key Bolt Db item not found: [database not open]
// Also Note: BoltDB will continue to offer up queries
// from its cache (esp on small databases) even after a database file
// is deleted, for a short time (more so on Mac OSX than Linux, ymmv).
func DBWatcher(dbname string) *c.Err {
	pidstr, ts := GetPidTS()
	dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher starting for %s", dbname))
	watch := watcher.New()
	watch.SetMaxEvents(1)
	// Note: since we are only watching 1 file (and not the directory it lives in),
	// the watcher package code will never send us watcher.Remove, watcher.Rename or watcher.Move
	// events. Instead we get watch.Error channel errors with watcher.ErrWatchedFileDeleted
	// So we filter out the events we do not expect ever to see, and list only those we want to see
	watch.FilterOps(watcher.Write, watcher.Chmod, watcher.Create)
	defer watch.Close()
	var err error
	go func() {
		for {
			select {
			case event := <-watch.Event:
				pidstr, ts := GetPidTS()
				dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher event: %s", event.String()))
				if (event.Op&watcher.Write == watcher.Write) ||
					(event.Op&watcher.Create == watcher.Create) ||
					(event.Op&watcher.Chmod == watcher.Chmod) {
					// e.g. watcher is active on pubkeys.db and one of these happens:
					//      mv pubkeys_new.db pubkeys.db
					//      touch pubkeys.db
					//	cp pubkeys_new.db pubkeys.db
					//	chmod ??? pubkeys.db
					dblogger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB altered, restarting DB: %s", dbname))
					restarted := false
					for !restarted {
						derr := DBRestart(dbname, dblogger)
						if derr != nil {
							pidstr, ts := GetPidTS()
							dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB failed to restart: %v, retrying in 20s", derr))
							time.Sleep(20 * time.Second)
							continue
						}
						pidstr, ts := GetPidTS()
						dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB restarted %s", dbname))
						restarted = true
					}
				}
			case err := <-watch.Error:
				// This is hit in the event loop when the watched file is deleted
				// e.g.	mv pubkeys.db junk
				//	rm pubkeys.db
				// NB watch.Remove(dbname) is done already internally
				restarted := false
				pidstr, ts := GetPidTS()
				dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB removed, will check for file and restart in 20s %s", dbname))
				time.Sleep(20 * time.Second)
				for !restarted {
					derr := DBRestart(dbname, dblogger)
					if derr != nil {
						pidstr, ts := GetPidTS()
						dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB failed to restart %s: %v, retrying in 20s", dbname, derr))
						time.Sleep(20 * time.Second)
						continue
					}
					pidstr, ts := GetPidTS()
					dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB restarted %s", dbname))
					restarted = true
				}
				err = watch.Add(dbname)
				if err != nil {
					pidstr, ts := GetPidTS()
					dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher error adding file %s,  %v", dbname, err))
					// IF - in the tiny interval between the above block
					// and the watch.Add call, the file is deleted again, we re-up this error
					// in a goroutine so that it does not block watch.Error
					go func() {
						watch.Error <- watcher.ErrWatchedFileDeleted
					}()
				} else {
					pidstr, ts := GetPidTS()
					dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB resumed watching %s", dbname))
				}
			case <-watch.Closed:
				pidstr, ts := GetPidTS()
				dblogger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher is closed without restarting %s", dbname))
				return
			}
		}
	}()

	err = watch.Add(dbname)
	if err != nil {
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher error adding file %s,  %v", dbname, err))
	}

	err = watch.Start(time.Second * 3) // This call blocks until the go routine completes via watch.Closed
	if err != nil {
		pidstr, ts := GetPidTS()
		dblogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher failed to start %v", err))
		return c.ErrF("grpcsig whitelist DB watcher failed to start: %v", err)
	}

	pidstr, ts = GetPidTS()
	dblogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig whitelist DB watcher closed on %s", dbname))
	return nil
}

// DBRestart - restart whitelist DB via BoltDB api through a close and open cycle.
func DBRestart(dbname string, dblog clog.Logger) *c.Err {
	pidstr, ts := GetPidTS()
	dblog.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig BoltDB restart %s", dbname))
	FiniPubKeyLookup()
	err := InitPubKeyLookup(dbname, dblog)
	if err != nil {
		msg := fmt.Sprintf("cannot restart grpcsig BoltDB database %s", dbname)
		dblog.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%s", msg)
	}
	return nil
}
