// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package reeve

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pborman/uuid"

	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/muck"
)

const dirdelim = "/"

// GetCurrentPubkeys - returns an array of grpcsig.PubKeyT that are
// active in .muck/current
// TODO - limit this to the specific keys required in registration process...
func GetCurrentPubkeys(debug bool) ([]grpcsig.PubKeyT, *c.Err) {
	if debug {
		fmt.Printf("Current:\n")
	}
	pubkeys := []grpcsig.PubKeyT{}
	currentdir := muck.CurrentKeysDir()
	pubfiles, err := walkDir(currentdir)
	if err != nil {
		return []grpcsig.PubKeyT{}, err
	}
	prefixlen := len(currentdir)
	if currentdir[:2] == "./" {
		prefixlen = prefixlen - 2
	}
	for _, keyfile := range pubfiles {
		// remove currentdir prefix
		// remove .pub suffix
		sfx := len(keyfile) - 4
		// fmt.Printf("keyfile:[%s]\nkeyfile[prefixlen:sfx][%s]\n", keyfile, keyfile[prefixlen:sfx])
		keyID := keyfile[prefixlen:sfx]
		if debug {
			fmt.Printf("  %s\n", keyID)
		}
		// load the raw public key file linked to it
		pubkey := &grpcsig.PubKeyT{}
		pubkey, err = loadLinkedPubKey(keyfile, debug)
		if err != nil {
			return []grpcsig.PubKeyT{}, c.ErrF("error - failed to load current pubkey %s, %v", keyfile, err)
		}
		terms := strings.Split(keyID, "/")
		if len(terms) < 4 {
			return []grpcsig.PubKeyT{}, c.ErrF("error - failed to parse out fields from keyID %s", keyID)
		}
		pubkey.Service = terms[1]
		pubkey.Name = terms[2]
		// TODO double check filesystem fingerprint is as recalculated from actual public key
		pubkey.KeyID = keyID
		pubkeys = append(pubkeys, *pubkey)
	}
	return pubkeys, nil
}

// GetDeprecatedPubkeys - Looks in the ./muck/deprecated for local keyIDs that
// have been marked deprecated, and packages their keyIDs as an array of strings
func GetDeprecatedPubkeys() ([]string, *c.Err) {
	// fmt.Printf("Deprecated:\n")
	deprecdir := muck.DeprecKeysDir()
	keys, err := walkDir(deprecdir)
	if err != nil {
		return []string{}, err
	}
	prefixlen := len(deprecdir) - 2
	trimmedkeys := []string{}
	for _, key := range keys {
		// remove deprecdir prefix
		// and .pub suffix
		sfx := len(key) - 4
		keyID := key[prefixlen:sfx]
		trimmedkeys = append(trimmedkeys, keyID)
	}
	return trimmedkeys, nil
}

// loadLinkedPubKey - loads a linked Public Key file into a nascent grpcsig.PubKeyT
// where Fingerprint holds only the raw fingerprint.
// caller must fill in Name, Service, update Fingeprint to full keyID lookup string.
func loadLinkedPubKey(path string, debug bool) (*grpcsig.PubKeyT, *c.Err) {
	linkfi, perr := os.Lstat(path)
	if perr != nil {
		return nil, c.ErrF("error cannot load public key file, bad symbolic link %v %v", path, perr)
	}
	destfile := ""
	if linkfi.Mode()&os.ModeSymlink != 0 {
		destfile, perr = os.Readlink(path)
		if perr != nil {
			return nil, c.ErrF("error cannot load public key file, failed to follow symbolic link %v %v", path, perr)
		}
	}
	destfi, perr := os.Stat(destfile)
	if perr != nil {
		return nil, c.ErrF("error cannot load public key file, cannot stat %s %v", destfile, perr)
	}
	if debug {
		fmt.Printf("%s -> %s\n", path, destfi.Name())
	}
	return grpcsig.LoadAndValidateGenericPubKey(muck.AllKeysDir() + dirdelim + destfi.Name())
}

