package flock

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/ec25519"

	"github.com/erixzone/crypto/pkg/ed25519"
	"github.com/erixzone/crypto/pkg/tls"
	"github.com/erixzone/crypto/pkg/x509"

	"github.com/prometheus/client_golang/prometheus"
)

// UDPFlock is the per-flock struct.
type UDPFlock struct {
	ip         net.IP
	bad        float32 // if -ve, then proceed normally. otherwise, probeAddrs is already set
	networks   []*net.IPNet
	probeAddrs []string
}

// Session : our data concerning the peer
type Session struct {
	pubKey     ec25519.Key // of the peer
	sessKey    *Key        // the one we've agreed upon
	prevKey    *Key        // the one that's timing out
	respKey    *Key        // the new one (not yet seen in a SessData msg)
	respCount  int          // this many Sends allowed before a SessData msg
	offerKey   *Key        // a nonce key
	offerNonce *Nonce      // also used to manage glare
}

// UDPNode is the per-node struct.
type UDPNode struct {
	sync.Mutex
	me          NodeID
	ip          net.IP
	port        int
	portS       string
	srcAddrS    string
	sock        *net.UDPConn
	inbound     chan []byte
	epochID     *Nonce
	KCVec       *prometheus.CounterVec
	KCount      [KCMax]prometheus.Counter
	Kmon        chan crux.MonInfo
	sharedKey   *Key                // BUG - still needed for beacon
	verifyOpts  *x509.VerifyOptions // cert chain
	certSecrKey ed25519.PrivateKey  // for signing
	cert        *x509.Certificate   // our certificate
	tlsCert     crux.TLSCert        // cert pieces packaged for TLS
	secrKey     ec25519.Key         // DH key, from certSecrKey
	pubKey      ec25519.Key
	sessions    map[string]*Session
}

// message types
const (
	_ = iota
	PubkeyOffer
	PubkeyResp
	SessData
)

// N.B. the following structs are marshaled by gob, so
// certain member names need to look "exported".

// PseudoHdr : additional authenticated data
type PseudoHdr struct {
	MsgType int
	SrcAddr string
	DstAddr string
}

func (ps PseudoHdr) String() string {
	return fmt.Sprintf("{ %d %s %s }", ps.MsgType, ps.SrcAddr, ps.DstAddr)
}

// Msg : struct to marshal/unmarshal flock messages
type Msg struct {
	MsgType int
	SrcAddr string
	CertDER []byte // only present in PubkeyOffer, PubkeyResp
	Nonce   []byte // only present in PubkeyResp
	Payload []byte
	cert    *x509.Certificate
	hdrBits []byte
	ecPub   ec25519.Key
}

func (k *UDPNode) newMsg(msgType int, dest string) *Msg {
	msg := Msg{MsgType: msgType, SrcAddr: k.srcAddrS}
	switch msgType {
	case PubkeyOffer, PubkeyResp:
		msg.CertDER = k.cert.Raw
	}
	pshdr := PseudoHdr{MsgType: msgType, SrcAddr: k.srcAddrS, DstAddr: dest}
	msg.hdrBits = gobSmack(&pshdr, nil)
	return &msg
}

func gobSmack(v interface{}, data []byte) []byte {
	var buf *bytes.Buffer
	var err error
	if data == nil {
		buf = new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		err = enc.Encode(v)
	} else {
		buf = bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		err = dec.Decode(v)
	}
	if err != nil {
		panic(fmt.Errorf("gobSmack: %s", err.Error()))
	}
	return buf.Bytes()
}

// CIDRbits : number of "1" bits in the network mask.
//            there are 2**(32-CIDRbits) host addresses in the subnet.
var CIDRbits uint = 28

// CIDRmask : the network mask
var CIDRmask uint32 = 0xfffffff0

// SetCIDR : set the size of the network to be probed
func SetCIDR(nbits uint) {
	if nbits < 12 { // 1M addresses
		nbits = 12
	} else if nbits > 29 { // 8 addresses
		nbits = 29
	}
	CIDRbits = nbits
	CIDRmask = 0xffffffff &^ ((1 << (32 - nbits)) - 1)
}

