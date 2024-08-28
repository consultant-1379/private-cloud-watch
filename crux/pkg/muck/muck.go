// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package muck

// Handles initialization of .muck filesystem
// which holds the persistent data for a fulcrum
// instance, including principal, and public/private
// key information.
// Provides/returns the persitent principal when called.
// cerr := InitMuck("","") in its simplest form.
// No key functions here, this just sets up key directories.

// Warning - Do not hand-edit entries stored in this filesystem.

/* Directory structure (Sept 2018)
.muck                                - entrypoint
.muck/principal                      - persistent fulcrum identifier (or a userid)
.muck/reeve/state                    - int32 - last state update from steward()
.muck/reeve/clients/keys/current/    - fingerprint named symlinks here (active keys)
.muck/reeve/clients/keys/deprcated   - fingerprint name symlinks, removed pubkeys (reachable)
.muck/reeve/clients/keys/killed/     - appended file of fingerprint names - keys actually wiped from all
.muck/reeve/clients/keys/all/        - uuid named public and private key files go here
.muck/reeve/endpoints/               - servicename/serviceAPI/servicerev/hash/pluginfile
.muck/pastiche/                      - pastiche stuff goes here
.muck/steward                        - steward stuff goes here
.muck/register/			     - registered steward info goes here, stewardnid, stewardnod, stewardkid
*/

import (
	"io/ioutil"
	"net/mail"
	"os"
	"regexp"

	"github.com/nats-io/nuid"

	c "github.com/erixzone/crux/pkg/crux"
)

var (
	muckdir     string
	principal   string
	muckkeys    bool
	reevedir    string
	pastichedir string
	registrydir string
	stewarddir  string
	reevekeys   string
	clients     string
	endpoints   string
	alldir      string
	currentdir  string
	deprecdir   string
	killeddir   string
	blobdir     string
)

const muckDirDefault string = ".muck"
const stewardKeyIDFile = "stewardkid"
const stewardNetIDFile = "stewardnid"
const stewardNodeIDFile = "stewardnod"
const principalFile = "principal"
const hordeFilename = "horde"
const reeveName = "reeve"
const registryName = "register"
const pasticheName = "pastiche"
const stewardName = "steward"
const dd = "/"

// InitMuck - if not already set, intializes .muck directory infrastructure
// and persistent principal (i.e. the fulcrum instance's unique userid)
// Pass ("","") arguments for default naming scheme where
//  dir gets muckDirDefault, principal (pid) gets an NUID (NATS unquie ID)
// If you pass in a dir, leave off trailing / delimiter
func InitMuck(dir string, pid string) *c.Err {
	if IsMuckInited() {
		return nil // muckdir already set
	}
	if len(pid) != 0 { // check provided principal value for directory name compatibility
		err := CheckName(pid)
		if err != nil {
			return c.ErrF("bad principal (pid argument) name %s supplied: %v", pid, err)
		}
	}
	derr := setMuckDir(dir)
	if derr != nil {
		return derr
	}
	serr := setMuckSubdirs()
	if serr != nil {
		return serr
	}
	werr := setPrincipal(pid)
	if werr != nil {
		return werr
	}
	return nil
}

// IsMuckInited - Check to see if muck has been initialized.
func IsMuckInited() bool {
	return (len(muckdir) != 0) && (len(principal) != 0) && muckkeys
}

// Dir - returns current directory of private ssh keys for services (aka .muck/)
func Dir() string {
	return muckdir
}

// BlobDir - returns current directory of pastiche blobs
func BlobDir() string {
	return blobdir
}

// StewardDir - returns current directory of steward databases - if present
func StewardDir() string {
	return stewarddir
}

// RegistryDir - returns current directory of register information - if present
func RegistryDir() string {
	return registrydir
}

// AllKeysDir - path to .muck/all - used for storing keys
func AllKeysDir() string {
	return alldir
}

// CurrentKeysDir - path to .muck/current - stores symlinks to keys
func CurrentKeysDir() string {
	return currentdir
}

// DeprecKeysDir - path to .muck/deprecated - stores symlinks to keys
func DeprecKeysDir() string {
	return deprecdir
}

// KilledKeysDir - path to .muck/killed - stores list of killed keys
func KilledKeysDir() string {
	return killeddir
}

// Principal - returns the persistent name for the runtime that launched Reeve,
// either as provided - or defaults to an NUID; stored in .muck/principal by default
func Principal() (string, *c.Err) {
	if principal != "" {
		return principal, nil
	}
	return "", c.ErrF("persistent principal not set")
}

// EndpointDir - path to .muck/reeve/endpoints - stores reeve registered endpoints on local node
func EndpointDir() string {
	return endpoints
}

// ClientDir - path to .muck/reeve/clients - stores reeve registered clients from local node
func ClientDir() string {
	return clients
}