/* TODO - reconsider when implementing key rolling
func GetKilledPubkeys() ([]string, *c.Err) {
	// fmt.Printf("Killed:\n")
	killeddir := muck.getKilledKeysDir()
	keys, err := walkDir(killeddir)
	if err != nil {
		return []string{}, err
	}
	prefixlen := len(killeddir) - 2
	trimmedkeys := []string{}
	for _, key := range keys {
		// remove killeddir prefix, .pub suffix
		trimmedkeys = append(trimmedkeys, key[prefixlen])
	}
	for _, k := range trimmedkeys {
		fmt.Println("["+k+"]")
	}
	return trimmedkeys, nil
}
*/

func walkDir(startpath string) ([]string, *c.Err) {
	files := []string{}
	rerr := filepath.Walk(startpath, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".pub") {
			files = append(files, path)
		}
		return err
	})
	if rerr != nil {
		return []string{}, c.ErrF("error getting directory listing for %s: %v", startpath, rerr)
	}
	return files, nil
}

// InitReeveKeys - initializes the reeve Public/Private keys
func InitReeveKeys(ctype string, debug bool) (*grpcsig.PubKeyT, *c.Err) {
	// Set up the initial Reeve public/private key
	// or grab the current one if it exists
	reevekey, kerr := setReeveSvcKeys(ctype, debug)
	if kerr != nil {
		return nil, kerr
	}
	// Add the reeve key to the local Agent
	err := AddCurrentKeyToAgent(reevekey.KeyID, debug)
	if err != nil {
		return nil, err
	}
	// return the grpcsig.PubKeyT to pass to the Fulcrum.
	// even if it is the one we already found.
	if debug {
		fmt.Printf("Reeve Public Key: %v\n", reevekey)
	}

	return reevekey, nil
}

// setReeveSvcKeys - either generates new keys or uses ones already
// established by a previous run in ./muck/current/reeve/principalid/keys/fingerprint
// linked file, if it exists
func setReeveSvcKeys(ctype string, debug bool) (*grpcsig.PubKeyT, *c.Err) {
	return MakeServiceKeys(ReeveRev, ctype, debug)
}

