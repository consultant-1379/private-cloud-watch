// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hague

package reeve

// This starts up reeve as the first plugin
// from the fulcrum perspective
// and presents a functional reeveAPI via an interface
// found in rucklib.ReeveAPI
// It provides the startup and hot-test for the ssh-agent
// and the grpcsig database (whitelist, endpoints)
// On restart - this sets up all the previously
// known current signers/keys for clients to use
// and reloads the local database

// The ReeveAPI functions exposed to fulcrum:
//   SetEndPtsHorde(string) string
//   ClientSigner(string) (interface{}, *Err)
//   PubKeysFromSigner(interface{}) (string, string)
//   SelfSigner() (interface{})
//   SecureService(string) interface{}
//   LocalPort() int
//   ReeveCallBackInfo() (string, string, string, string, interface{})
//   StartStewardIO() *Err
//   StopStewardIO()

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
)

// TODO on deprecating a client or key - needs to manage ssh-agent
// and in-mem signer struct.

const dd = "/"

// SignerMapT - map of client signers by servicerev string
type SignerMapT map[string]*grpcsig.ClientSignerT

// StateT - implements the ReeveAPI interface for the Reeve Server
// for passing around signers that can't go over gRPC.
type StateT struct {
	certificate      *c.TLSCert // may be nil
	cursigners       SignerMapT
	selfsigner       *grpcsig.AgentSigner
	imp              *grpcsig.ImplementationT
	dbname           string
	nodeid           string
	netid            string
	port             int
	pubkey           grpcsig.PubKeyT
	keyid            string
	pubkeyjson       string
	registryaddress  string
	stewardaddress   string
	stewardprincipal string
	stewardsigner    *grpcsig.AgentSigner
	catalog          []pb.CatalogInfo
	rules            []pb.RuleInfo
}

// GetCertificate - happy lint
func (s *StateT) GetCertificate() *c.TLSCert {

	return s.certificate
}

// ReeveState - holds the internal reeve state information.
var ReeveState *StateT

// AgentInit - initializes the .muck, ssh-agent and whitelist database system
// Logger passed here is only used for this initializtion process.
// Any errors sent back here are normally treated as fatal
func AgentInit(muckdir string, logger clog.Logger) (string, *c.Err) {
	if logger == nil {
		return "", c.ErrF("nil logger provided")
	}
	// establish a .muck, or set up existing one
	merr := muck.InitMuck(muckdir, "")
	if merr != nil {
		return "", c.ErrF("muck InitMuck failed : %v", merr)
	}

	// init self keys and ssh-agent
	kerr := grpcsig.InitSelfSSHKeys(false)
	if kerr != nil {
		return "", c.ErrF("grpcsig InitSelfSSHKeys failed : %v", kerr)
	}

	// quietly ensure sure ssh-agent can read keys
	_, lerr := grpcsig.ListKeysFromAgent(false)
	if lerr != nil {
		return "", c.ErrF("grpcsig ListKeysFromAgent failed accessing private keys with ssh-agent : %v", lerr)
	}

	dbfile := muck.Dir() + "/whitelist.db"
	if !grpcsig.PubKeyDBExists(dbfile) {
		derr := grpcsig.StartNewPubKeyDB(dbfile)
		if derr != nil {
			return "", c.ErrF("grpcsig StartnewPubKeyDB failed : %v", derr)
		}
	}

	ierr := grpcsig.InitPubKeyLookup(dbfile, logger)
	if ierr != nil {
		return "", c.ErrF("grpcsig InitPubKeyLookup failed : %v", ierr)
	}

	// Add self public key to whitelist DB (i.e. this test talks to itself!)
	kerr = grpcsig.AddPubKeyToDB(grpcsig.GetSelfPubKey())
	if lerr != nil {
		return "", c.ErrF("grpcsig AddPubKeyToDB failed with self key : %v", lerr)
	}

	grpcsig.FiniPubKeyLookup()

	return dbfile, nil
}

// Fini - intended to stop all the Reeve internals
func Fini() {
	pidstr, ts := grpcsig.GetPidTS()
	logger := ReeveState.imp.Logger
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "Reeve shutting down")
	// Remove self key from whitelist
	derr := grpcsig.RemoveSelfPubKeysFromDB()
	if derr != nil {
		msg1 := "Reeve Fini could not RemoveSelfPubkeysFromDB"
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg1)
	}
	// Remove the private key from ssh-agent
	ferr := grpcsig.FiniSelfSSHKeys(false)
	if ferr != nil {
		msg2 := fmt.Sprintf("Reeve Fini could not FiniSelfSSHKeys : %v", ferr)
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg2)
	}

	// Stop the lookup service
	grpcsig.FiniDefaultService()
	return
}

