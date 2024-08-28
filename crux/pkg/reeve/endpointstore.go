// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// Reeve - enpdoint local storage

package reeve

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/muck"
)

type endpointMap map[string]pb.EndpointInfo
type uuidMap map[string]string

// Endpoints data is organized in-memory into maps with key value Localuuid
type endpointState struct {
	lock         *sync.RWMutex
	pending      endpointMap
	completed    endpointMap
	failed       endpointMap
	toremoteuuid uuidMap
	tolocaluuid  uuidMap
}

// ReeveEvents - internal pointer to our event loop
var ReeveEvents *Ingest

// eps - internal pointer to our storage maps
var eps endpointState

const pendingName = "pending.json"
const completedName = "completed.json"
const failedName = "failed.json"
const tolocalName = "tolocal.json"
const toremoteName = "toremote.json"

// endpointsLocalIni - starts local endpoint information storage maps
func endpointsLocalIni() *c.Err {
	// initialize maps
	eps = endpointState{}
	eps.lock = new(sync.RWMutex)
	eps.pending = make(endpointMap)
	eps.completed = make(endpointMap)
	eps.failed = make(endpointMap)
	eps.toremoteuuid = make(uuidMap)
	eps.tolocaluuid = make(uuidMap)
	return LoadEndpoints()
}

// EndpointPending - adds a pending EndpointInfo to local map, storage
func EndpointPending(ep pb.EndpointInfo, localuuid string) *c.Err {
	eps.lock.Lock()
	defer eps.lock.Unlock()
	// Add to the map
	eps.pending[localuuid] = ep
	// Checkpoint it
	serr := SaveEndpoints()
	if serr != nil {
		return serr
	}
	return nil
}

// EndpointCompleted - adds a completed (i.e. steward recieved this) EndpointInfo
// to local map, storage, removes it from pending. Endpoints on this list will
// be in steward's registrydb. Propogation of this endpoint to other nodes is
// not indicated by items on this list.
func EndpointCompleted(ep pb.EndpointInfo, localuuid string, remoteuuid string) *c.Err {
	if len(localuuid) == 0 || len(remoteuuid) == 0 {
		return c.ErrF("local and remote uuids not provided")
	}
	eps.lock.Lock()
	defer eps.lock.Unlock()
	// Add to the completed map
	eps.completed[localuuid] = ep
	// Remove from pending map
	if _, ok := eps.pending[localuuid]; ok {
		delete(eps.pending, localuuid)
	}
	// Add to the toremoteuuid map
	eps.toremoteuuid[localuuid] = remoteuuid
	// Add to the tolocaluuid map
	eps.tolocaluuid[remoteuuid] = localuuid
	// Checkpoint it
	serr := SaveEndpoints()
	if serr != nil {
		return serr
	}
	return nil
}

// EndpointFailed - adds an endpoint to the failed list, removes it from pending
// these will be outright rejected by steward and not processed further.
func EndpointFailed(ep pb.EndpointInfo, localuuid string, remoteuuid string) *c.Err {
	if len(localuuid) == 0 {
		return c.ErrF("local uuid not provided")
	}
	eps.lock.Lock()
	defer eps.lock.Unlock()
	// Add to the failed map
	eps.failed[localuuid] = ep
	// Remove it from pending map
	if _, ok := eps.pending[localuuid]; ok {
		delete(eps.pending, localuuid)
	}
	// If the failure came from the remote end, store its remoteuuid
	if len(remoteuuid) > 0 {
		// Add to the toremoteuuid map
		eps.toremoteuuid[localuuid] = remoteuuid
		// Add to the tolocaluuid map
		eps.tolocaluuid[remoteuuid] = localuuid
	}
	// Checkpoint it
	serr := SaveEndpoints()
	if serr != nil {
		return serr
	}
	return nil
}