// NewUDP returns a NewUDP struct.
func NewUDP(port int, name, ip, networks string, kmon chan crux.MonInfo) (*UDPFlock, *UDPNode, *crux.Err) {
	// Create UDPNode to return.
	un := UDPNode{}
	un.port = port
	un.portS = fmt.Sprintf("%d", port)
	un.inbound = make(chan []byte, 99)
	if name == "" {
		name = crux.SmallID()
	}
	un.Kmon = kmon
	un.me.Moniker = name
	un.me.Addr = ip
	un.sessions = make(map[string]*Session)
	err1 := ec25519.NewKeyPair(&un.secrKey, &un.pubKey)
	if err1 != nil {
		return nil, nil, crux.ErrF("ec25519.NewKeyPair failed (%s)", err1.Error())
	}
	x, err := getIP(ip)
	if err != nil {
		return nil, nil, crux.ErrF("bad ip string '%s' (%s)", ip, err.Error())
	}
	un.ip = x
	un.srcAddrS = fmt.Sprintf("%s:%d", x.String(), port)
	clog.Log.Logi(nil, "NewUDP(%s): %s %s:%d pubkey %x", ip, name, un.ip.String(), port, un.pubKey)
	addr := net.UDPAddr{IP: un.ip, Port: un.port}

	un.KCVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "key_count",
			Help: "No. of key messages, partitioned by disposition.",
		},
		[]string{"disp"},
	)
	prometheus.MustRegister(un.KCVec)

	ifcs, err1 := net.Interfaces()
	if err1 == nil {
		for _, ifc := range ifcs {
			clog.Log.Logi(nil, "ifc %d mtu %d name %s", ifc.Index, ifc.MTU, ifc.Name)
		}
	} else {
		clog.Log.Logi(nil, "net.Interfaces() failed: %s", err1.Error())
	}

	sock, err1 := net.ListenUDP("udp", &addr)
	if err1 != nil {
		return nil, nil, crux.ErrF("UDP listener(%s:%d) failed: %s", un.ip.String(), un.port, err1.Error())
	}
	un.sock = sock
	go un.listener()

	// Create UDPFlock to return.
	uf := UDPFlock{}
	uf.bad = -1 // -ve means normal
	// The IP in UDPFlock is the same as the one in UDPNode.
	uf.ip = un.ip
	// Populate the networks field, which tells us the networks to probe as
	// part of this flock.
	err = uf.populateNetworks(networks)
	if err != nil {
		return nil, nil, crux.ErrF("Couldn't populate networks: %s", err.Error())
	}
	if uf.bad < 0 {
		// Populate the probeAddrs field using the networks list.
		err = uf.populateProbeAddrs()
		if err != nil {
			return nil, nil, crux.ErrF("Couldn't populate probeAddrs: %s", err.Error())
		}
	}
	return &uf, &un, nil
}

// KeyInit : intialize certificates and private key
func (k *UDPNode) KeyInit(vOpts *x509.VerifyOptions, priv interface{}, cert *x509.Certificate) error {
	chains, err := cert.Verify(*vOpts)
	if err != nil {
		return err
	}
	k.verifyOpts = vOpts
	k.certSecrKey = make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	copy(k.certSecrKey, priv.(ed25519.PrivateKey))
	k.cert = cert
	err = ec25519.NewKeyPairFromSeed(priv.(ed25519.PrivateKey), &k.secrKey, &k.pubKey)
	if err != nil {
		return err
	}
	var ecPub ec25519.Key
	err = ed25519.ECPublic(ecPub[:], cert.PublicKey.(ed25519.PublicKey))
	if err != nil {
		return err
	}
	if k.pubKey != ecPub {
		return fmt.Errorf("ec25519 and ed25519 public keys do not match")
	}
	tlsLeaf := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  priv,
		Leaf:        cert,
	}
	tlsPool := x509.NewCertPool()
	for _, c := range chains[0][1:] {
		tlsPool.AddCert(c)
	}
	k.tlsCert = crux.TLSCert{Leaf: tlsLeaf, Pool: tlsPool}
	return nil
}