func startReeveError(msg string, logger clog.Logger) (*StateT, *c.Err) {
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "FATAL", "PID", pidstr, "TS", ts, msg)
	return nil, c.ErrF("%s", msg)
}

// StartReeveAPI - set up reeveAPI for organza
func StartReeveAPI(dbname, block, horde, node string, port int, stewardaddress string, cert *c.TLSCert, initlogger clog.Logger) (*StateT, *c.Err) {
	// Make an ImplementationT for grpc-signatures keyid lookup and validation
	logger := clog.Log.With("focus", ReeveRev, "mode", "grpc-signatures")
	imp, ierr := grpcsig.InitDefaultService(dbname, ReeveRev, cert, logger, 300, false)
	if ierr != nil {
		msg2 := fmt.Sprintf("StartReeveAPI fatal error - grpcsig InitDefaultService failed : %v", ierr)
		return startReeveError(msg2, initlogger)
	}
	// Get our SelfSinger
	selfsigner, qerr := grpcsig.SelfSigner(cert)
	if qerr != nil {
		msg3 := fmt.Sprintf("StartReeveAPI fatal error - grpcsig SelfSigner failed : %v", qerr)
		return startReeveError(msg3, initlogger)
	}
	// Make or find our current reeve keys
	pk, kerr := InitReeveKeys("", false)
	if kerr != nil {
		msg4 := fmt.Sprintf("StartReeveAPI fatal error - grpcsig InitReeveKeys failed : %v", kerr)
		return startReeveError(msg4, initlogger)
	}
	// need json for registration
	pkjson, jerr := grpcsig.PubKeyToJSON(pk)
	if jerr != nil {
		msg5 := fmt.Sprintf("StartReeveAPI fatal error - grpcsig PubKeyToJSON failed : %v", jerr)
		return startReeveError(msg5, initlogger)
	}

	reeve := StateT{}
	reeve.certificate = cert
	reeve.cursigners = make(map[string]*grpcsig.ClientSignerT)
	reeve.selfsigner = selfsigner
	reeve.dbname = dbname
	reeve.pubkey = *pk
	reeve.keyid = pk.KeyID
	reeve.pubkeyjson = pkjson
	reeve.imp = &imp
	reeve.stewardaddress = stewardaddress

	// Load in all the existing current pubkeys for all services in .muck
	serr := SignersFromCurrentPubkeys(&reeve, false)
	if serr != nil {
		msg6 := fmt.Sprintf("StartReeveAPI fatal error - SignersFromCurrentPubkeys failed : %v", serr)
		return startReeveError(msg6, initlogger)
	}
	nod, ferr := idutils.NewNodeID(block, horde, node, ReeveName, ReeveAPI)
	if ferr != nil {
		msg7 := fmt.Sprintf("StartReeveAPI fatal error in idutils NewNodeID : %v", ferr)
		return startReeveError(msg7, initlogger)
	}
	reeve.nodeid = nod.String()
	principal, _ := muck.Principal()
	nid, nerr := idutils.NewNetID(ReeveRev, principal, node, port)
	if nerr != nil {
		msg8 := fmt.Sprintf("StartReeveAPI fatal error in idutils NewNetID : %v", nerr)
		return startReeveError(msg8, initlogger)
	}
	reeve.netid = nid.String()
	reeve.port = port

	// Set up (new or existing) keys and signer for steward - public key gets sent to the registry for
	// the steward whitelist
	stewardclisignif, kerr := reeve.ClientSigner(StewardRev)
	if kerr != nil {
		msg9 := fmt.Sprintf("StartReeeveAPI fatal error in reeve ClientSigner -  cannot make steward access key pair : %v", kerr)
		return startReeveError(msg9, initlogger)
	}

	var clisign *grpcsig.ClientSignerT
	clisign = *stewardclisignif
	reeve.stewardsigner = clisign.Signer
	ReeveState = &reeve
	return &reeve, nil
}

func reeveLaunchError(msg string, logger clog.Logger) *c.Err {
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
	return c.ErrF(msg)
}

