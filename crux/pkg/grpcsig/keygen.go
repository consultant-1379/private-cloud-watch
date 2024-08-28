// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/muck"
)

// AddKeyToAgent - adds private key to ssh-agent via call to ssh-add
// Works without password only if no passphrase in public key
// Since this is to use transient self-keys - we are ok with this.
func AddKeyToAgent(keypath string, debug bool) *c.Err {
	fileflag := "-k"
	filearg := keypath
	var cmd *exec.Cmd
	if debug {
		fmt.Printf("%s %s %s\n",
			sshadd, fileflag, filearg)
	}
	cmd = exec.Command(sshadd, fileflag, filearg)
	cmd.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return c.ErrF("exec ssh-add failed [%s] to start [%v]", stderr.String(), err)
	}
	err = cmd.Wait()
	if err != nil {
		return c.ErrF("exec ssh-add failed [%s] exec returned [%v]", stderr.String(), err)
	}
	return nil
}

// RemoveKeyFromAgent - removes a private key to ssh-agent via call to ssh-add
// Works without password only if no passphrase in public key
// Since this is to use transient self-keys - we are ok with this.
func RemoveKeyFromAgent(keypath string, debug bool) *c.Err {
	fileflag := "-d"
	filearg := keypath
	var cmd *exec.Cmd
	if debug {
		fmt.Printf("%s %s %s\n",
			sshadd, fileflag, filearg)
	}
	cmd = exec.Command(sshadd, fileflag, filearg)
	cmd.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return c.ErrF("exec ssh-add -d failed [%s] to start [%v]", stderr.String(), err)
	}
	err = cmd.Wait()
	if err != nil {
		return c.ErrF("exec ssh-add -d failed [%s] exec returned [%v]", stderr.String(), err)
	}
	return nil
}

// ListKeysFromAgent - prints what is in the ssh-agent at present when in debug mode
// exit status 1 error is an empty list
// Mostly just here for testing the package
func ListKeysFromAgent(debug bool) (*bytes.Buffer, *c.Err) {
	listflag := "-l"
	var cmd *exec.Cmd
	if debug {
		fmt.Printf("%s %s\n", sshadd, listflag)
	}
	cmd = exec.Command(sshadd, listflag)
	cmd.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return nil, c.ErrF("exec ssh-add list failed [%s] to start [%v]", stderr.String(), err)
	}
	err = cmd.Wait()
	if err != nil {
		return nil, c.ErrF("exec ssh-add list failed [%s] exec returned [%v]", stderr.String(), err)
	}
	if debug {
		fmt.Printf("Stderr:[%s]\nStdout:\n%s\n", stderr.String(), stdout.String())
	}
	return &stdout, nil
}

// NewKeyPair - executes ssh-keygen to make public/private key pairs
// Default ctype (cryptotype) is ed25519 when "" is passed as first argument
// or supply "ecdsa" or "rsa"
// Argument keypath is path to target private key file
// Provide a passphrase for keyfile encryption or "" for none.
// Pass debug as true - to print command line equivalent
// Return a partially completed *PubKeyT, without full KeyID (raw fingeprint, no Name or Service fields)
func NewKeyPair(ctype string, keypath string, passphrase string, comment string, debug bool) (*PubKeyT, *c.Err) {
	if len(ctype) == 0 {
		ctype = cryptotype
	}
	fileflag := "-f"
	filearg := keypath
	fmtflag := "-E"
	fmtarg := "md5"
	typeflag := "-t"
	typearg := ctype
	quietflag := "-q"
	nopflag := "-N"
	nopwarg := "''"
	if len(passphrase) > 0 {
		nopwarg = passphrase
	}
	commentflag := "-C"
	commentarg := comment
	var cmd *exec.Cmd
	bashrun := fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s %s %s\n",
		sshkeygen, fileflag, filearg, fmtflag, fmtarg, typeflag, typearg,
		commentflag, commentarg, quietflag, nopflag, nopwarg)
	if debug {
		fmt.Printf("%s\n", bashrun)
	}
	cmd = exec.Command("bash", "-c", bashrun)
	cmd.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return nil, c.ErrF("exec ssh-keygen failed [%s] to start [%v]", stderr.String(), err)
	}
	err = cmd.Wait()
	if err != nil {
		return nil, c.ErrF("exec ssh-keygen failed [%s] exec returned [%v]", stderr.String(), err)
	}
	return LoadAndValidateGenericPubKey(keypath + ".pub")
}

// MakeLocalKeyID - Adds in the KeyID from the local .muck held principal and the provided servicerev
func MakeLocalKeyID(servicerev string, pk *PubKeyT) *c.Err {
	pid, serr := muck.Principal()
	if serr != nil {
		return serr
	}
	nerr := muck.CheckName(servicerev)
	if nerr != nil {
		return nerr
	}
	fp := pk.KeyID // raw fingerprint
	if len(fp) == 0 {
		return c.ErrF("public key has no fingerprint")
	}
	pk.Name = pid
	pk.Service = servicerev
	pk.KeyID = fmt.Sprintf("/%s/%s/keys/%s", servicerev, pid, fp)
	return nil
}

// LoadAndValidateGenericPubKey - loads a public keyfile, decodes it, tests parsing it
// check the crypto type, makes its fingerprint, returns a partially filled out *PubKeyT
// Returns a JSON formatted PubKeyT matching our BoltDB system, but with no Service, no Name and
// the raw Fingerprint only.
// Caller must prepend the /servicerev/principal/keys/ string to the returned PubKey, if needed
func LoadAndValidateGenericPubKey(pubkeypath string) (*PubKeyT, *c.Err) {
	pubkey := &PubKeyT{}
	// Read the public key file.
	f, ferr := os.Open(pubkeypath)
	if ferr != nil {
		return nil, c.ErrF("unable to read public key file %s : %v", pubkeypath, ferr)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan() // one line
	pubkey.PubKey = scanner.Text()

	// See if it parses
	_, cryptotype, cerr := DecodeSSHPublicKey(pubkey.PubKey)
	if cerr != nil {
		return nil, c.ErrF("bad SSH public key file - crypto type %s, file %s, error %v",
			cryptotype, pubkeypath, cerr)
	}

	// Get its raw fingerprint
	fingerprint, gerr := FingerprintSSHPublicKey(pubkey.PubKey)
	if gerr != nil {
		return nil, gerr
	}

	pubkey.KeyID = fingerprint // This is just the raw fingerprint, not a full KeyID
	return pubkey, nil
}
