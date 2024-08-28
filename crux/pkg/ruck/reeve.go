package ruck

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/flock"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/register"
	"github.com/erixzone/crux/pkg/rucklib"
	"github.com/erixzone/crux/pkg/steward"
)

const reevePort int = 50059
const regPort int = 50060
const stewPort int = 50061

// Starts Reeve service, returning the non-gRPC part of the interface
// This is fatal on error - we can't start up until resolved.
func newReeveAPI(flock, horde, node string, port int, stewaddress string, cert *crux.TLSCert, logger clog.Logger) *reeve.StateT {
	logger.Log(nil, "about to reeveAPI addr=%s", stewaddress)
	reeve, err := reeve.StartReeve("", flock, horde, node, port, stewaddress, cert, logger)
	logger.Log(nil, "did reeveAPI addr=%s; err=%+v", stewaddress, err)
	if err != nil {
		fmt.Printf("Fatal -  reeve cannot start: %v", err)
		crux.Exit(1)
	}
	return reeve
}

// Organza reeve starter
// Not fatal on error - let workflow fail back to start with error message
func startReeveAPI(dbname, block, horde, node string, port int, stewaddress string, cert *crux.TLSCert, logger clog.Logger) (*reeve.StateT, error) {
	reeve, err := reeve.StartReeveAPI(dbname, block, horde, node, port, stewaddress, cert, logger)
	if err != nil {
		msg := fmt.Sprintf("Failed to StartReeve() : %v", err)
		logger.Log("fatal", msg)
		return nil, fmt.Errorf(msg)
	}
	logger.Log("info", "reeveAPI started")
	return reeve, nil
}

// FlockWait -- Waits for stable emergence of flock leader, steward, reeve host.
func FlockWait(c crux.Confab, sleeptime, minlimit, maxlimit time.Duration, regkey string, logboot clog.Logger) (leader, me, regaddress, stewaddress, fkey string) {
	var cluster string
	var stable bool
	start := time.Now()
	var wait1, total time.Duration
	iter := 0
	defer logboot.Log(nil, "reeve done")
	// Attempts to give the flocking protocol time to restart if muliple leaders emerge
	// Flock leader and host of registry/steward must be non-0, self-consistent to escape
	for {
		// Delay 1: Wait a minimum time for stable flag to emerge from flocking protocol
		start1 := time.Now()
		j := 0
		for {
			j++
			time.Sleep(sleeptime)
			cn := c.GetNames()
			cluster, leader, stable, me = cn.Bloc, cn.Leader, cn.Stable, cn.Node

			logboot.Log("FlockWait() c.GetNames()", fmt.Sprintf("%d.%d", iter, j), "cluster", cluster, "leader", leader, "stable", stable, "me", me)
			wait1 = time.Now().Sub(start1)
			if wait1 > minlimit {
				if stable {
					break
				}
			}
			total = time.Now().Sub(start)
			if total > maxlimit {
				logboot.Log(nil, "FlockWait() Delay 1 iter %d - Flock not stable after %s- aborting service start", iter, total.String())
				crux.Exit(1)
			}
		}

		// Skip first time -
		// test previously circulated ports for consistency with leader, then we can proceed
		if iter > 0 {
			cn := c.GetNames()
			regaddress, fkey, stewaddress = cn.RegistryAddr, cn.RegistryKey, cn.Steward
			logboot.Log("FlockWait() c.GetRegistry()", fmt.Sprintf("%d", iter), "registry", regaddress, "flockkey", fkey, "steward", stewaddress)
			// Did we recover the intended port numbers (non-0) back from flocking?
			if stewPort == idutils.SplitPort(stewaddress) && regPort == idutils.SplitPort(regaddress) {
				// Is the host for steward/registry same as the leader?
				if leader == idutils.SplitHost(stewaddress) && leader == idutils.SplitHost(regaddress) {
					if leader == me {
						logboot.Log(nil, fmt.Sprintf("FlockWait() %d I AM LEADER", iter))
					}
					// Commit to this being stable for now, bring up services
					return // with named variables
				}
			}
		}

		// On first iteration or when last iteration was not consistent
		// Current Leader "candidate" injects proposed registry/steward values
		if leader == me {
			logboot.Log(nil, fmt.Sprintf("FlockWait() %d I THINK I MAY BE LEADER", iter))
			if len(fkey) == 0 {
				c.SetRegistry(me, regPort, regkey)
			} else {
				c.SetRegistry(me, regPort, fkey)
			}
			c.SetSteward(me, stewPort)
		}
		iter++
	}
}