// Launch - launches reeve server for organza boot up
func Launch(reevenod idutils.NodeIDT, reevenid idutils.NetIDT, impinterface **grpcsig.ImplementationT, stopch *chan bool) *c.Err {
	// log this startup function to ReeveLaunch
	logger := clog.Log.With("focus", "ReeveLaunch")

	imp := *impinterface
	// Start reeve server proper
	s := imp.NewServer()
	pb.RegisterReeveServer(s, &server{})
	grpc_prometheus.Register(s)
	lis, lerr := net.Listen("tcp", reevenid.Port)
	if lerr != nil {
		msg2 := fmt.Sprintf("ReeveLaunch() failed - in net.Listen : %v", lerr)
		return reeveLaunchError(msg2, logger)
	}
	// Ready to serve
	go s.Serve(lis)
	// Put up the stoping function
	stopfn := func(server *grpc.Server, nod idutils.NodeIDT, nid idutils.NetIDT, logger clog.Logger, stop *chan bool) {
		msg1 := fmt.Sprintf("%s GracefulStop Service  %s", nod.String(), nid.String())
		pidstr, ts := grpcsig.GetPidTS()
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg1)
		<-*stop
		server.GracefulStop()
		lis.Close()
		msg2 := fmt.Sprintf("%s Service Stopped  %s", nod.String(), nid.String())
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg2)
	}
	go stopfn(s, reevenod, reevenid, logger, stopch)
	msg3 := fmt.Sprintf("%s Serving %s", reevenod.String(), reevenid.String())
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg3)
	return nil
}

// StartReeve - starts up reeve service, .muck storage, checks and
// initializes signers that are found on start. Upstream should
// normally consider any errors returned to be fatal.
func StartReeve(muckdir, bloc, horde, node string, port int, stewardaddress string, cert *c.TLSCert, initlogger clog.Logger) (*StateT, *c.Err) {
	// Start or Re-Start .muck, ssh-agent, self-keys, grpc-signatures database
	dbname, err := AgentInit(muckdir, initlogger)
	if err != nil {
		msg1 := fmt.Sprintf("StartReeve fatal error in AgentInit : %v", err)
		return startReeveError(msg1, initlogger)
	}
	// Make an ImplementationT for grpc-signatures keyid lookup and validation
	logger := clog.Log.With("focus", ReeveRev, "mode", "grpc-signatures")
	imp, ierr := grpcsig.InitDefaultService(dbname, ReeveRev, cert, logger, 300, false)
	if ierr != nil {
		msg2 := fmt.Sprintf("StartReeve fatal error - grpcsig InitDefaultService failed : %v", ierr)
		return startReeveError(msg2, initlogger)
	}
	// Get our SelfSinger
	selfsigner, qerr := grpcsig.SelfSigner(cert)
	if qerr != nil {
		msg3 := fmt.Sprintf("StartReeve fatal error - grpcsig SelfSigner failed : %v", qerr)
		return startReeveError(msg3, initlogger)
	}
	// Make or find our current reeve keys
	pk, kerr := InitReeveKeys("", false)
	if kerr != nil {
		msg4 := fmt.Sprintf("StartReeve fatal error - grpcsig InitReeveKeys failed : %v", kerr)
		return startReeveError(msg4, initlogger)
	}
	// need json for registration
	pkjson, jerr := grpcsig.PubKeyToJSON(pk)
	if jerr != nil {
		msg5 := fmt.Sprintf("StartReeve fatal error - grpcsig PubKeyToJSON failed : %v", jerr)
		return startReeveError(msg5, initlogger)
	}

	reeve := StateT{}
	reeve.cursigners = make(map[string]*grpcsig.ClientSignerT)
	reeve.selfsigner = selfsigner
	reeve.dbname = dbname
	reeve.pubkey = *pk
	reeve.keyid = pk.KeyID
	reeve.pubkeyjson = pkjson
	reeve.imp = &imp
	reeve.stewardaddress = stewardaddress

	// Load in all the existing current pubkeys for all services in .muck
	serr := SignersFromCurrentPubkeys(&reeve, false)
	if serr != nil {
		msg6 := fmt.Sprintf("StartReeve fatal error - SignersFromCurrentPubkeys failed : %v", serr)
		return startReeveError(msg6, initlogger)
	}
	nod, ferr := idutils.NewNodeID(bloc, horde, node, ReeveName, ReeveAPI)
	if ferr != nil {
		msg7 := fmt.Sprintf("StartReeve fatal error in idutils NewNodeID : %v", ferr)
		return startReeveError(msg7, initlogger)
	}
	reeve.nodeid = nod.String()
	principal, _ := muck.Principal()
	nid, nerr := idutils.NewNetID(ReeveRev, principal, node, port)
	if nerr != nil {
		msg8 := fmt.Sprintf("StartReeve fatal error in idutils NewNetID : %v", nerr)
		return startReeveError(msg8, initlogger)
	}
	reeve.netid = nid.String()
	reeve.port = port

	// Set up (new or existing) keys and signer for steward - public key gets sent to the registry for
	// the steward whitelist
	stewardclisignif, kerr := reeve.ClientSigner(StewardRev)
	if kerr != nil {
		msg9 := fmt.Sprintf("StartReeeve fatal error in reeve ClientSigner -  cannot make steward access key pair : %v", kerr)
		return startReeveError(msg9, initlogger)
	}

	var clisign *grpcsig.ClientSignerT
	clisign = *stewardclisignif
	reeve.stewardsigner = clisign.Signer
	ReeveState = &reeve

	// Start reeve server proper
	s := reeve.imp.NewServer()
	pb.RegisterReeveServer(s, &server{})
	grpc_prometheus.Register(s)
	lis, oerr := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if oerr != nil {
		msg11 := fmt.Sprintf("StartReeve failed in net.Listen : %v", oerr)
		pidstr, ts := grpcsig.GetPidTS()
		initlogger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg11)
		return nil, c.ErrF("%s", msg11)
	}
	// Ready to serve
	go s.Serve(lis)
	// Pause for handy MacOS X firewall dialogue to appear & clicky-clicky
	if runtime.GOOS == "darwin" {
		time.Sleep(4 * time.Second)
	}
	// log that reeve has started
	msg12 := fmt.Sprintf("%s Serving %s", reeve.nodeid, reeve.netid)
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg12)
	return &reeve, nil
}

