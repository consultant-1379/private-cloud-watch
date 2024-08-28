package casbah

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
	"github.com/erixzone/xaas/platform/ssh-utils/pkg/pwd"
)

type ProxySession struct {
	net.Conn
	ssh.Session
}

func NewProxySession(sockname, username, dialstr string, hkeyfiles ...string) (*ProxySession, error) {
	hkeyCallback, err := KnownHostsCallback(hkeyfiles...)
	if err != nil {
		return nil, fmt.Errorf("knownhosts.New failed: %s", err)
	}
	km := NewKeyMaster(hkeyCallback)
	var keyring []ssh.Signer
	if _, ok := os.LookupEnv("KEYDICT"); ok {
		keys, err := pwd.GetKeyDict(username, dialstr, "")
		if err != nil {
			return nil, fmt.Errorf("pwd.GetKeyDict failed: %s", err)
		}
		keyring = append(keyring, keys...)
	}
	if _, ok := os.LookupEnv("KEYFILES"); ok {
		keys, err := pwd.GetKeys()
		if err != nil {
			return nil, fmt.Errorf("pwd.GetKeys failed: %s", err)
		}
		keyring = append(keyring, keys...)
	}
	var authlist []ssh.AuthMethod
	if len(keyring) > 0 {
		authlist = append(authlist, ssh.PublicKeys(keyring...))
	}
	if _, ok := os.LookupEnv("PWDFILE"); ok {
		pwds, err := pwd.GetPwds("")
		if err != nil {
			return nil, fmt.Errorf("pwd.GetPwds failed: %s", err)
		}
		authlist = append(authlist, ssh.Password(pwds.Pwd(username)))
	}
	if len(authlist) == 0 {
		return nil, fmt.Errorf("no authentication methods")
	}
	config := &ssh.ClientConfig{
		User: username,
		Auth: authlist,
		HostKeyCallback: km.Callback,
	}

	if !strings.Contains(dialstr, ":") {
		dialstr += ":22"
	}
	var conn net.Conn
	if sockname != "" {
		conn, err = client.ProxyDial(sockname, dialstr)
		if err != nil {
			return nil, fmt.Errorf("ProxyDial %s %s failed: %s\n", sockname, dialstr, err)
		}
	} else {
		conn, err = net.Dial("tcp", dialstr)
		if err != nil {
			return nil, fmt.Errorf("Dial %s failed: %s\n", dialstr, err)
		}
	}
	pSession := new(ProxySession)
	pSession.Conn = conn

	client, err := client.SshClient(conn, config)
	if err != nil {
		conn.Close()
		if strings.Contains(err.Error(), dialstr) {
			return nil, km.KeyError(err)
		} else {
			return nil, km.KeyError(fmt.Errorf("%s: %s", dialstr, err))
		}
	}
	session, err := client.NewSession()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("NewSession failed: %s", err)
	}
	pSession.Session = *session
	return pSession, nil
}

func (pSession *ProxySession) Close() {
	pSession.Session.Close()
	pSession.Conn.Close()
}

func KnownHostsCallback(hkeyfiles... string) (ssh.HostKeyCallback, error) {
	if hkeyfiles == nil {
		hkeyfiles = []string{"~/.ssh/known_hosts"}
	}
	var usr *user.User
	var err error
	for i, hkf := range hkeyfiles {
		if strings.HasPrefix(hkf, "~/") {
			if usr == nil {
				usr, err = user.Current()
				if err != nil {
					return nil, fmt.Errorf("Who do you think you are: %s", err)
				}
			}
			hkeyfiles[i] = path.Join(usr.HomeDir, hkf[2:])
		}
	}
	return knownhosts.New(hkeyfiles...)
}

type KeyMaster struct {
	hkc      ssh.HostKeyCallback
	hostname string
	key      ssh.PublicKey
	err      error
}

type HostKeyError struct {
	error
	Line string
}

func NewKeyMaster(hkc ssh.HostKeyCallback) *KeyMaster {
	return &KeyMaster{hkc: hkc}
}

func (km *KeyMaster) Callback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	km.hostname = hostname
	km.key = key
	km.err = km.hkc(hostname, remote, key)
	return km.err
}

func (km *KeyMaster) KeyError(err error) error {
	if km.err == nil {
		return err
	}
	hke := new(HostKeyError)
	hke.error = err
	hostname := knownhosts.Normalize(km.hostname)
	hke.Line = knownhosts.Line([]string{hostname}, km.key)
	return hke
}