// GetCertificate : cert chain for TLS
func (k *UDPNode) GetCertificate() *crux.TLSCert {
	return &k.tlsCert
}

// KCIncr : increment message counter
func (k *UDPNode) KCIncr(p KeyCount) {
	m := k.KCount[p]
	if m == nil {
		m = k.KCVec.WithLabelValues(p.String())
		k.KCount[p] = m
	}
	m.Inc()
}

// Populate the networks field, which tells us the networks to probe to create
// this flock. If the netcsv doesn't contain any networks, we try to use the
// local network.
func (k *UDPFlock) populateNetworks(netcsv string) (err *crux.Err) {
	// First, look for a file argument
	if (len(netcsv) > 5) && (netcsv[0:5] == "file:") {
		err := k.parseNetworkNames(netcsv[5:])
		if err != nil {
			return err
		}
		return nil
	}
	// Next, try to parse the netcsv string. If we get nothing back, we'll try
	// the local network instead.
	networks, err := parseNetworks(netcsv)
	if err != nil {
		return crux.ErrF("Couldn't run parseNetworks: %s", err.Error())
	}
	if len(networks) < 1 {
		localNet, err := getLocalNet(k.ip)
		if err != nil {
			return crux.ErrF("Couldn't get local network: %s", err.Error())
		}
		networks = []*net.IPNet{localNet}
	}
	k.networks = networks
	return nil
}

func (k *UDPFlock) parseNetworkNames(ifile string) (err *crux.Err) {
	ifile = strings.TrimSpace(ifile)
	fmt.Printf("reading file '%s'\n", ifile)
	b, err1 := ioutil.ReadFile(ifile)
	if err1 != nil {
		return crux.ErrE(err)
	}
	fmt.Printf("read %d bytes from %s\n", len(b), ifile)
	var miss float32
	var nl int
	k.probeAddrs = make([]string, 0)
	for cursor := 0; ; cursor += nl + 1 {
		// read addresses line by line
		nl = strings.IndexByte(string(b[cursor:]), '\n')
		if nl < 0 {
			break
		}
		ent := string(b[cursor : cursor+nl])
		fmt.Printf("analysing '%s'\n", ent)
		var s string
		var per float32
		n, err1 := fmt.Sscanf(ent, "%s %f ", &s, &per)
		if err1 != nil {
			n = 0 // just skip
		}
		if (s == "nil") && (n == 2) {
			miss = per
		} else {
			ip, err := getIP(ent)
			if err != nil {
				return err
			}
			k.probeAddrs = append(k.probeAddrs, ip.String())
			fmt.Printf("\tadding %s\n", ip.String())
		}
	}
	if len(k.probeAddrs) < 1 {
		return crux.ErrF("file %s contained no addresses", ifile)
	}
	k.bad = miss
	fmt.Printf("BAD:%g addrs:%v\n", k.bad, k.probeAddrs)
	return nil
}

// Given a comma-separated list of networks, return a slice of *net.IPNet.
// If any of them fail to parse, return an error.
func parseNetworks(networks string) (ipnets []*net.IPNet, err *crux.Err) {
	// Split the networks string by comma. Whitespace isn't cleaned.
	netstrings := strings.Split(networks, ",")
	// For each element, try to parse it as a CIDR network. If any element
	// can't be parsed, error. If elements are all empty, return nil slice.
	for _, n := range netstrings {
		if len(n) < 1 {
			continue
		}
		_, thisNet, err := net.ParseCIDR(n)
		if err != nil {
			return nil, crux.ErrF("Couldn't parse network %s: %s", n, err.Error())
		}
		ipnets = append(ipnets, thisNet)
	}
	return ipnets, nil
}