// setMuckDir - creates local state/key directory - default dir argument "" gets you ./.muck
func setMuckDir(dir string) *c.Err {
	if len(dir) == 0 {
		dir = muckDirDefault
	}
	_, err := os.Stat(dir) // Does this directory exist?
	if err == nil {
		muckdir = dir
		return nil // we use it.
	}
	err = os.Mkdir(dir, 0700) // Otherwise we make it
	if err != nil {
		muckdir = "" // We got nothing
		return c.ErrF("unable to create local muck directory %s, %v\n", dir, err)
	}
	muckdir = dir
	return nil
}

// setMuckSubdirs - sets up the muck subdirectories for local key mananagement
func setMuckSubdirs() *c.Err {
	if muckdir == "" {
		return c.ErrF("muckdir not set")
	}
	if !muckkeys {
		reevedir = muckdir + dd + reeveName
		if _, err := os.Stat(reevedir); os.IsNotExist(err) {
			err := os.Mkdir(reevedir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", reevedir, err)
			}
		}
		registrydir = muckdir + dd + registryName
		if _, err := os.Stat(registrydir); os.IsNotExist(err) {
			err := os.Mkdir(registrydir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", registrydir, err)
			}
		}
		stewarddir = muckdir + dd + stewardName
		if _, err := os.Stat(stewarddir); os.IsNotExist(err) {
			err := os.Mkdir(stewarddir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", stewarddir, err)
			}
		}
		pastichedir = muckdir + dd + pasticheName
		if _, err := os.Stat(pastichedir); os.IsNotExist(err) {
			err := os.Mkdir(pastichedir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", pastichedir, err)
			}
		}
		clients = reevedir + dd + "clients"
		if _, err := os.Stat(clients); os.IsNotExist(err) {
			err := os.Mkdir(clients, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", clients, err)
			}
		}
		endpoints = reevedir + dd + "endpoints"
		if _, err := os.Stat(endpoints); os.IsNotExist(err) {
			err := os.Mkdir(endpoints, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", endpoints, err)
			}
		}
		reevekeys = clients + dd + "keys"
		if _, err := os.Stat(reevekeys); os.IsNotExist(err) {
			err := os.Mkdir(reevekeys, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", reevekeys, err)
			}
		}
		alldir = reevekeys + dd + "all"
		if _, err := os.Stat(alldir); os.IsNotExist(err) {
			err := os.Mkdir(alldir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", alldir, err)
			}
		}
		currentdir = reevekeys + dd + "current"
		if _, err := os.Stat(currentdir); os.IsNotExist(err) {
			err = os.Mkdir(currentdir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s %v", currentdir, err)
			}
		}
		deprecdir = reevekeys + dd + "deprecated"
		if _, err := os.Stat(deprecdir); os.IsNotExist(err) {
			err = os.Mkdir(deprecdir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s, %v", deprecdir, err)
			}
		}
		killeddir = reevekeys + dd + "killed"
		if _, err := os.Stat(killeddir); os.IsNotExist(err) {
			err = os.Mkdir(killeddir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s, %v", killeddir, err)
			}
		}
		blobdir = pastichedir + dd + "blob_dirs"
		if _, err := os.Stat(blobdir); os.IsNotExist(err) {
			err = os.Mkdir(blobdir, 0700)
			if err != nil {
				return c.ErrF("unable to mkdir %s, %v", blobdir, err)
			}
		}

		muckkeys = true
	}
	return nil
}

// HordeName - saves the hordename in persistent muck storage
// returns the string if saved properly
// If "" passed - returns the stored string if present
func HordeName(hordename string) string {
	hordefile := muckdir + dd + hordeFilename
	return ioStringFile(hordefile, hordename)
}

// StewardKeyID - saves the steward KeyID string in persistent muck storage
// returns the string if saved properly
// If "" passed - returns the stored string if present
func StewardKeyID(kid string) string {
	kidfile := registrydir + dd + stewardKeyIDFile
	return ioStringFile(kidfile, kid)
}

// StewardNetID - saves the steward NetID string in persistent muck storage
// returns the string if saved properly
// If "" passed - returns the stored string if present
func StewardNetID(nid string) string {
	nidfile := registrydir + dd + stewardNetIDFile
	return ioStringFile(nidfile, nid)
}

// StewardNodeID - saves the steward NodeID string in persistent muck storage
// returns the string if saved properly
// If "" passed - returns the stored string if present
func StewardNodeID(nod string) string {
	nodfile := registrydir + dd + stewardNodeIDFile
	return ioStringFile(nodfile, nod)
}

func ioStringFile(filepath, literal string) string {
	if !muckkeys { // stuff not initialised
		return ""
	}
	if literal == "" {
		// don't set, just return it
		if statFile(filepath) {
			return readStringFile(filepath)
		}
		return ""
	}
	return writeStringFile(filepath, literal)
}

