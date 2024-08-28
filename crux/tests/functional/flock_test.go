package ftests

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/ec25519"
	"github.com/erixzone/crux/pkg/flock"
	"github.com/erixzone/crux/pkg/x509ca"

	"github.com/erixzone/crypto/pkg/ed25519"
	"github.com/erixzone/crypto/pkg/tls"
	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/crypto/pkg/x509/pkix"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var scruntLevel = 1

const debug = false

// shut up lint
const (
	testDepth = 20000 // should be enough
)

func TestFlock(t *testing.T) { TestingT(t) }

type flockSuite struct {
	f flockStuff
}

type quirk struct {
	t time.Time
	s string
}

type flockStuff struct {
	sync.Mutex
	n        int
	inq      map[string]chan []byte
	info     chan crux.MonInfo
	interest []quirk
	nodes    []*nodeStuff
	density  float32
}

// Session : our data concerning the peer
type Session struct {
	pubKey     ec25519.Key  // of the peer
	sessKey    *flock.Key   // the one we've agreed upon
	prevKey    *flock.Key   // the one that's timing out
	respKey    *flock.Key   // the new one (not yet seen in a SessData msg)
	respCount  int          // this many Sends allowed before a SessData msg
	offerKey   *flock.Key   // a nonce key
	offerNonce *flock.Nonce // also used to manage glare
}

// nodeStuff is the per-node struct.
type nodeStuff struct {
	sync.Mutex  // for session update
	f           *flock.Flock
	me          flock.NodeID
	n           int
	epochID     *flock.Nonce
	KCount      [flock.KCMax]int
	KCPCount    [flock.KCMax]prometheus.Counter
	kerr        int
	flock       *flockStuff
	sharedKey   *flock.Key          // BUG - still needed for beacon
	verifyOpts  *x509.VerifyOptions // certificate chain
	certSecrKey ed25519.PrivateKey  // signing key
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

func (k *nodeStuff) newMsg(msgType int, dest string) *Msg {
	msg := Msg{MsgType: msgType, SrcAddr: k.me.Addr}
	switch msgType {
	case PubkeyOffer, PubkeyResp:
		msg.CertDER = k.cert.Raw
	}
	pshdr := PseudoHdr{MsgType: msgType, SrcAddr: k.me.Addr, DstAddr: dest}
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

type PromHandler struct {
	sync.Mutex
	testComplete bool
	quitChan     chan bool
	prom         http.Handler
	qcount       int
	ptotal       int
	ppost        int
}

var waitForQuit = flag.Bool("wait", false, "wait for quit message")

func NewPromHandler() *PromHandler {
	h := &PromHandler{
		quitChan: make(chan bool, 1),
		prom:     promhttp.Handler(),
	}
	http.Handle("/metrics", h)
	http.Handle("/quit", h)
	return h
}

func (h *PromHandler) WaitFinish() {
	h.testComplete = true
	if len(h.quitChan) > 0 {
		// pass
	} else if *waitForQuit {
		fmt.Printf("waiting on quit channel...\n")
	} else if h.ptotal > 0 {
		fmt.Printf("waiting for final scrapes...\n")
	} else {
		fmt.Printf("no scrapers, no waiting.\n")
		return
	}
	<-h.quitChan
	fmt.Printf("quit message received.\n")
}

func (h *PromHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Lock()
	defer h.Unlock()
	switch r.URL.Path {
	case "/quit":
		h.qcount++
		if h.qcount == 1 && h.ppost < 2 {
			fmt.Fprintf(w, "abyssinia\n")
			h.quitChan <- true
		}
	case "/metrics":
		h.ptotal++
		h.prom.ServeHTTP(w, r)
		if h.testComplete && !*waitForQuit {
			h.ppost++
			if h.qcount < 1 && h.ppost == 2 {
				fmt.Printf("zot!\n")
				h.quitChan <- true
			}
		}
	}
}

var promHandler *PromHandler
var kCPVec *prometheus.CounterVec

func init() {
	kCPVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "key_count",
			Help: "No. of key messages, partitioned by node and disposition.",
		},
		[]string{"node", "disp"},
	)
	prometheus.MustRegister(kCPVec)
	promHandler = NewPromHandler()
	go http.ListenAndServe("localhost:8090", nil)
	Suite(&flockSuite{})
	logf, err := os.Create("junk.log")
	crux.Assert(err == nil)
	clog.Log = crux.GetLoggerW(logf)
	//	clog.Log.Formatter.TimestampFormat = "15:04:05.000"
}