// Given an IP of a local interface, return the network for that IP.
// If we can't find it, return an error.
func getLocalNet(ip net.IP) (*net.IPNet, *crux.Err) {
	if ip == nil {
		return nil, crux.ErrF("getLocalNet called with nil ip, can't continue")
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, crux.ErrF("Couldn't get local interfaces: %s", err.Error())
	}
	// Walk through the interfaces, and for each address, see if it's the one
	// that matches the ip we were given.
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, crux.ErrF("Couldn't get addresses for interface %s: %s", i.Name, err.Error())
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				// If we found an ipNet, compare it to see if it matches.
				if ip.Equal(v.IP) {
					return v, nil
				}
			}
		}
	}
	// If we didn't match anything, return an error.
	return nil, crux.ErrF("Couldn't find ip %s in interface list", ip.String())
}

// Populate the probeAddrs field based on the networks list.
// Make sure our own IPs aren't in the list.
func (k *UDPFlock) populateProbeAddrs() (err *crux.Err) {
	// If we were called but there's not a list of networks to probe, complain.
	if len(k.networks) < 1 {
		return crux.ErrF("generateProbeAddrs called without networks available")
	}
	var probeAddrs []string
	probeAddrs, err = makeHostAddrs(k.networks)
	if err != nil {
		return crux.ErrF("Couldn't make a list of addresses from networks: %s", err.Error())
	}
	// Prune out all of our own IPs.
	probeAddrs = pruneLocalAddrs(probeAddrs)
	k.probeAddrs = probeAddrs
	return nil
}

// Given a list of IP addresses, prune out the ones that correspond to local IP
// addresses.
func pruneLocalAddrs(addrs []string) []string {
	// Make a pointer to the original array's slice.
	// The new slice will overwrite the old, which should be ok.
	newAddrs := addrs[:0]
	for _, addr := range addrs {
		_, err := getLocalNet(net.ParseIP(addr))
		// If err is nil, we found this IP on a local interface.
		// In other words, if we got an error, we like this IP.
		if err != nil {
			newAddrs = append(newAddrs, addr)
		}
	}
	return newAddrs
}

// Given a list of networks, return the host addresses in those networks as
// strings to be used by Probe().
func makeHostAddrs(networks []*net.IPNet) (hostAddrs []string, err *crux.Err) {
	// Get the host addresses for one network at a time.
	for _, n := range networks {
		var netAddrs []string
		// Make a copy of the network address and increment until the network
		// no longer contains the address.
		for addr := n.IP.Mask(n.Mask); n.Contains(addr); incrementIP(addr) {
			netAddrs = append(netAddrs, addr.String())
		}
		// Remove the network address and the broadcast address (the first and
		// last addresses) before adding these to hostAddrs.
		if len(netAddrs) > 2 {
			netAddrs = netAddrs[1 : len(netAddrs)-1]
			hostAddrs = append(hostAddrs, netAddrs...)
		}
	}
	// If we found no hostAddrs, this should be considered an error.
	if len(hostAddrs) < 1 {
		return nil, crux.ErrF("Couldn't find any host addresses in given networks [%+v]", networks)
	}
	return hostAddrs, nil
}

// Increment a given IP address.
func incrementIP(ip net.IP) {
	for octet := len(ip) - 1; octet >= 0; octet-- {
		ip[octet]++
		// If we flipped this octet to 0, we'll need to walk back to the next
		// lowest octet and increment it, too. Otherwise, we're done.
		if ip[octet] > 0 {
			break
		}
	}
}

func getIP(addr string) (net.IP, *crux.Err) {
	ip := net.ParseIP(addr)
	if ip == nil {
		ips, err := net.LookupIP(addr)
		if err != nil {
			return ip, crux.ErrF("bad ip string '%s' (%s)", addr, err.Error())
		}
		ip = ips[0] // could pick any, i guess
	}
	return ip, nil
}

func (k *UDPNode) listener() {
	for {
		p := make([]byte, 9999)
		n, a, err := k.sock.ReadFrom(p)
		if err != nil {
			k.Logf("UDP listener(%s) read failed: %s", k.srcAddrS, err.Error())
			continue
		}
		k.Logf("read pkt (%d) from %s", n, a.String())
		k.inbound <- p[:n]
	}
}

