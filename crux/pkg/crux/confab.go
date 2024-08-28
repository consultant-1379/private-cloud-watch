package crux

/*
	this is the interface used by the communication module for crux.
	the primary interface is implemented by the flock pkg.
*/

import (
	"fmt"
	"time"

	"github.com/erixzone/crypto/pkg/tls"
	"github.com/erixzone/crypto/pkg/x509"
)

// MonInfo is a data packet for monitoring the flocking
type MonInfo struct {
	Op      MonOp
	Moniker string
	T       time.Time
	Flock   string
	Oflock  string
	N       int
}

// SString is an abbreviated string version
func (m MonInfo) SString() string {
	return fmt.Sprintf("%s: ldr=%s n=%d oflock=%s op=%d", m.Flock, m.Moniker, m.N+1, m.Oflock, m.Op)
}

// TLSCert : collected piece parts for easy assembly of a tls.Config
type TLSCert struct {
	Leaf tls.Certificate // our certificate
	Pool *x509.CertPool  // its parents
}

// ConfabN is (nearly) all the (readonly) aspects of flocking.
// old order was cluster, leader string, stable bool, node string
type ConfabN struct {
	// what am i
	Bloc  string // cluster is the flock's idea of the cluster name (machine generated; transient)
	Horde string // leader is the leader's node name
	Node  string // node is the node's name (machine generated; transient) TBD
	// who am i
	Leader string // leader is the leader's node name
	Stable bool   // stable is true if the flocking code thinks the leader is stable (it may take a while)
	// who can i talk to
	Yurt         string // the gateway to the ADMIN cluster
	Steward      string // address for steward
	RegistryAddr string // address of the registry
	RegistryKey  string // key to the registry
}

// Confab is the interface used by the communication module for crux.
type Confab interface {
	GetNames() ConfabN
	SetSteward(ip string, port int)
	SetRegistry(ip string, port int, key string)
	SetYurt(ip string, port int)
	SetHorde(string)
	GetCertificate() *TLSCert
	Monitor() chan MonInfo
	Close()
}