func (k *nodeStuff) KCIncr(p flock.KeyCount) {
	k.KCount[p]++ // trust but verify
	m := k.KCPCount[p]
	if m == nil {
		m = kCPVec.WithLabelValues(k.me.Moniker, p.String())
		k.KCPCount[p] = m
	}
	m.Inc()
}

func (k *nodeStuff) KCValue(p flock.KeyCount) int {
	m := k.KCPCount[p]
	if m == nil {
		return 0
	}
	return int(testutil.ToFloat64(m))
}

func (k *flockSuite) SetUpSuite(c *C) {
}

func (k *flockSuite) TearDownSuite(c *C) {
}

const VersionString = "2019-03-06 v.02"

func (k *flockSuite) TestBasic(c *C) {
	//for debugging, use: k.experiment(c, 10, 0.1)
	//for testing, use k.experiment(c, 100, 0.01)
	fmt.Printf("test vers. \"%s\" flock vers. \"%s\"\n", VersionString, flock.VersionString)
	k.experiment(c, 10, 0.1)

	promHandler.WaitFinish()
}

func (k *flockSuite) experiment(c *C, nnodes int, density float32) {
	var epoch time.Time
	insert := func(sss string) {
		k.f.Lock()
		k.f.interest = append(k.f.interest, quirk{t: time.Now().UTC(), s: sss})
		k.f.Unlock()
	}
	rebootStorm := func(nstorm, frac float32) {
		msg := fmt.Sprintf("reboot storm %.0f (%.0f%%)", nstorm, frac*100)
		fmt.Println(msg)
		insert(msg)
		epoch = time.Now().UTC()
		for _, kk := range k.f.nodes {
			if rand.Float32() < frac {
				kk.f.Reboot()
			}
		}
		// it will take about reboot+nodeprune before the action starts
	}

	fmt.Printf("starting flock of %d nodes and density %.2f\n", nnodes, density)

	subject := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux"},
		Locality:           []string{"Brookside"},
		Province:           []string{"NJ"},
		CommonName:         "Erixzone crux Root Certificate",
	}
	root, rootPriv, err := x509ca.MakeRootCert(subject, 120,
		"If you trust this Root Certificate then we have a bridge that will interest you.")
	if err != nil {
		panic(fmt.Sprintf("MakeRootCert: %s", err))
	}
	subject.CommonName = "Erixzone crux Server CA X1"
	CA, caPriv, err := x509ca.MakeCACert(subject, 36,
		"This Certificate Authority is for entertainment purposes only.",
		root, rootPriv)
	if err != nil {
		panic(fmt.Sprintf("MakeCACert: %s", err))
	}
	rootPool := x509.NewCertPool()
	rootPool.AddCert(root)
	intPool := x509.NewCertPool()
	intPool.AddCert(CA)
	verifyOpts := x509.VerifyOptions{Roots: rootPool, Intermediates: intPool}

	k.f = flockStuff{inq: make(map[string]chan []byte), info: make(chan crux.MonInfo, testDepth), density: density, n: nnodes}
	qa := make(chan bool)
	qb := make(chan string)
	go k.analyse(c, qa, qb)

	base, _ := flock.NewEncryptionKey()
	insert(fmt.Sprintf("run start heartbeat=%s", k.f.Heartbeat().String()))
	epoch = time.Now().UTC()
	for i := 0; i < nnodes; i++ {
		name := fmt.Sprintf("node%03d", i)
		subject = pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Erixzone"},
			OrganizationalUnit: []string{"Crux"},
			DomainComponent:    []string{"TestFlock"},
			CommonName:         name,
		}
		leafCert, leafPriv, err := x509ca.MakeLeafCert(subject, 3,
			"Keep away from children.  This certificate is a toy.",
			CA, caPriv)
		if err != nil {
			panic(fmt.Sprintf("MakeLeafCert (%s): %s", name, err))
		}
		ns := nodeStuff{flock: &k.f, sessions: make(map[string]*Session)}
		err = ns.keyInit(&verifyOpts, leafPriv, leafCert)
		if err != nil {
			panic(fmt.Sprintf("keyInit %s: %s", name, err))
		}
		kk := flock.NewFlockNode(flock.NodeID{Moniker: name}, &ns, &k.f, base, "xxx", false)
		ns.f = kk
		k.f.nodes = append(k.f.nodes, &ns)
		tc := ns.GetCertificate()
		if tc == nil {
			panic(fmt.Sprintf("GetCertificate failed on %s", name))
		}
	}

	result := <-qb
	fmt.Printf("%s [%s]\n", result, time.Now().UTC().Sub(epoch))
	if true {
		rebootStorm(1, .1)
		result = <-qb
		fmt.Printf("%s [%s]\n", result, time.Now().UTC().Sub(epoch))
		if !debug {
			rebootStorm(2, .5)
			result = <-qb
			fmt.Printf("%s [%s]\n", result, time.Now().UTC().Sub(epoch))
			rebootStorm(3, .9)
			result = <-qb
			fmt.Printf("%s [%s]\n", result, time.Now().UTC().Sub(epoch))
		}
	}

	for _, kk := range k.f.nodes {
		kk.Quit()
	}
	// stop analyse
	qa <- true
	<-qb
	pktsum(k.f.nodes)
}