// reeve0_1 is a standin for the true startup routines. this is work TBD.
// port :  the port for flocking UDP
// skey :  the flocking key (cmd argument --key)
// ipname :  is the hostname of the node we are on
// ip is : ip address of the host or resolvable hostname of the host we are on
// beacon : is an Address (ip:port) intended as the flock leader (cmd argument --beacon)
// horde : is the name of the horde on which this process's endpoints are running.
// networks: a list of CIDR networks to probe. If this is blank, we probe the local network of the given ip.
func reeve0_1(port int, skey, ipname, ip, beacon, horde, networks string, net *flock.Flock) (*reeve.StateT, *crux.Err) {
	logboot := clog.Log.With("focus", "flock_boot", "node", ipname)

	logboot.Log("node", ipname, "in reeve0_1 bootstrap")
	defer logboot.Log("node", ipname, "done reeve0_1 bootstrap")

	// Start the Flock service, returns a *flock.Flock from
	// flock.NewFlockNode
	// net := newFlock(port, skey, ipname, ip, beacon, networks)

	// This makes the *flock.Flock a crux.Confab interface
	// with methods to Get/Set stuff (Leader, Me, Register, Steward)
	var c crux.Confab = net
	// cc := &c
	// Now cc is a pointer pointer.

	// Wait for the flock to stablize, restart flocking if necessary,
	// declare consistent leadership and locations for startup of registry & steward
	sleeptime := time.Duration(2.1 * float32(net.GetFflock().Heartbeat())) // just a little over two heartbeats should be good
	minlimit := 8 * net.GetFflock().NodePrune()                            // minimum time limit for stability to emerge
	maxlimit := 200 * time.Second                                          // max time limit for stability to emerge
	reevekey := skey
	leader, me, regaddress, stewaddress, fkey := FlockWait(c, sleeptime, minlimit, maxlimit, reevekey, logboot)

	// Start Reeve service, return an interface to its non-grpc services
	// for client grpcsig signing, and for server public key database lookups
	// which are local, pointer based structs that are passed via interface{}
	logboot.Log("node", ipname, "Starting Reeve")
	var reeveapi rucklib.ReeveAPI

	// here - if this fails, program exits with an error
	logreeve := clog.Log.With("focus", "srv_reeve", "node", ipname)

	reeveapiif := newReeveAPI("flock", horde, me, reevePort, stewaddress, c.GetCertificate(), logreeve)
	reeveapi = reeveapiif

	// Information about our reeve service - note it is not yet talking to steward!
	reevenodeid, reevenetid, reevekeyid, reevepubkeyjson, reeveimp := reeveapi.ReeveCallBackInfo()
	if reeveimp == nil {
		logboot.Log("node", ipname, "fatal", "Failed reeveapi.ReeveCallBackInfo")
		crux.Exit(1)
	}
	reevenid, ierr := idutils.NetIDParse(reevenetid)
	if ierr != nil {
		logboot.Log("node", ipname, "fatal", "failed to parse reevenetid : %v", ierr)
		crux.Exit(1)
	}
	reevenod, merr := idutils.NodeIDParse(reevenodeid)
	if merr != nil {
		logboot.Log("node", ipname, "fatal", "failed to parse reevenodeid : %v", merr)
		crux.Exit(1)
	}

	// Get our principal (retrieves from .muck if there)
	principal, derr := muck.Principal()
	if derr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error : %v", derr))
		crux.Exit(1)
	}
	var stewstartedts string
	var stewnod idutils.NodeIDT
	var stewnid idutils.NetIDT
	// Fire up Registry Server and Steward Server - if we are Leader
	if leader == me {
		logboot.Log("Starting Registry and Steward")

		// make a nodeid for Registry
		regnodeid, ferr := idutils.NewNodeID("flock", horde, me, register.RegistryName, register.RegistryAPI)
		if ferr != nil {
			logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - invalid nodeid params for registry: %v", ferr))
			crux.Exit(1)
		}

		// Usually we would call reeveapi.SecureService(servicerev) for a service
		// but here, registry is not grpcsig secured. It has a reverse mechanism,
		// where it needs to do lookups based on the reeve servicerev, so...
		// We need a SecureService interface{} for reeve callback authentication
		imp := reeveapi.SecureService(reevenid.ServiceRev)
		if imp == nil {
			logboot.Log("node", ipname, "fatal", "Failed reeveapi.SecureService")
			crux.Exit(1)
		}
		// This is how long we give a reeve to execute a grpc callback
		reevetimeout := 10 * time.Second
		// Now we can run the registry server
		rerr := register.RegistryInit(regnodeid, regaddress, stewaddress, fkey, reevetimeout, imp)
		if rerr != nil {
			logboot.Log("node", ipname, "fatal", rerr.String(), "stack", rerr.Stack)
			crux.Exit(1)
		}

		// Start up Steward on the LEADER, keep it running (don't end at close of bootstrap...)
		// set the variables as above.

		// make a nodeid for Steward
		var werr *crux.Err
		stewnod, werr = idutils.NewNodeID("flock", horde, me, steward.StewardName, steward.StewardAPI)
		if werr != nil {
			logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - invalid nodeid params for steward: %v", werr))
			crux.Exit(1)
		}

		// Set the netid
		var eerr *crux.Err
		stewnid, eerr = idutils.NewNetID(steward.StewardRev, principal, me, stewPort)
		if eerr != nil {
			logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - invalid netid params for steward: %v", eerr))
			crux.Exit(1)
		}

		// We need a SecureService interface{} for steward grpcsig authentication
		stewimp := reeveapi.SecureService(steward.StewardRev)
		if stewimp == nil {
			logboot.Log("node", ipname, "fatal", "error - failed reeveapi.SecureService for steward")
			crux.Exit(1)
		}
		// Ok go
		serr := steward.StartSteward(stewnod, stewnid, stewimp)
		if serr != nil {
			logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - did not StartSteward: %v", eerr))
			crux.Exit(1)
		}
		stewstartedts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	}

	// Now we can register this node's Reeve Service on the Registry Server
	// allowing this Reeve to communicate with the centralized Steward Service
	// This works also on the node hosting the Registry & Steward Servers themselfves

	// Make a client to talk to the Registry Server, holding our reeve
	// callback information

	logboot.Log("node", ipname, "info", "Making RegisterClient")

	var reg crux.RegisterClient
	registercli := newRegisterClient(reevenodeid, reevenetid, reeveimp)
	if registercli == nil {
		logboot.Log("node", ipname, "fatal", "Failed to get newRegisterClient")
		crux.Exit(1)
	}
	reg = registercli // handle the interface

	// Now we can call the Register server and
	// invoke the Registration Method to register our
	// reeve with the flock, executing
	// the two-way exchange of public keys
	logboot.Log("node", ipname, "info", fmt.Sprintf("Registering our Reeve with: %s, %v", regaddress, reevepubkeyjson))
	gerr := reg.AddAReeve(regaddress, fkey, reevepubkeyjson)
	if gerr != nil {
		// Fatal if we cannot register
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("Could not AddAReeve() %v", gerr))
		crux.Exit(1)
	}

	// Establish that reeeve - to - steward gRPC communication works.
	// (i.e. it PingSleep blocks until Steward node appears)
	stewtimeout := 60 * time.Second
	stewerr := reeveapi.StartStewardIO(stewtimeout)
	if stewerr != nil {
		// Fatal if we cannot talk to steward
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("StartStewardIO failed after %v - %v", stewtimeout, stewerr))
		crux.Exit(1)
	}
	reevestartedts := time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	logboot.Log("node", ipname, "info", "This reeve can communicate with steward")

	// Henceforth Reeve can talk to Steward.

	// We will now register Reeve with Steward via itself  formally.
	// Get the self-signer

	selfsig := reeveapi.SelfSigner()
	// Dial the local gRPC client
	reeveclien, cerr := reeve.OpenGrpcReeveClient(reevenid, selfsig, logboot)
	if cerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - local reeve grpc client failed - %v", cerr))
		crux.Exit(1)
	}

	// Construct what we need to RegisterEndpoint
	reeveep := pb.EndpointInfo{
		Tscreated: reevestartedts,
		Tsmessage: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Status:    pb.ServiceState_UP,
		Nodeid:    reevenod.String(),
		Netid:     reevenid.String(),
		Filename:  reeve.ReeveRev, // any plugin file hash goes here
	}

	// Make the gRPC call to local reeve to register itself
	ackE, xerr := reeveclien.RegisterEndpoint(context.Background(), &reeveep)
	if xerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - RegisterEndpoint failed for reeve: %v", xerr))
		crux.Exit(1)
	}
	logboot.Log("node", ipname, "info", fmt.Sprintf("reeve is endpoint registered with reeve: %v", ackE))

	// Now, Register reeve's steward client, formally

	reevecl := pb.ClientInfo{
		Nodeid:  reevenod.String(),
		Keyid:   reevekeyid,
		Keyjson: reevepubkeyjson,
		Status:  pb.KeyStatus_CURRENT,
	}

	// Make the gRPC call to local reeve to register itself
	ackC, yerr := reeveclien.RegisterClient(context.Background(), &reevecl)
	if yerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - RegisterClient failed for reeve: %v", xerr))
		crux.Exit(1)
	}
	logboot.Log("node", ipname, "info", fmt.Sprintf("reeve client is registered with reeve: %v", ackC))

	// Finally, if we are running Steward, we need to register it, formally.
	// It is an endpoint and runs as a client of reeve.
	if leader == me {
		// Construct what we need to RegisterEndpoint
		stewep := pb.EndpointInfo{
			Tscreated: stewstartedts,
			Tsmessage: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
			Status:    pb.ServiceState_UP,
			Nodeid:    stewnod.String(),
			Netid:     stewnid.String(),
			Filename:  steward.StewardRev, // any plugin file hash goes here
		}

		// Make the gRPC call to local reeve to register itself
		ackE, xerr := reeveclien.RegisterEndpoint(context.Background(), &stewep)
		if xerr != nil {
			logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - RegisterEndpoint failed for steward: %v", xerr))
			crux.Exit(1)
		}
		logboot.Log("node", ipname, "info", fmt.Sprintf("steward is endpoint registered with reeve: %v", ackE))

		// Now, Register steward's reeve client, formally - whose public keys are held in register for
		// bootstrapping purposes.
		stewcl := pb.ClientInfo{
			Nodeid:  stewnod.String(),
			Keyid:   register.GetStewardKeyID(),
			Keyjson: register.GetStewardPubkeyJSON(),
			Status:  pb.KeyStatus_CURRENT,
		}

		// Make the gRPC call to local reeve to register itself
		ackC, yerr := reeveclien.RegisterClient(context.Background(), &stewcl)
		if yerr != nil {
			logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - RegisterClient failed for steward: %v", xerr))
			crux.Exit(1)
		}
		logboot.Log("node", ipname, "info", fmt.Sprintf("steward client is registered with reeve: %v", ackC))
	}

	// At this point our reeve - register/steward infrastructure is ready
	return reeveapiif, nil
}
