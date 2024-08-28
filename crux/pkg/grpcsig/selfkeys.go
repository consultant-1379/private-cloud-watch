// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/muck"
)

var (
	pidstr     string
	sshkeygen  string
	sshadd     string
	keyfile    string
	selfkeyfp  string
	selfPubKey *PubKeyT
)

// This section maintains "self" keys so that a single executable (by PID) can
// send and receive messages to and from different gRPC interfaces running concurrently
// (e.g plugins). It sets a seed public/private key pair for a process on startup,
// adds it to the ssh-agent and provides it for adding to a BoltDB public key database

// Defaults when string args are ""
const cryptotype string = "ed25519"
const dirdelim = "/"

// Selfkeyprefix is used by pubkeydb.go in this package for cleanup of transient keys
const Selfkeyprefix = "self-key-"

// InitSelfSSHKeys - sets up self keys in directory provided in muck.InitMuck()
// A passphrase is not supported as an argument for self-keys, hard to automate ssh-add with one present.
// Pass debug as true, to print command line issued for ssh-keygen, key file path, ssh-add
func InitSelfSSHKeys(debug bool) *c.Err {
	if !muck.IsMuckInited() {
		return c.ErrF("need to call muck.InitMuck() first")
	}
	passphrase := ""
	if len(keyfile) != 0 && selfPubKey != nil {
		return nil // Already initialized
	}
	// Set our self PID
	setSelfPidstr()
	// Pre-Flight check for ssh tools in the path
	cerr := SSHKeygenExists()
	if cerr != nil {
		return cerr
	}
	cerr = sshAddExists()
	if cerr != nil {
		return cerr
	}
	// ssh-agent has to be running before this PID started up
	_, aerr := TryAgent()
	if aerr != nil {
		return aerr
	}
	// Make a default ed25519 key pair in the above noted directory
	var jerr *c.Err
	selfPubKey, jerr = newSelfKeyPair("", passphrase, debug)
	if jerr != nil {
		return jerr
	}
	keypath := muck.Dir() + dirdelim + keyfile
	if debug {
		fmt.Printf("keypath: %s\n", keypath)
	}
	cerr = AddKeyToAgent(keypath, debug)
	if cerr != nil {
		return cerr
	}
	return nil
}

// GetSelfPidStr - Gets our PID as a string
func GetSelfPidStr() string {
	return pidstr
}

// GetSelfPubKey Gets our self public-key as a PubKeyT (i.e. the BoltDB json format)
func GetSelfPubKey() *PubKeyT {
	return selfPubKey
}

// GetSelfFingerprint - Gets our raw fingerprint for ssh-agent lookup
func GetSelfFingerprint() string {
	return selfkeyfp
}

// FiniSelfSSHKeys - removes the current self-key from the ssh-agent, and deletes
// the two key files. Errors returned if any combination of these fails
// no going back after it is done.
func FiniSelfSSHKeys(debug bool) *c.Err {
	keypath := muck.Dir() + dirdelim + keyfile
	selfPubKey = nil
	selfkeyfp = ""
	status := ""
	rerr := RemoveKeyFromAgent(keypath, debug) // must do first
	cerr := SelfKeyFileExists()
	if debug {
		println(cerr.Error())
	}
	if cerr == nil {
		// Remove the files
		err := os.Remove(keypath)
		if err != nil {
			status = fmt.Sprintf("failed to remove private key file %s - %v", keypath, err)
		}
		if debug {
			fmt.Printf("Removed %s\n", keypath)
		}
		err = os.Remove(keypath + ".pub")
		if err != nil {
			status = status + fmt.Sprintf(" failed to remove public key file %s - %v", keypath+".pub", err)
		}
		if debug {
			fmt.Printf("Removed %s\n", keypath+".pub")
		}
	}
	keyfile = ""
	if rerr != nil && status != "" {
		return c.ErrF("%v %s\n", rerr, status)
	}
	if rerr != nil {
		return rerr
	}
	if status != "" {
		return c.ErrF("%s\n", status)
	}
	return nil
}

// setSelfPidstr - finds the PID and sets Pidstr - our process ID is "self"
func setSelfPidstr() {
	pidstr = strconv.Itoa(os.Getpid())
}

// SSHKeygenExists - looks for ssh-keygen in the path, nil if ok, error if not
func SSHKeygenExists() *c.Err {
	var err error
	sshkeygen, err = exec.LookPath("ssh-keygen")
	if err != nil {
		return c.ErrF("system missing ssh-keygen, %v", err)
	}
	return nil
}

// sshAddExists - looks for ssh-add in the path, nil if ok, error if not.
func sshAddExists() *c.Err {
	var err error
	sshadd, err = exec.LookPath("ssh-add")
	if err != nil {
		return c.ErrF("system missing ssh-add, %v", err)
	}
	return nil
}

// SelfKeyFileExists - returns nil if all good, if file does not
// exist, it unsets keyfile
func SelfKeyFileExists() *c.Err {
	if len(keyfile) == 0 {
		return c.ErrF("keyfile not yet generated")
	}

	// Check private key file
	keypath := muck.Dir() + dirdelim + keyfile
	_, err := os.Stat(keypath)
	if err != nil {
		keyfile = ""
		return c.ErrF("system missing private key file %s, %v", keyfile, err)
	}
	// Check public key file
	_, err = os.Stat(keypath + ".pub")
	if err != nil {
		keyfile = ""
		return c.ErrF("system missing public key file %s, %v", keyfile, err)
	}
	return nil
}

// newSelfKeyPair - sets a PID based self keypair for this process
// Default ctype (cryto type) is ed25519 when "" is passed as first argument
// or supply "ecdsa" or "rsa"
// Provide a passphrase for keyfile encryption or "" for none.
// If it fails, keyfile is unset.
// Pass debug as true - to print command line equivalent
// Returns a JSON formatted PubKeyT matching our BoltDB system
func newSelfKeyPair(ctype string, passphrase string, debug bool) (*PubKeyT, *c.Err) {
	setSelfPidstr()
	if len(ctype) == 0 {
		ctype = cryptotype
	}
	keyfile = Selfkeyprefix + pidstr + "-" + ctype
	keypath := muck.Dir() + dirdelim + keyfile
	pubkey, err := NewKeyPair(ctype, keypath, passphrase, keyfile, debug)
	if err != nil {
		keyfile = ""
		selfkeyfp = ""
		return nil, err
	}
	// Fix up pubkey to keyID /service/principalid/keys/fp form
	selfkeyfp = pubkey.KeyID // raw fingerprint returned by NewKeyPair
	pubkey.Name = "self"
	pubkey.Service = "self"
	pubkey.KeyID = fmt.Sprintf("/%s/%s/keys/%s", "self", "self", selfkeyfp)
	return pubkey, nil
}