func (k *nodeStuff) keyInit(vOpts *x509.VerifyOptions, priv ed25519.PrivateKey, cert *x509.Certificate) error {
	chains, err := cert.Verify(*vOpts)
	if err != nil {
		return err
	}
	k.verifyOpts = vOpts
	k.certSecrKey = make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	copy(k.certSecrKey, priv)
	k.cert = cert
	err = ec25519.NewKeyPairFromSeed(priv, &k.secrKey, &k.pubKey)
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
	k.me.Moniker = cert.Subject.CommonName
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
func (k *nodeStuff) GetCertificate() *crux.TLSCert {
	return &k.tlsCert
}

func pktsum(nodes []*nodeStuff) {
	var b strings.Builder
	var k flock.KeyCount

	for k = 0; k < flock.KCMax; k++ {
		var c int
		for _, n := range nodes {
			v := n.KCValue(k)
			if v != n.KCount[k] {
				panic(fmt.Errorf("pktsum: node %s counter %s expected %d got %d", n.me.Moniker, k, n.KCount[k], v))
			}
			c += v
		}
		fmt.Fprintf(&b, " %s=%d", k.String(), c)
	}
	fmt.Printf("pkts:%s\n", b.String())
}

// gobEncrypt : marshal and encrypt a struct,
//              possibly with additional authenticated data
func gobEncrypt(v interface{}, addtext []byte, key *flock.Key) ([]byte, error) {
	plaintext := gobSmack(v, nil)
	return flock.EncryptWithAdd(plaintext, addtext, key)
}

// gobDecrypt : decrypt and unmarshall a payload (likely from a SessData message)
func gobDecrypt(ciphertext []byte, v interface{}, addtext []byte, key *flock.Key) error {
	plaintext, err := flock.DecryptWithAdd(ciphertext, addtext, key)
	if err != nil {
		return err
	}
	_ = gobSmack(v, plaintext)
	return nil
}

// gobCertify : marshal and sign a Msg
func (k *nodeStuff) gobCertify(msg *Msg) []byte {
	msgbits := gobSmack(msg, nil)
	sig := ed25519.Sign(k.certSecrKey, msg.hdrBits, msgbits)
	return append(msgbits, sig...)
}

// offerPkt : marshal a PubkeyOffer to put on the wire
func (k *nodeStuff) offerPkt(dest string, info *flock.Info, sess *Session) ([]byte, error) {
	key, err := flock.NewEncryptionKey()
	if err != nil {
		k.Logf("NewEncryptionKey failed for offer %s => %s", k.me.Addr, dest)
		return nil, err
	}
	msg := k.newMsg(PubkeyOffer, dest)
	msg.Payload, err = gobEncrypt(info, nil, key)
	if err != nil {
		k.Logf("gobEncrypt failed for offer %s => %s", k.me.Addr, dest)
		return nil, err
	}
	sess.offerKey = key
	sess.offerNonce = new(flock.Nonce)
	copy(sess.offerNonce[:], msg.Payload[:flock.NonceSize])
	DBPrintf("offer sent %s => %s (%s) nonce %x\n",
		k.me.Addr, info.Dest.Addr, dest, *sess.offerNonce)
	return k.gobCertify(msg), nil
}

// dataPkt : marshal a SessData message to put on the wire
func (k *nodeStuff) dataPkt(dest string, info *flock.Info, key *flock.Key) ([]byte, error) {
	var err error
	msg := k.newMsg(SessData, dest)
	msg.Payload, err = gobEncrypt(info, msg.hdrBits, key)
	if err != nil {
		return nil, err
	}
	return gobSmack(msg, nil), nil
}

func DBPrintf(format string, v ...interface{}) (int, error) {
	if debug {
		return fmt.Printf(format, v...)
	}
	return 0, nil
}

// Send a packet.
func (k *nodeStuff) Send(info *flock.Info) {
	var pkt []byte
	var err error
	var pKey *flock.Key

	bailOnErr := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		k.Logf("%s: %s", msg, err.Error())
		k.Unlock()
	}

	dest := string(info.Dest.Addr)
	if dest == "" {
		return
	}
	k.Lock()
	sess := k.sessions[dest]
	if sess == nil {
		sess = &Session{}
		k.sessions[dest] = sess
	}
	if sess.sessKey != nil {
		pKey = sess.sessKey
	} else if sess.respKey != nil && sess.respCount > 0 {
		pKey = sess.respKey
		sess.respCount--
	} else if dest == k.me.Addr {
		pKey, err = flock.NewEncryptionKey()
		if err != nil {
			bailOnErr("NewEncryptionKey failed for self sess %s", k.me.Addr)
			return
		}
		sess.sessKey = pKey
		DBPrintf("self sess  %s\n", k.me.Addr)
	}
	if pKey != nil { // we have a valid key
		pkt, err = k.dataPkt(dest, info, pKey)
		if err != nil {
			bailOnErr("dataPkt failed for Send %s => %s", k.me.Addr, dest)
			return
		}
	} else { // no session key, so make an offer
		pkt, err = k.offerPkt(dest, info, sess)
		if err != nil {
			bailOnErr("offerPkt failed for Send %s => %s", k.me.Addr, dest)
			return
		}
	}
	k.Unlock()

	k.sendPkt(dest, pkt)
}