// gobEncrypt : marshal and encrypt a struct,
//              possibly with additional authenticated data
func gobEncrypt(v interface{}, addtext []byte, key *Key) ([]byte, error) {
	plaintext := gobSmack(v, nil)
	return EncryptWithAdd(plaintext, addtext, key)
}

// gobDecrypt : decrypt and unmarshall a payload (likely from a SessData message)
func gobDecrypt(ciphertext []byte, v interface{}, addtext []byte, key *Key) error {
	plaintext, err := DecryptWithAdd(ciphertext, addtext, key)
	if err != nil {
		return err
	}
	_ = gobSmack(v, plaintext)
	return nil
}

// gobCertify : marshal and sign a Msg
func (k *UDPNode) gobCertify(msg *Msg) []byte {
	msgbits := gobSmack(msg, nil)
	sig := ed25519.Sign(k.certSecrKey, msg.hdrBits, msgbits)
	return append(msgbits, sig...)
}

// offerPkt : marshal a PubkeyOffer to put on the wire
func (k *UDPNode) offerPkt(dest string, info *Info, sess *Session) ([]byte, error) {
	key, err := NewEncryptionKey()
	if err != nil {
		k.Logf("NewEncryptionKey failed for offer %s => %s", k.srcAddrS, dest)
		return nil, err
	}
	msg := k.newMsg(PubkeyOffer, dest)
	msg.Payload, err = gobEncrypt(info, nil, key)
	if err != nil {
		k.Logf("gobEncrypt failed for offer %s => %s", k.srcAddrS, dest)
		return nil, err
	}
	sess.offerKey = key
	sess.offerNonce = new(Nonce)
	copy(sess.offerNonce[:], msg.Payload[:NonceSize])
	k.Logf("offer sent %s => %s (%s) nonce %x",
		k.srcAddrS, info.Dest.Addr, dest, *sess.offerNonce)
	return k.gobCertify(msg), nil
}

// dataPkt : marshal a SessData message to put on the wire
func (k *UDPNode) dataPkt(dest string, info *Info, key *Key) ([]byte, error) {
	var err error
	msg := k.newMsg(SessData, dest)
	msg.Payload, err = gobEncrypt(info, msg.hdrBits, key)
	if err != nil {
		return nil, err
	}
	return gobSmack(msg, nil), nil
}

// Send a packet.
func (k *UDPNode) Send(info *Info) {
	var pkt []byte
	var err error
	var pKey *Key

	bailOnErr := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		k.Logf("%s: %s", msg, err.Error())
		k.Unlock()
	}

	ip, err1 := getIP(info.Dest.Addr)
	if err1 != nil {
		k.Logf("getIP(%s) failed: %s", info.Dest.Addr, err1.Error())
		return
	}
	uaddr := ip.String() + ":" + k.portS
	k.Logf("send pkt to %s (%s)", info.Dest.Addr, uaddr)

	k.Lock()
	sess := k.sessions[uaddr]
	if sess == nil {
		sess = &Session{}
		k.sessions[uaddr] = sess
	}
	if sess.sessKey != nil {
		pKey = sess.sessKey
	} else if sess.respKey != nil && sess.respCount > 0 {
		pKey = sess.respKey
		sess.respCount--
	} else if uaddr == k.srcAddrS {
		pKey, err = NewEncryptionKey()
		if err != nil {
			bailOnErr("NewEncryptionKey failed for self sess %s", k.srcAddrS)
			return
		}
		sess.sessKey = pKey
		k.Logf("self sess  %s", k.srcAddrS)
	}
	if pKey != nil { // we have a valid key
		pkt, err = k.dataPkt(uaddr, info, pKey)
		if err != nil {
			bailOnErr("dataPkt failed for Send %s => %s", k.srcAddrS, uaddr)
			return
		}
	} else { // no session key, so make an offer
		if k.cert == nil {
			bailOnErr("no certificate for Send %s => %s", k.srcAddrS, uaddr)
			return
		}
		pkt, err = k.offerPkt(uaddr, info, sess)
		if err != nil {
			bailOnErr("offerPkt failed for Send %s => %s", k.srcAddrS, uaddr)
			return
		}
	}
	k.Unlock()

	k.sendPkt(uaddr, pkt)
}