// LocalPort - returns the local port for plugins to figure out localhost:port grpc access
func (s *StateT) LocalPort() int {
	return s.port
}

// GetNetIDString - Make it clear we're returning the stringification, not a NetIDT struct.
func (s *StateT) GetNetIDString() string {
	return s.netid
}

// ReeveCallBackInfo - returns info about this node's reeve server needed for register to call back
func (s *StateT) ReeveCallBackInfo() (string, string, string, string, **grpcsig.ImplementationT) {

	return s.nodeid, s.netid, s.keyid, s.pubkeyjson, &s.imp
}

// StartStewardIO - wait time is the timeout value for connecting this Reeve to Steward
func (s *StateT) StartStewardIO(wait time.Duration) *c.Err {
	cerr := clientsLocalIni()
	if cerr != nil {
		msg1 := fmt.Sprintf("StartStewardIO failed in ClientsLocalIni : %v", cerr)
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg1)
		return c.ErrF("%v", msg1)
	}
	perr := endpointsLocalIni()
	if perr != nil {
		msg2 := fmt.Sprintf("StartStewardIO failed in EndpointsLocalIni : %v", perr)
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg2)
		return c.ErrF("%v", msg2)
	}
	err := wakeUpSteward(wait)
	if err != nil {
		msg2 := fmt.Sprintf("StartStewardIO - Reeve has no Steward Connectivity : %v", err)
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg2)
		return c.ErrF("%v", msg2)
	}
	pidstr, ts := grpcsig.GetPidTS()
	s.imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "Steward is WOKE")
	ReeveEvents = startIngest()
	return nil
}

// StopStewardIO - stops reeve ingest event loop
func (s *StateT) StopStewardIO() {
	ReeveEvents.Quit()
}

// SelfSigner - returns the self signer for inter-process gRPC calls
func (s *StateT) SelfSigner() **grpcsig.ClientSignerT {
	clientsigner := &grpcsig.ClientSignerT{}
	clientsigner.Signer = s.selfsigner
	return &clientsigner
}

// ClientSigner - is what registers clients, communicates with Steward, manages signers.
// This is not possible to do from gRPC over a network connection
func (s *StateT) ClientSigner(srvcRev string) (**grpcsig.ClientSignerT, *c.Err) {
	// Look in s.cursigners first
	if cs, ok := s.cursigners[srvcRev]; ok {
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("ClientSigner retrieved for %s", srvcRev))
		return &cs, nil
	}

	// Not there, let's make one in .muck - it will get reloaded on restart
	pk, kerr := MakeServiceKeys(srvcRev, "", false)
	if kerr != nil {
		msg1 := fmt.Sprintf("ClientSigner failed on MakeServiceKeys for %s : %v", srvcRev, kerr)
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
		return nil, c.ErrF("%v", msg1)
	}

	// set it up in the ssh-agent
	aerr := AddCurrentKeyToAgent(pk.KeyID, false)
	if aerr != nil {
		msg2 := fmt.Sprintf("ClientSigner failed to add keyid %s to ssh-agent : %v", pk.KeyID, aerr)
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg2)
		return nil, c.ErrF("%v", msg2)
	}

	// add it to the in-memory list
	cerr := addClientSignerFromPubKey(pk.KeyID, pk, s)
	if cerr != nil {
		msg3 := fmt.Sprintf("ClientSigner failed adding  %s to cursigners list : %v ", pk.KeyID, cerr)
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg3)
		return nil, c.ErrF("%v", msg3)
	}

	// Now, try that again
	if cs, ok := s.cursigners[srvcRev]; ok {
		pidstr, ts := grpcsig.GetPidTS()
		s.imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("ClientSigner added for %s", srvcRev))
		return &cs, nil
	}

	// should be unreachable code, but whatever
	return nil, c.ErrF("ClientSigner failed to add a signer - unknown error")
}