// MakeServiceKeys - initializes a services's key directory in .muck if not set,
// makes an initial key - or returns current key found.
// Gets current grpcsig.PubKeyT for service, if missing, inits the keypair for the service
func MakeServiceKeys(service string, ctype string, debug bool) (*grpcsig.PubKeyT, *c.Err) {
	pid, err := muck.Principal()
	if err != nil {
		return nil, err
	}
	verr := muck.CheckName(service)
	if verr != nil {
		return nil, verr
	}
	// Do we have a .muck/current/service ?
	svcdir := muck.CurrentKeysDir() + dirdelim + service
	if _, err := os.Stat(svcdir); os.IsNotExist(err) {
		err := os.MkdirAll(svcdir, 0700)
		if err != nil {
			return nil, c.ErrF("unable to mkdir %s %v", svcdir, err)
		}
	}
	// Do we have a .muck/current/service/serviceid/keys ?
	mysvcdir := svcdir + dirdelim + pid + dirdelim + "keys"
	if _, err := os.Stat(mysvcdir); os.IsNotExist(err) {
		err := os.MkdirAll(mysvcdir, 0700)
		if err != nil {
			return nil, c.ErrF("unable to mkdir -p  %s %v", mysvcdir, err)
		}
	}
	// Do we have a service key for this already?
	files, ferr := ioutil.ReadDir(mysvcdir)
	if ferr != nil {
		return nil, c.ErrF("unable to get diretory contents for %s %v", mysvcdir, ferr)
	}
	filecount := len(files)
	if filecount == 0 {
		if debug {
			fmt.Printf("Creating new key pair for service %s\n", service)
		}
		realpath, pk, err := newSvcKeyPair(service, ctype, "", debug)
		if err != nil {
			return nil, c.ErrF("unable to make a new key pair for service  %s in %s: %v", service, mysvcdir, err)
		}
		// prepend the currentdir to the PubKey.KeyID query string for the symlink path
		symlink := muck.CurrentKeysDir() + dirdelim + pk.KeyID
		// Symlink to public key
		lerr := os.Symlink(realpath, symlink)
		if lerr != nil {
			return nil, c.ErrF("error in symlink creating %s -> %s : %v", symlink, realpath, lerr)
		}
		// Symlink to private key
		lerr = os.Symlink(realpath+".pub", symlink+".pub")
		if lerr != nil {
			return nil, c.ErrF("error in symlink creating %s -> %s : %v", symlink, realpath, lerr)
		}
		return pk, nil
	}
	// service key symlinks (or something) is in this directory.
	// should be only 2 symlinks, one ending in .pub if present
	if filecount > 2 {
		derr := deprecateOldestKeys(mysvcdir, files)
		if derr != nil {
			return nil, derr
		}
		// reread the directory and file count
		files, ferr = ioutil.ReadDir(mysvcdir)
		if ferr != nil {
			return nil, c.ErrF("error - unable to get diretory contents for %s %v", mysvcdir, ferr)
		}
		filecount = len(files)
	}
	if filecount != 2 {
		return nil, c.ErrF("error - something is wrong in key directory, can't deprecate keys properly %s,  %v", mysvcdir, files)
	}
	// find the .pub file, and follow the link to fill in the grpcsig.PubKeyT
	for _, file := range files { // os.FileInfo
		if debug {
			fmt.Printf("Checking possible %s key file %s ", service, file.Name())
			/*
				  	fmt.Println(file.Size())
					fmt.Println(file.Mode())
					fmt.Println(file.ModTime())
					fmt.Println(file.IsDir())
			*/
		}
		linkcheck := mysvcdir + dirdelim + file.Name()
		linkfi, perr := os.Lstat(linkcheck)
		if perr != nil {
			return nil, c.ErrF("error following symbolic link %v %v", linkcheck, perr)
		}

		if linkfi.Mode()&os.ModeSymlink != 0 {
			destfile, perr := os.Readlink(linkcheck)
			if perr != nil {
				return nil, c.ErrF("error following symbolic link %v %v", linkcheck, perr)
			}
			destfi, perr := os.Stat(destfile)
			if perr != nil {
				return nil, c.ErrF("error getting stat for %s %v", destfile, perr)
			}
			if debug {
				fmt.Printf("-> %s\n", destfi.Name())
				/*
					fmt.Println(destfi.Size())
					fmt.Println(destfi.Mode())
					fmt.Println(destfi.ModTime())
					fmt.Println(destfi.IsDir())
				*/
			}
			if strings.HasSuffix(file.Name(), ".pub") && strings.HasSuffix(destfi.Name(), ".pub") {
				pk, err := grpcsig.LoadAndValidateGenericPubKey(muck.AllKeysDir() + dirdelim + destfi.Name())
				if err != nil {
					return nil, err
				}
				// Fix up grpcsig.PubKeyT to /service/serviceid/keys/fingrprint  form
				_ = grpcsig.MakeLocalKeyID(service, pk)
				return pk, nil
			}
		}
	}
	return nil, c.ErrF("Failed to get current public key for service %s", service)
}

// deperecateOldestKeys takes a dir with symlinks to multiple keypairs,
// and moves the symlinks from currentdir to deprecdir
func deprecateOldestKeys(dir string, files []os.FileInfo) *c.Err {
	return c.ErrF("TODO - Not yet implemented")
}