// called from Send or Recv
func (k *UDPNode) sendPkt(dest string, pkt []byte) bool {
	x := strings.SplitN(dest, ":", 2)
	ip, err1 := getIP(x[0])
	if err1 != nil {
		k.Logf("getIP(%s) failed: %s", dest, err1.Error())
		return false
	}
	addr := net.UDPAddr{IP: ip, Port: k.port}
	n, err := k.sock.WriteTo(pkt, &addr)
	if err != nil {
		k.Logf("send pkt failed: %s", err.Error())
		return false
	}
	if n != len(pkt) {
		k.Logf("pkt length error: sent %d, expected %d", n, len(pkt))
		return false
	}
	return true
}

// called under lock from recvMsg
func (k *UDPNode) recvOffer(sess *Session, msg *Msg) {
	var sesskey Key
	var nonceA Nonce
	var nonceB *Nonce
	var err error

	copy(nonceA[:], msg.Payload[:NonceSize])
	k.Logf("offer rcvd %s <= %s nonce %x", k.srcAddrS, msg.SrcAddr, nonceA)
	if sess.offerNonce != nil {
		nonceB = sess.offerNonce
		if bytes.Compare(nonceA[:], nonceB[:]) <= 0 { // glare: one side defers
			k.Logf("offer glare %s <= %s, dropping", k.srcAddrS, msg.SrcAddr)
			k.KCIncr(KCOfferGlare)
			return
		}
		k.Logf("offer glare %s <= %s, accepting", k.srcAddrS, msg.SrcAddr)
	} else {
		nonceB, err = NewGCMNonce()
		if err != nil {
			k.Logf("NewGCMNonce failed in recvOffer: %s", err.Error())
			return
		}
	}
	k.KCIncr(KCOffer)
	copy(sesskey[:], ec25519.SharedKey(&k.secrKey, &msg.ecPub,
		nonceA[:], nonceB[:]))
	k.Logf("resp sent  %s => %s session key %x", k.srcAddrS, msg.SrcAddr, sesskey)

	rmsg := k.newMsg(PubkeyResp, msg.SrcAddr)
	rmsg.Nonce = nonceB[:]
	rmsg.Payload, err = Encrypt(msg.Payload, &sesskey)
	if err != nil {
		k.Logf("Re-encrypt msg.Payload failed: %s", err.Error())
		return
	}

	sess.pubKey = msg.ecPub
	sess.respKey = &sesskey
	sess.respCount = 1
	sess.offerKey = nil
	sess.offerNonce = nil
	k.sendPkt(msg.SrcAddr, k.gobCertify(rmsg))
}

// called under lock from recvMsg
func (k *UDPNode) recvResp(sess *Session, msg *Msg) {

	var sesskey Key

	if sess.offerNonce == nil {
		return
	}

	copy(sesskey[:], ec25519.SharedKey(&k.secrKey, &msg.ecPub,
		sess.offerNonce[:], msg.Nonce[:]))
	k.Logf("resp rcvd  %s <= %s session key %x", k.srcAddrS, msg.SrcAddr, sesskey)
	offerBits, err := Decrypt(msg.Payload, &sesskey)
	if err != nil {
		k.Logf("Decrypt offerBits failed in recvResp: %s", err.Error())
		k.KCIncr(KCRespFail)
		return
	}
	var ff Info
	err = gobDecrypt(offerBits, &ff, nil, sess.offerKey)
	if err != nil { // a response to a different offer, perhaps
		k.Logf("gobDecrypt offerBits failed in recvResp: %s", err.Error())
		return
	}
	k.KCIncr(KCResp)
	pkt, err := k.dataPkt(msg.SrcAddr, &ff, &sesskey) // new session key
	if err != nil {
		k.Logf("dataPkt session first payload failed: %s", err.Error())
		return
	}
	sess.pubKey = msg.ecPub
	sess.sessKey = &sesskey
	sess.respKey = nil
	sess.offerKey = nil
	sess.offerNonce = nil
	k.sendPkt(msg.SrcAddr, pkt)
}