// called from Send or Recv
func (k *nodeStuff) sendPkt(dest string, pkt []byte) {
	k.flock.Lock()
	defer k.flock.Unlock()
	ochan := k.flock.inq[dest]
	if ochan == nil {
		ochan = make(chan []byte, testDepth)
		k.flock.inq[dest] = ochan
	}

	ochan <- pkt
	k.n++
}

// called under lock from recvMsg
func (k *nodeStuff) recvOffer(sess *Session, msg *Msg) {
	var sesskey flock.Key
	var nonceA flock.Nonce
	var nonceB *flock.Nonce
	var err error

	copy(nonceA[:], msg.Payload[:flock.NonceSize])
	DBPrintf("offer rcvd %s <= %s nonce %x\n", k.me.Addr, msg.SrcAddr, nonceA)
	if sess.offerNonce != nil {
		nonceB = sess.offerNonce
		if bytes.Compare(nonceA[:], nonceB[:]) <= 0 { // glare: one side defers
			DBPrintf("offer glare %s <= %s, dropping\n", k.me.Addr, msg.SrcAddr)
			k.KCIncr(flock.KCOfferGlare)
			return
		}
		DBPrintf("offer glare %s <= %s, accepting\n", k.me.Addr, msg.SrcAddr)
	} else {
		nonceB, err = flock.NewGCMNonce()
		if err != nil {
			k.Logf("NewGCMNonce failed in recvOffer: %s", err.Error())
			return
		}
	}
	k.KCIncr(flock.KCOffer)
	copy(sesskey[:], ec25519.SharedKey(&k.secrKey, &msg.ecPub,
		nonceA[:], nonceB[:]))
	DBPrintf("resp sent  %s => %s session key %x\n", k.me.Addr, msg.SrcAddr, sesskey)

	rmsg := k.newMsg(PubkeyResp, msg.SrcAddr)
	rmsg.Nonce = nonceB[:]
	rmsg.Payload, err = flock.Encrypt(msg.Payload, &sesskey)
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
func (k *nodeStuff) recvResp(sess *Session, msg *Msg) {

	var sesskey flock.Key

	if sess.offerNonce == nil {
		return
	}

	copy(sesskey[:], ec25519.SharedKey(&k.secrKey, &msg.ecPub,
		sess.offerNonce[:], msg.Nonce[:]))
	DBPrintf("resp rcvd  %s <= %s session key %x\n", k.me.Addr, msg.SrcAddr, sesskey)
	offerBits, err := flock.Decrypt(msg.Payload, &sesskey)
	if err != nil {
		k.Logf("Decrypt offerBits failed in recvResp: %s", err.Error())
		k.KCIncr(flock.KCRespFail)
		return
	}
	var ff flock.Info
	err = gobDecrypt(offerBits, &ff, nil, sess.offerKey)
	if err != nil { // a response to a different offer, perhaps
		k.Logf("gobDecrypt offerBits failed in recvResp: %s", err.Error())
		return
	}
	k.KCIncr(flock.KCResp)
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
func (k *nodeStuff) recvData(sess *Session, msg *Msg) *flock.Info {

	var info flock.Info
	var err error

	if sess.respKey != nil { // new session key pending?
		err = gobDecrypt(msg.Payload, &info, msg.hdrBits, sess.respKey)
		if err == nil { // rotate keys
			sess.sessKey = sess.respKey
			sess.respKey = nil
			k.KCIncr(flock.KCDataResp)
			return &info
		}
	}
	if sess.sessKey != nil { // usual case
		err = gobDecrypt(msg.Payload, &info, msg.hdrBits, sess.sessKey)
		if err == nil {
			k.KCIncr(flock.KCData)
			return &info
		}
	}
	if sess.prevKey != nil { // sent with the old key, maybe
		err = gobDecrypt(msg.Payload, &info, msg.hdrBits, sess.prevKey)
		if err == nil {
			k.KCIncr(flock.KCDataPrev)
			return &info
		}
	}
	if err == nil {
		k.Logf("recvData: no keys")
		k.KCIncr(flock.KCDataNoKey)
	} else {
		k.Logf("recvData: corrupt info Payload: %s", err.Error())
		k.KCIncr(flock.KCDataFail)
	}
	return nil
}

func (k *nodeStuff) recvMsg(pkt []byte) *flock.Info {
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
	pshdr := PseudoHdr{MsgType: msg.MsgType, SrcAddr: msg.SrcAddr, DstAddr: k.me.Addr}
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
		_, err = msg.cert.Verify(*k.verifyOpts)
		if err != nil {
			k.Logf("recvMsg: %s", err.Error())
			return nil
		}
		if !ed25519.VerifyMulti(msg.cert.PublicKey.(ed25519.PublicKey), sig, msg.hdrBits, pkt) {
			k.Logf("recvMsg: signature failure on msg type %d", msg.MsgType)
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
func (k *nodeStuff) Recv() *flock.Info {
	node := string(k.me.Addr)
	for {
		k.flock.Lock()
		if len(k.flock.inq[node]) >= 1 {
			pkt := <-k.flock.inq[node]
			k.flock.Unlock()
			ff := k.recvMsg(pkt)
			if ff != nil {
				return ff
			}
		} else {
			k.flock.Unlock()
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (k *nodeStuff) SetMe(me *flock.NodeID) {
	if me.Moniker == "" {
		me.Moniker = crux.SmallID()
	}
	me.Addr = "ip_" + me.Moniker
	k.me = *me
}

func (k *flockSuite) analyse(c *C, req chan bool, res chan string) {
	var stableT = time.Duration(2.5 * float64(k.f.NodePrune()))
	world := make(map[string]crux.MonInfo)
	reboot := make(map[string]bool)
	var genghis string
	var tstable, tstart time.Time
	log := clog.Log.Log("where", "analysis")

loop:
	for {
		select {
		case <-req:
			break loop
		case mi := <-k.f.info:
			switch mi.Op {
			case crux.JoinOp:
				log.Log("node", mi.Moniker, "flock", mi.Flock, "oflock", mi.Oflock, "change flock")
				//fmt.Printf("%s %s: %s -> %s\n", mi.T.Format(flock.Atime), mi.Moniker, mi.Oflock, mi.Flock)
			case crux.LeaderStartOp:
				world[mi.Flock] = mi
				delete(reboot, mi.Moniker)
				log.Log("node", mi.Moniker, "flock", mi.Flock, "oflock", mi.Oflock, "start new flock")
				//fmt.Printf("%s %s: start %s (old=%s)\n", mi.T.Format(flock.Atime), mi.Moniker, mi.Flock, mi.Oflock)
			case crux.LeaderHeartOp, crux.LeaderDeltaOp:
				old := world[mi.Flock]
				//fmt.Printf("%s heart: old=%s new=%s genghis=%s stable=%s\n", mi.T.Format(flock.Atime), old.String(), mi.String(), genghis, tstable.Format(flock.Atime))
				if old.SString() == mi.SString() {
					if old.SString() == genghis {
						if mi.T.After(tstable) {
							res <- fmt.Sprintf("flock %s has been stable for %s", genghis, mi.T.Sub(tstart))
							world = make(map[string]crux.MonInfo)
							tstable = mi.T.Add(100 * time.Minute) // prevent premature misfire
						}
					}
					continue
				}
				log.Log("flock", mi.Flock, "oldmem", old.N+1, "newmem", mi.N+1, "membership change")
				//fmt.Printf("%s %s: %d -> %d members\n", mi.T.Format(flock.Atime), mi.Flock, old.N+1, mi.N+1)
				world[mi.Flock] = mi
				if (mi.Op == crux.LeaderHeartOp) && (mi.N+1 == k.f.n) && (len(reboot) == 0) {
					genghis = mi.SString()
					tstart = mi.T
					tstable = mi.T.Add(stableT)
					log.Log(fmt.Sprintf("starting stable test %s\n", genghis))
				}
			case crux.ProbeOp:
				//fmt.Printf("%s %s: %d probes\n", mi.T.Format(flock.Atime), mi.Moniker, mi.N)
			case crux.RebootOp:
				reboot[mi.Moniker] = true
				log.Log("node", mi.Moniker, "nreboot", len(reboot), "rebooting")
				//fmt.Printf("%s rebooting; %d total\n", mi.Moniker, len(reboot))
			}
			// prune out old entries
			cutoff := mi.T.Add(time.Duration(-1.5 * float32(stableT)))
			for k, m := range world {
				if m.T.Before(cutoff) {
					delete(world, k)
				}
			}
		}
	}
	res <- "done"
}

// period for sending heartbeats
func (k *flockStuff) Heartbeat() time.Duration {
	return 500 * time.Millisecond
}

// period for changing epochID
func (k *flockStuff) KeyPeriod() time.Duration {
	return 10 * k.Heartbeat()
}

// lose flock membership if no heartbeats
func (k *flockStuff) NodePrune() time.Duration {
	return 3 * k.Heartbeat()
}

// forget old nodes after this
func (k *flockStuff) HistoryPrune() time.Duration {
	return 2 * k.Heartbeat()
}

// drop a checkpoint after this period
func (k *flockStuff) Checkpoint() time.Duration {
	return 1 * time.Minute
}

// period between rework probes
func (k *flockStuff) Probebeat() time.Duration {
	return 3 * k.Heartbeat()
}

// how many probes per rework
func (k *flockStuff) ProbeN() int {
	return 4
}

// return a random address within a given range
func (k *flockStuff) Probe() (string, bool) {
	if rand.Float32() >= k.density {
		return "", true
	}
	// now pick one at random
	i := rand.Intn(len(k.nodes))
	return k.nodes[i].me.Addr, true
}

// use nil for unused keys
func (k *nodeStuff) SetKeys(epochID *flock.Nonce, sec0, sec1 *flock.Key) {
	if k.sharedKey == nil && sec0 != nil {
		k.sharedKey = new(flock.Key)
		*k.sharedKey = *sec0
		k.Logf("node %s shared key %x", k.me.Addr, *k.sharedKey)
	}
	if k.epochID != epochID { // new session keys, please
		DBPrintf("new epochID %s : %s \n", k.me.Addr, epochID)
		k.Lock()
		k.epochID = epochID
		k.Logf("node %s new session keys", k.me.Addr)
		for _, sess := range k.sessions {
			if sess.sessKey != nil {
				sess.prevKey = sess.sessKey
				sess.sessKey = nil
			}
		}
		k.Unlock()
	}
}

// monitoring channel; nil is off
func (k *nodeStuff) Monitor() chan crux.MonInfo {
	return k.flock.info
}

// we're done
func (k *nodeStuff) Quit() {
}

// how we log stuff
func (k *nodeStuff) Logf(format string, args ...interface{}) {
	x := fmt.Sprintf(format, args...)
	k.flock.Lock()
	k.flock.interest = append(k.flock.interest, quirk{t: time.Now().UTC(), s: x})
	k.flock.Unlock()
	clog.Log.Log(nil, fmt.Sprintf("%s: %s", k.me.Addr, x))
}

// for shutting down
func (k *flockStuff) Quit() {

}