// LoadEndpoints - On restart, reloads data from the Endpoint maps
func LoadEndpoints() *c.Err {
	// Don't read & throw errors if they don't exist
	dir := muck.EndpointDir()
	var pexists, cexists, fexists, rexists, lexists bool
	if _, err := os.Stat(dir + "/" + pendingName); !os.IsNotExist(err) {
		pexists = true
	}
	if _, err := os.Stat(dir + "/" + completedName); !os.IsNotExist(err) {
		cexists = true
	}
	if _, err := os.Stat(dir + "/" + failedName); !os.IsNotExist(err) {
		fexists = true
	}
	if _, err := os.Stat(dir + "/" + toremoteName); !os.IsNotExist(err) {
		rexists = true
	}
	if _, err := os.Stat(dir + "/" + tolocalName); !os.IsNotExist(err) {
		lexists = true
	}
	eps.lock.Lock()
	defer eps.lock.Unlock()

	var perr, cerr, ferr, rerr, lerr *c.Err
	if pexists {
		eps.pending, perr = readEndpoints(pendingName)
		if perr != nil {
			return perr
		}
	}
	if cexists {
		eps.completed, cerr = readEndpoints(completedName)
		if cerr != nil {
			return cerr
		}
	}
	if fexists {
		eps.failed, ferr = readEndpoints(failedName)
		if ferr != nil {
			return ferr
		}
	}
	if rexists {
		eps.toremoteuuid, rerr = readUUIDConvert(muck.EndpointDir() + "/" + toremoteName)
		if rerr != nil {
			return rerr
		}
	}
	if lexists {
		eps.tolocaluuid, lerr = readUUIDConvert(muck.EndpointDir() + "/" + tolocalName)
		if lerr != nil {
			return lerr
		}
	}
	return nil
}

// SaveEndpoints - Saves Endpoint maps as checkpoints.
// TOOD this can be sped up by dirty bits, writing only what has changed.
func SaveEndpoints() *c.Err {
	var perr, cerr, ferr, rerr, lerr *c.Err
	perr = writeEndpoints(&eps.pending, pendingName)
	if perr != nil {
		return perr
	}
	cerr = writeEndpoints(&eps.completed, completedName)
	if cerr != nil {
		return cerr
	}
	ferr = writeEndpoints(&eps.failed, failedName)
	if ferr != nil {
		return ferr
	}
	rerr = writeUUIDConvert(&eps.tolocaluuid, muck.EndpointDir()+"/"+tolocalName)
	if rerr != nil {
		return rerr
	}
	lerr = writeUUIDConvert(&eps.toremoteuuid, muck.EndpointDir()+"/"+toremoteName)
	if lerr != nil {
		return lerr
	}
	return nil
}

// writeEndpoints - writes an endpointMap to a json file.
func writeEndpoints(epsList *endpointMap, filename string) *c.Err {
	jsonstr, jerr := json.Marshal(epsList)
	if jerr != nil {
		return c.ErrF("json marshal error with endpointMap : %v", jerr)
	}
	path := muck.EndpointDir() + "/" + filename
	jsonstrlf := string(jsonstr) + "\n"
	ferr := ioutil.WriteFile(path, []byte(jsonstrlf), 0600)
	if ferr != nil {
		return c.ErrF("failed to write %s - %v", path, ferr)
	}
	return nil
}

// readEndpoints - reads an endpointMap from json file
func readEndpoints(filename string) (endpointMap, *c.Err) {
	path := muck.EndpointDir() + "/" + filename
	jsonstrlf, rerr := ioutil.ReadFile(path)
	if rerr != nil {
		return endpointMap{}, c.ErrF("failed to read %s - %v", path, rerr)
	}
	jsonstr := jsonstrlf[:len(jsonstrlf)-1]
	var emp endpointMap
	jerr := json.Unmarshal(jsonstr, &emp)
	if jerr != nil {
		return endpointMap{}, c.ErrF("failed to json unmarshall %s - %v", path, jerr)
	}
	return emp, nil
}

// writeUUIDConvert - write a uuid convert map to checkpoint json file
func writeUUIDConvert(uuidList *uuidMap, path string) *c.Err {
	jsonstr, jerr := json.Marshal(uuidList)
	if jerr != nil {
		return c.ErrF("json marshal error with uuidMap : %v", jerr)
	}
	jsonstrlf := string(jsonstr) + "\n"
	ferr := ioutil.WriteFile(path, []byte(jsonstrlf), 0600)
	if ferr != nil {
		return c.ErrF("failed to write %s - %v", path, ferr)
	}
	return nil
}

// readUUIDConvert - read a uuid convert map from checkpoint json file.
func readUUIDConvert(path string) (uuidMap, *c.Err) {
	jsonstrlf, rerr := ioutil.ReadFile(path)
	if rerr != nil {
		return uuidMap{}, c.ErrF("failed to read %s - %v", path, rerr)
	}
	jsonstr := jsonstrlf[:len(jsonstrlf)-1]
	var uids uuidMap
	jerr := json.Unmarshal(jsonstr, &uids)
	if jerr != nil {
		return uuidMap{}, c.ErrF("failed to json unmarshall %s - %v", path, jerr)
	}
	return uids, nil
}
