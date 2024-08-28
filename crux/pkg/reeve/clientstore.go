// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// Reeve - enpdoint local storage

package reeve

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/pborman/uuid"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
)

type clientMap map[string]pb.ClientInfo

// Clients data is organized in-memory into maps with key value Localuuid
type clientState struct {
	lock         *sync.RWMutex
	pending      clientMap
	completed    clientMap
	failed       clientMap
	toremoteuuid uuidMap
	tolocaluuid  uuidMap
}

// cls - internal pointer to our storage maps
var cls clientState

// clientsLocalIni - starts local client information storage maps
func clientsLocalIni() *c.Err {
	// initialize maps
	cls = clientState{}
	cls.lock = new(sync.RWMutex)
	cls.pending = make(clientMap)
	cls.completed = make(clientMap)
	cls.failed = make(clientMap)
	cls.toremoteuuid = make(uuidMap)
	cls.tolocaluuid = make(uuidMap)
	return LoadClients()
}

// ClientPending - adds a pending ClientInfo to local map, storage
func ClientPending(cl pb.ClientInfo, localuuid string) *c.Err {
	cls.lock.Lock()
	defer cls.lock.Unlock()
	// Add to the map
	cls.pending[localuuid] = cl
	// Checkpoint it
	serr := SaveClients()
	if serr != nil {
		return serr
	}
	return nil
}

// LocalClientExists - hashes KeyID, looks up in pending or completed,
// to see if this client exists in our local list, or if it
// is the self-signer
func LocalClientExists(kid *idutils.KeyIDT) bool {
	if kid == nil {
		return false
	}
	cls.lock.Lock()
	defer cls.lock.Unlock()
	uid := uuid.NewMD5(uuid.NIL, []byte(kid.String()))
	if _, ok := cls.pending[uid.String()]; ok {
		return true
	}
	if _, ok := cls.completed[uid.String()]; ok {
		return true
	}
	// Is it the self-key?
	ReeveState.imp.Logger.Log(nil, "LCE: %s vs %s", kid.String(), ReeveState.selfsigner.KeyID.String())
	if kid.String() == ReeveState.selfsigner.KeyID.String() {
		return true
	}
	pidstr, ts := grpcsig.GetPidTS()
	ReeveState.imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("reeve LocalClientExists observes non-local client attempt from %s", kid.String()))
	return false
}

// ClientCompleted - adds a completed (i.e. steward recieved this) ClientInfo
// to local map, storage, removes it from pending. Clients on this list will
// be in steward's registrydb. Propogation of this client to other nodes is
// not indicated by items on this list.
func ClientCompleted(cl pb.ClientInfo, localuuid string, remoteuuid string) *c.Err {
	if len(localuuid) == 0 || len(remoteuuid) == 0 {
		return c.ErrF("local and remote uuids not provided")
	}
	cls.lock.Lock()
	defer cls.lock.Unlock()
	// Add to the completed map
	cls.completed[localuuid] = cl
	// Remove from pending map
	if _, ok := cls.pending[localuuid]; ok {
		delete(cls.pending, localuuid)
	}
	// Add to the toremoteuuid map
	cls.toremoteuuid[localuuid] = remoteuuid
	// Add to the tolocaluuid map
	cls.tolocaluuid[remoteuuid] = localuuid
	// Checkpoint it
	serr := SaveClients()
	if serr != nil {
		return serr
	}
	return nil
}

// ClientFailed - adds an client to the failed list, removes it from pending
// these will be outright rejected by steward and not processed further.
func ClientFailed(cl pb.ClientInfo, localuuid string, remoteuuid string) *c.Err {
	if len(localuuid) == 0 {
		return c.ErrF("local uuid not provided")
	}
	cls.lock.Lock()
	defer cls.lock.Unlock()
	// Add to the failed map
	cls.failed[localuuid] = cl
	// Remove it from pending map
	if _, ok := cls.pending[localuuid]; ok {
		delete(cls.pending, localuuid)
	}
	// If the failure came from the remote end, store its remoteuuid
	if len(remoteuuid) > 0 {
		// Add to the toremoteuuid map
		cls.toremoteuuid[localuuid] = remoteuuid
		// Add to the tolocaluuid map
		cls.tolocaluuid[remoteuuid] = localuuid
	}
	// Checkpoint it
	serr := SaveClients()
	if serr != nil {
		return serr
	}
	return nil
}

// LoadClients - On restart, reloads data from the Client maps
func LoadClients() *c.Err {
	// Don't read & throw errors if they don't exist
	dir := muck.ClientDir()
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

	cls.lock.Lock()
	defer cls.lock.Unlock()

	var perr, cerr, ferr, rerr, lerr *c.Err
	if pexists {
		cls.pending, perr = readClients(pendingName)
		if perr != nil {
			return perr
		}
	}
	if cexists {
		cls.completed, cerr = readClients(completedName)
		if cerr != nil {
			return cerr
		}
	}
	if fexists {
		cls.failed, ferr = readClients(failedName)
		if ferr != nil {
			return ferr
		}
	}
	if rexists {
		cls.toremoteuuid, rerr = readUUIDConvert(muck.ClientDir() + "/" + toremoteName)
		if rerr != nil {
			return rerr
		}
	}
	if lexists {
		cls.tolocaluuid, lerr = readUUIDConvert(muck.ClientDir() + "/" + tolocalName)
		if lerr != nil {
			return lerr
		}
	}
	return nil
}

// SaveClients - Saves Client maps as checkpoints.
// TOOD this can be sped up by dirty bits, writing only what has changed.
func SaveClients() *c.Err {
	var perr, cerr, ferr, rerr, lerr *c.Err
	perr = writeClients(&cls.pending, pendingName)
	if perr != nil {
		return perr
	}
	cerr = writeClients(&cls.completed, completedName)
	if cerr != nil {
		return cerr
	}
	ferr = writeClients(&cls.failed, failedName)
	if ferr != nil {
		return ferr
	}
	rerr = writeUUIDConvert(&cls.tolocaluuid, muck.ClientDir()+"/"+tolocalName)
	if rerr != nil {
		return rerr
	}
	lerr = writeUUIDConvert(&cls.toremoteuuid, muck.ClientDir()+"/"+toremoteName)
	if lerr != nil {
		return lerr
	}
	return nil
}

// writeClients - writes a clientMap to a json file.
func writeClients(clsList *clientMap, filename string) *c.Err {
	jsonstr, jerr := json.Marshal(clsList)
	if jerr != nil {
		return c.ErrF("json marshal error with clientMap : %v", jerr)
	}
	path := muck.ClientDir() + "/" + filename
	jsonstrlf := string(jsonstr) + "\n"
	ferr := ioutil.WriteFile(path, []byte(jsonstrlf), 0600)
	if ferr != nil {
		return c.ErrF("failed to write %s - %v", path, ferr)
	}
	return nil
}

// readClients - reads a clientMap from json file
func readClients(filename string) (clientMap, *c.Err) {
	path := muck.ClientDir() + "/" + filename
	jsonstrlf, rerr := ioutil.ReadFile(path)
	if rerr != nil {
		return clientMap{}, c.ErrF("failed to read %s - %v", path, rerr)
	}
	jsonstr := jsonstrlf[:len(jsonstrlf)-1]
	var emp clientMap
	jerr := json.Unmarshal(jsonstr, &emp)
	if jerr != nil {
		return clientMap{}, c.ErrF("failed to json unmarshall %s - %v", path, jerr)
	}
	return emp, nil
}