// called under lock from recvMsg
func (k *UDPNode) recvData(sess *Session, msg *Msg) *Info {

	var info Info
	var err error

	if sess.respKey != nil { // new session key pending?
		err = gobDecrypt(msg.Payload, &info, msg.hdrBits, sess.respKey)
		if err == nil { // rotate keys
			sess.sessKey = sess.respKey
			sess.respKey = nil
			k.KCIncr(KCDataResp)
			return &info
		}
	}
	if sess.sessKey != nil { // usual case
		err = gobDecrypt(msg.Payload, &info, msg.hdrBits, sess.sessKey)
		if err == nil {
			k.KCIncr(KCData)
			return &info
		}
	}
	if sess.prevKey != nil { // sent with the old key, maybe
		err = gobDecrypt(msg.Payload, &info, msg.hdrBits, sess.prevKey)
		if err == nil {
			k.KCIncr(KCDataPrev)
			return &info
		}
	}
	if err == nil {
		k.Logf("recvData: no keys")
		k.KCIncr(KCDataNoKey)
	} else {
		k.Logf("recvData: corrupt info Payload: %s", err.Error())
		k.KCIncr(KCDataFail)
	}
	return nil
}

func (k *UDPNode) ouCompare(certB *x509.Certificate) bool {
	ouA := k.cert.Subject.OrganizationalUnit
	ouB := certB.Subject.OrganizationalUnit
	ok := len(ouA) == len(ouB)
	if ok {
		for i, a := range ouA {
			if a != ouB[i] {
				ok = false
				break
			}
		}
	}
	if !ok {
		k.Logf("recvMsg: OU mismatch: expected %s, got %s", ouA, ouB)
	}
	return ok
}

func (k *UDPNode) recvMsg(pkt []byte) *Info {
	var err error
	var msg Msg

	sig := gobSmack(&msg, pkt)
	if len(sig) > 0 {
		if len(sig) != ed25519.SignatureSize {
			k.Logf("recvMsg: bad signature size")
			return nil
		}
		n := len(pkt) - ed25519.SignatureSize
		pkt = pkt[:n]
	}
	pshdr := PseudoHdr{MsgType: msg.MsgType, SrcAddr: msg.SrcAddr, DstAddr: k.srcAddrS}
	msg.hdrBits = gobSmack(&pshdr, nil)
	if len(sig) > 0 {
		if msg.CertDER == nil {
			k.Logf("recvMsg: signature without cert")
			return nil
		}
		msg.cert, err = x509.ParseCertificate(msg.CertDER)
		if err != nil {
			k.Logf("recvMsg: %s", err.Error())
			return nil
		}
		if k.verifyOpts == nil {
			k.Logf("recvMsg: no certificate chain")
			return nil
		}
		_, err = msg.cert.Verify(*k.verifyOpts)
		if err != nil {
			k.Logf("recvMsg: %s", err.Error())
			return nil
		}
		if !ed25519.VerifyMulti(msg.cert.PublicKey.(ed25519.PublicKey), sig, msg.hdrBits, pkt) {
			k.Logf("recvMsg: signature failure on msg type %d", msg.MsgType)
			return nil
		}
		if !k.ouCompare(msg.cert) {
			return nil
		}
		err = ed25519.ECPublic(msg.ecPub[:], msg.cert.PublicKey.(ed25519.PublicKey))
		if err != nil {
			k.Logf("recvMsg: %s", err.Error())
			return nil
		}
	} else if msg.CertDER != nil {
		k.Logf("recvMsg: corrupt Msg: cert without signature")
		return nil
	}
	k.Lock()
	defer k.Unlock()
	sess := k.sessions[msg.SrcAddr]
	if sess == nil {
		sess = &Session{}
		k.sessions[msg.SrcAddr] = sess
	}
	k.Logf("recvMsg (%d) %s", len(pkt), pshdr.String())

	switch msg.MsgType {
	case PubkeyOffer:
		k.recvOffer(sess, &msg)
	case PubkeyResp:
		k.recvResp(sess, &msg)
	case SessData:
		return k.recvData(sess, &msg)
	default:
		k.Logf("bogus MsgType %d", msg.MsgType)
	}
	return nil
}