func statFile(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func readStringFile(filepath string) string {
	filebytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return ""
	}
	literal := string(filebytes)
	if literal[len(literal)-1:] == "\n" {
		// chop off any newline
		return literal[:len(literal)-1]
	}
	return literal
}

// writeStringFile
// writes the string to the specified file
// with no newline at the end
// Overwrites any previously applied string
// Returns the string written if successful
func writeStringFile(filepath, literal string) string {
	err := ioutil.WriteFile(filepath, []byte(literal), 0600)
	if err != nil {
		return ""
	}
	return literal
}

// setPrincipal - use to set the persistent username for
// this fulcrum instanceover its life cycle.
// Pass in "" for a NUID based username. If a name was
// already written to storage, we will re-use that one.
// Does not change an existing name on with this, it will
// give you an error.
// If you choose to provide a username, it must be unique
// for each node. Since we  are using usernames in
// directory structures, ensure they follow character rules.
func setPrincipal(iam string) *c.Err {
	var whoiam string
	var err *c.Err
	var fileexists bool
	if !muckkeys { // stuff not initialised
		return c.ErrF("muck directories not set, cannot proceed")
	}
	if iam == "" { // you elect to use an NUID or previous saved name
		if statPrincipal() { // previous name exists
			whoiam, err = readPrincipal()
			if err != nil { // can't read previous name, or it is bad
				return err
			}
			fileexists = true
		} else { // no previous name, none provided, so we NUID it
			whoiam = nuid.Next() // comes CheckName() compliant already
		}
	} else { // user provided iam string
		if principal != "" {
			return c.ErrF("Name provided '%s', but name %s already set", iam, principal)
		}
		if statPrincipal() { // previous name file exists
			// if it is the exact same string, don't sweat it.
			whoiam, err = readPrincipal()
			if err != nil { // can't read previous name, or it is bad
				return err
			}
			if iam != whoiam { // not allowed to change the persistent name here
				return c.ErrF("Name provided '%s', mismatch to previous name '%s'", iam, whoiam)
			}
			fileexists = true
		}
		// iam  is CheckName() tested in InitMuck now
		whoiam = iam
	}
	if whoiam == "" { // Paranoia check
		return c.ErrF("failed to get a name string in SetPrincipal")
	}
	// Set the package variable
	principal = whoiam
	// Save it only if it is new, file timestamp should not change
	if !fileexists { // don't overwrite - if it exists and we just read it
		// Write principal to file
		err = writePrincipal(principal)
		if err != nil {
			return c.ErrF("Unable to store persistent name: %v", err)
		}
	}
	return nil
}

func statPrincipal() bool {
	principalFile := muckdir + dd + principalFile
	_, err := os.Stat(principalFile)
	return err == nil
}

// readPrincipal - reads the identity string in the muckdir/principal file
// confirms that the name follows the character conventions in CheckName()
func readPrincipal() (string, *c.Err) {
	principalFile := muckdir + dd + principalFile
	principalbytes, err := ioutil.ReadFile(principalFile)
	if err != nil {
		return "", c.ErrF("unable to read from principal file %s %v", principalFile, err)
	}
	principal = string(principalbytes)
	cerr := CheckName(principal)
	if cerr != nil {
		return "", c.ErrF("bad saved principal string '%s' %v", principal, cerr)
	}
	return principal, nil
}

// writePrincipal - writes the identity string to the muckdir/principal file
// Does not overwrite any previously applied identity!
// Should only write this file once over the lifetime of the restartable process
// caller needs to CheckName before proceeding.
func writePrincipal(iam string) *c.Err {
	principalFile := muckdir + dd + principalFile
	err := ioutil.WriteFile(principalFile, []byte(iam), 0600)
	if err != nil {
		return c.ErrF("failed to write %s %v", principalFile, err)
	}
	return nil
}

var invalidChars *regexp.Regexp

// CheckName - ensures names provided meet with
// certain requirements so they can be used in directory
// name (and consequently also as email names)
// OK with utf-8 symbols e.g. Röra
func CheckName(name string) *c.Err {
	// Primary rules:
	// must be email spec compliant prior to an imaginary
	// @domainname
	// catches \<>: "][, starting/trailing . and .. and tabs

	_, err := mail.ParseAddress(name + "@bob.com")
	if err != nil {
		return c.ErrF("name '%s' fails email format test, %v", name, err)
	}

	// Secondary rules:
	// reject additional characters with possible directory breaking symbols
	// NO [ / ~ # % & * { } ' ? ! , = ^ | ]
	// OK are: [- _ .]
	invalidChars = regexp.MustCompile(`[\/~#%&*{}'\?\!,=^|]`)
	match := invalidChars.MatchString(name)
	if match == true {
		return c.ErrF("name contains disallowed characters: %s", name)
	}
	return nil
}