// newSvcKeyPair -
// Default ctype (cryto type) is ed25519 when "" is passed as first argument
// or supply "ecdsa" or "rsa"
// Provide a passphrase for keyfile encryption or "" for none.
// Pass debug as true - to print command line equivalent
// Returns a JSON formatted grpcsig.PubKeyT matching our BoltDB system
// and the path to where the UUID named file pair landed
func newSvcKeyPair(service string, ctype string, passphrase string, debug bool) (string, *grpcsig.PubKeyT, *c.Err) {
	if len(ctype) == 0 {
		ctype = "ed25519"
	}
	keyfile := uuid.NewUUID().String() // This returns a v1 UUID with nodename and time encoded
	keypath := muck.AllKeysDir() + dirdelim + keyfile
	pid, _ := muck.Principal()
	pubkey, err := grpcsig.NewKeyPair(ctype, keypath, passphrase, pid+"-"+ctype, debug)
	if err != nil {
		return "", nil, err
	}
	// Fix up pubkey to /service/username/keys prefixed form
	_ = grpcsig.MakeLocalKeyID(service, pubkey)
	return keypath, pubkey, nil
}

// AddCurrentKeyToAgent adds a key in current to ssh-agent
func AddCurrentKeyToAgent(lookup string, debug bool) *c.Err {
	return setLinkedKeyInAgent(true, muck.CurrentKeysDir()+lookup, debug)
}

// RemoveCurrentKeyFromAgent removes a key in current from ssh-agent
func RemoveCurrentKeyFromAgent(lookup string, debug bool) *c.Err {
	return setLinkedKeyInAgent(false, muck.CurrentKeysDir()+lookup, debug)
}

/*

TODO
func addDeprecatedKeyToAgent(lookup string, debug bool) *c.Err {
	return setLinkedKeyInAgent(true, deprecdir+lookup,debug)
}

func removeDeprecatedKeyFromAgent(lookup string, debug bool) *c.Err {
	return setLinkedKeyInAgent(false, deprecdir+lookup,debug)
}

TODO - this needs to be cleaned up on exit..!!! we keep accumulating new
keys in ssh-agent
func removeCurrentKeyFromAgent(lookup string, debug bool) *c.Err {
	return RemoveLinkedKeyFromAgent(currentdir+lookup,debug)
}


*/

// setLinkedKeyInAgent - follows the specified symbolic link to the private key file,
// then calls AddKeyToAgent if true, RemoveKeyFromAgent if false, with the result
func setLinkedKeyInAgent(add bool, path string, debug bool) *c.Err {
	linkfi, perr := os.Lstat(path)
	if perr != nil {
		return c.ErrF("error cannot find key to set agent, bad symbolic link %v %v", path, perr)
	}
	destfile := ""
	if linkfi.Mode()&os.ModeSymlink != 0 {
		destfile, perr = os.Readlink(path)
		if perr != nil {
			return c.ErrF("error cannot find key to set agent, failed to follow symbolic link %v %v", path, perr)
		}
	}
	destfi, perr := os.Stat(destfile)
	if perr != nil {
		return c.ErrF("error cannot find key to set agent, cannot stat %s %v", destfile, perr)
	}
	if debug {
		fmt.Printf("%s -> %s\n", path, destfi.Name())
	}
	if add {
		return grpcsig.AddKeyToAgent(muck.AllKeysDir()+dirdelim+destfi.Name(), debug)
	}
	return grpcsig.RemoveKeyFromAgent(muck.AllKeysDir()+dirdelim+destfi.Name(), debug)
}

/*

Key Deprecation
Options
By time since last timestamp?
By last N keys?
By rate


Key Deprecation Cycle

Find keys to deprecate. ??
Does .muck/deprecated exist?
Make symlinks in .muck/deprecated/servicerev/serviceid/fingerprint
Make replacement - new keys in all
get fingerprint
Make new symlinks in .muck/current/servicerev/serviceid/fingerprint
Copy /servicerev/serviceid/fingerprint string and append to a file in .muck/killed/killedlist


*/