// Recv a packet.
func (k *UDPNode) Recv() *Info {
	for {
		pkt := <-k.inbound
		//k.Logf("Recv pkt (%d)", len(pkt))
		ff := k.recvMsg(pkt)
		if ff != nil {
			return ff
		}
	}
}

// SetMe sets my name
func (k *UDPNode) SetMe(me *NodeID) {
	if me.Moniker == "" {
		me.Moniker = k.me.Moniker
	}
	me.Addr = k.me.Addr
}

// Heartbeat is the period for sending heartbeats
func (k *UDPFlock) Heartbeat() time.Duration {
	return 1000 * time.Millisecond
}

// KeyPeriod is the period for changing the session keys
func (k *UDPFlock) KeyPeriod() time.Duration {
	return 100 * k.Heartbeat()
}

// NodePrune is when to lose flock membership if no heartbeats
func (k *UDPFlock) NodePrune() time.Duration {
	return 3 * k.Heartbeat()
}

// HistoryPrune is how to forget old nodes after this
func (k *UDPFlock) HistoryPrune() time.Duration {
	return 2 * k.Heartbeat()
}

// Checkpoint is when to drop a checkpoint after this period
func (k *UDPFlock) Checkpoint() time.Duration {
	return 1 * time.Minute
}

// Probebeat is the period between rework probes
func (k *UDPFlock) Probebeat() time.Duration {
	return 3 * k.Heartbeat()
}

// ProbeN is how many probes per rework
func (k *UDPFlock) ProbeN() int {
	return 12
}

// Probe returns a random address from our probeAddrs list.
func (k *UDPFlock) Probe() (string, bool) {
	// first check to see if we have a specified list
	if k.bad >= 0 {
		if rand.Float32() < k.bad {
			fmt.Printf("Probe: no addr\n")
			return "0", false
		}
		str := k.probeAddrs[rand.Intn(len(k.probeAddrs))]
		k.Logf("Probe: %s", str)
		return str, true
	}
	ipv4 := []byte(k.ip.To4())
	var ip uint32
	for _, octet := range ipv4 {
		ip = (ip << 8) | uint32(octet)
	}
	ip = (ip & CIDRmask) | another(ip&^CIDRmask, (1<<(32-CIDRbits)))
	str := net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)).String()
	k.Logf("Probe: %s", str)
	return str, true
}

func another(l uint32, max int) uint32 {
	for {
		x := uint32(rand.Intn(max))
		if x != l {
			return x
		}
	}
}

// SetKeys use nil for unused keys
func (k *UDPNode) SetKeys(epochID *Nonce, sec0, sec1 *Key) {
	if k.sharedKey == nil && sec0 != nil {
		k.sharedKey = new(Key)
		*k.sharedKey = *sec0
		k.Logf("node %s shared key %x", k.me.Addr, *k.sharedKey)
	}
	if k.epochID != epochID { // new session keys, please
		k.Lock()
		k.epochID = epochID
		k.Logf("node %s new session keys epoch %s", k.me.Addr, epochID)
		for _, sess := range k.sessions {
			if sess.sessKey != nil {
				sess.prevKey = sess.sessKey
				sess.sessKey = nil
			}
		}
		k.Unlock()
	}
}

// Monitor the flocking world; nil is off
func (k *UDPNode) Monitor() chan crux.MonInfo {
	return k.Kmon
}

// Quit is how we know we're done
func (k *UDPNode) Quit() {
}

// Logf is how we log stuff
func (k *UDPNode) Logf(format string, args ...interface{}) {
	clog.Log.Log("focus", "udp", fmt.Sprintf(format, args...))
}

// Logf is how we log stuff
func (k *UDPFlock) Logf(format string, args ...interface{}) {
	clog.Log.Log("focus", "udpf", fmt.Sprintf(format, args...))
}

// Quit is for shutting down
func (k *UDPFlock) Quit() {

}