// SetEndPtsHorde - sets current horde name in persistent storage for all endpoints
// running under this reeve.
// pass a string to set it (overwrites previous hordename)
// pass "" to get the current persistent horde name for all endpoints.
func (s *StateT) SetEndPtsHorde(hordename string) string {
	return muck.HordeName(hordename)
}

// PubKeysFromSigner - utility to extract the KeyID, JSON from a ClientSigner interface
func (s *StateT) PubKeysFromSigner(in **grpcsig.ClientSignerT) (string, string) {
	psigner := *in
	return psigner.PubKey.KeyID, psigner.PubKeyJSON
}

// SecureService - returns a grpcsig db lookup implementation for service "serviceRev"
func (s *StateT) SecureService(svcRev string) **grpcsig.ImplementationT {
	logger := clog.Log.With("focus", svcRev, "mode", "grpc-signatures")
	imp := &grpcsig.ImplementationT{
		PubKeyLookupFunc: s.imp.PubKeyLookupFunc,
		Service:          svcRev,
		Logger:           logger,
		ClockSkew:        s.imp.ClockSkew,
		LookupResource:   s.imp.LookupResource,
		Algorithms:       s.imp.Algorithms,
		Started:          s.imp.Started,
		Certificate:      s.imp.Certificate,
	}
	return &imp
}

// SignersFromCurrentPubkeys - called on process start - recovers
// from .muck any public keys and signers we need to get going
func SignersFromCurrentPubkeys(s *StateT, debug bool) *c.Err {
	currentdir := muck.CurrentKeysDir()
	pubfiles, err := walkDir(currentdir)
	if err != nil {
		return err
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
			return c.ErrF("failed to load current pubkey %s - %v", keyfile, err)
		}
		kid, kerr := idutils.KeyIDParse(keyID)
		if kerr != nil {
			return c.ErrF("bad keyid : %v", kerr)
		}
		pubkey.Service = kid.ServiceRev
		pubkey.Name = kid.Principal
		// TODO double check filesystem fingerprint is as recalculated from actual public key
		pubkey.KeyID = keyID

		aerr := AddCurrentKeyToAgent(keyID, false)
		if aerr != nil {
			return c.ErrF("failed to add keyid %s to ssh-agent: %v", keyID, aerr)
		}

		cerr := addClientSignerFromPubKey(keyID, pubkey, s)
		if cerr != nil {
			return c.ErrF("error - failed to add keyid %s to ssh-agent", keyID)
		}

	}
	return nil
}

// addClientSignerFromPubKey - makes a new signer. keyID must already be put
// into the ssh-agent with a previous call to AddCurrentKeyToAgent.
func addClientSignerFromPubKey(keyID string, pubkey *grpcsig.PubKeyT, s *StateT) *c.Err {
	kid, kerr := idutils.KeyIDParse(keyID)
	if kerr != nil {
		return c.ErrF("bad keyid : %v", kerr)
	}
	agentsigner, aerr := grpcsig.NewAgent(kid)
	if aerr != nil {
		return c.ErrF("unable to make ssh-agent signer : %v", aerr)
	}
	agentsigner.Certificate = s.imp.Certificate
	pkjson, jerr := grpcsig.PubKeyToJSON(pubkey)
	if jerr != nil {
		return jerr
	}
	clientsigner := grpcsig.ClientSignerT{}
	clientsigner.Signer = agentsigner
	clientsigner.PubKey = *pubkey
	clientsigner.PubKeyJSON = pkjson
	// Add it to the map so we can find it by ServiceRev
	s.cursigners[kid.ServiceRev] = &clientsigner
	return nil
}
