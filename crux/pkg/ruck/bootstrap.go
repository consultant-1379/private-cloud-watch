package ruck

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	ruck "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/flock"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/pastiche"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/register"
	rl "github.com/erixzone/crux/pkg/rucklib"
	"github.com/erixzone/crux/pkg/steward"
)

// RS is the import/export piece connecting organza stuff to us
type RS struct {
	sync.Mutex
	Reeveapi *reeve.StateT
	Net      *flock.Flock
	Conf     *crux.Confab
	Ipname   string
	IP       string
}

// Rs is the var of the above
var Rs RS

func godot(bloc string, port int, skey, ipname, ip, horde, beacon, networks, certdir string, visitor bool) {
	fmt.Printf("launching bq\n")
	go BootstrapOrganzaX("block", port, skey, ipname, ip, horde, beacon, networks, certdir)
	epoch := time.Now().UTC()
	tick := time.Tick(5 * time.Second)
loop:
	for {
		select {
		case <-tick:
			Rs.Lock()
			clog.Log.Log(nil, "checking bq at %s: reeveapi=%p confab=%p", time.Now().UTC().Sub(epoch).String(), Rs.Reeveapi, Rs.Conf)
			if (Rs.Reeveapi != nil) && (Rs.Conf != nil) {
				Rs.Unlock()
				break loop
			}
			Rs.Unlock()
		}
	}
	fmt.Printf("bq done\n")
}

// Bootstrap is a transient vestige, does flocking only
func Bootstrap(port int, skey, ipname, ip, horde, beacon, networks, certdir string, visitor bool) {
	logboot := clog.Log.With("focus", "flock_boot", "node", ipname)

	logboot.Log("node", ipname, "in bootstrap")
	defer logboot.Log("node", ipname, "done bootstrap")

	// Step 1: start the Flock service, returns a *flock.Flock
	net := newFlock(port, skey, ipname, ip, beacon, networks, certdir, visitor)
	logboot.Log(nil, "newFlock returned %v", net)
	time.Sleep(60 * time.Second)

	// we're done
	crux.Exit(0)
}

// BootstrapOrganza is the current link to organza
func BootstrapOrganza(bloc string, port int, skey, ipname, ip, horde, beacon, networks, certdir, hordespec string, visitor bool) {
	blocName := "bloc" // at some point, figure out how to assign this
	// start logging
	logboot := clog.Log.With("focus", "flock_boot", "node", ipname)
	logboot.Log("node", ipname, "in bootstrap")
	defer logboot.Log("node", ipname, "done bootstrap")

	// start flocking
	//net := newFlock(port, skey, ipname, ip, beacon, networks)
	godot(bloc, port, skey, ipname, ip, horde, beacon, networks, certdir, visitor)
	Rs.Lock()
	net := Rs.Net
	conf := Rs.Conf
	reeveapi := Rs.Reeveapi
	ipname = Rs.Ipname
	ip = Rs.IP
	Rs.Unlock()
	logboot.Log(nil, "bq progress")

	// Step 1: for completeness, start heartbeating the bootstrap
	myUUID := crux.SmallID()
	(net.Monitor()) <- crux.MonInfo{
		Op:      crux.HeartBeatOp,
		Moniker: myUUID,
		T:       time.Now().UTC(),
		Oflock:  fmt.Sprintf("%s %s %s", ipname, "builtin", "Bootstrap"),
	}
	logboot.Log(nil, "heartbeating bootstrap: UUID=%s", myUUID)

	// this shouldn't be here; start reeve (and steward) so that everything else can start
	// Step 2: start reeve and steward
	//reeveapi, err := reeve0_1(port, skey, ipname, ip, beacon, horde, networks, net)
	//crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "reeveapi=%p", reeveapi)
	sampleNOD, err := idutils.NewNodeID(blocName, horde, ipname, FlockName, FlockAPI)
	crux.FatalIfErr(logboot, err)

	for {
		// read flocking status
		node := (*conf).GetNames()
		logboot.Log(nil, "bqq: getnames=%+v", node)
		// are we at ground zero?
		if (node.Horde == "") && node.Stable && (node.Yurt == "") && (node.Leader == node.Node) {
			// nothing set and we are stable; become the new ADMIN horde
			net.SetHorde("+" + Admin)
			continue
		}
		// are we just starting a new horde?
		if (len(node.Horde) > 0) && (node.Horde[0] == '+') {
			// start a new horde
			node.Horde = node.Horde[1:] // fix the name
			logboot.Log(nil, "starting a new horde %s", node.Horde)
			deanClient, hbnid := startNodeServices(logboot, net, sampleNOD, reeveapi)
			if node.Horde == Admin {
				// new ADMIN horde
				deanProg := fmt.Sprintf("pick(%s, 1, ALL)\n", ProctorRev)
				dret, err1 := deanClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: deanProg})
				logboot.Log(nil, "deanspec2 dret=%v err=%v", dret, err1)
				rl.SyncRS(sampleNOD, PicketRev, reeveapi, []string{PicketRev})

				// now that proctor is up, start up the bloc-wide services
				procProg := fmt.Sprintf("pick(%s, 1, ALL)\n", HealthCheckRev) +
					//	fmt.Sprintf("pick(%s, 1, ALL)\n", ProctorRev) +	// this would be steward
					fmt.Sprintf("pick(%s, 1, ALL)\n", GenghisRev) +
					//	fmt.Sprintf("pick(%s, 1, ALL)\n", CARev) +
					fmt.Sprintf("pick(%s, 1, ALL)\n", YurtRev)
				proctorClient := findProctor(logboot, hbnid, reeveapi)
				pret, err1 := proctorClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: procProg})
				logboot.Log(nil, "proctor returned %v err=%v", pret, err1)
				rl.SyncRS(sampleNOD, PicketRev, reeveapi, []string{PicketRev})

				// at this point, we have a single node running all the bloc-wide services
				// and regular node-wide services. so set this node up as an Admin horde.
				createAdmin(logboot, reeveapi, node.Node, conf)
				createHordes(logboot, reeveapi, hordespec)
			} else {
				// new regular horde. start regular services
				deanClient, hbnid := startNodeServices(logboot, net, sampleNOD, reeveapi)
				// if i am leader of my horde, start proctor
				genghis := findGenghis(logboot, hbnid, reeveapi)
				crux.FatalIfErr(logboot, err)
				myhorde, err := genghis.ClientHorde(context.Background(), &pb.ClientHordeReq{Horde: node.Horde})
				crux.FatalIfErr(logboot, crux.ErrE(err))
				if node.Node == myhorde.Nodes[0] {
					deanProg := fmt.Sprintf("pick(%s, 1, ALL)\n", ProctorRev)
					dret, err1 := deanClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: deanProg})
					logboot.Log(nil, "deanspec is all!!! dret=%v err=%v", dret, err1)
					procProg := fmt.Sprintf("pick(%s, 1, ALL)\n", HealthCheckRev)
					proctorClient := findProctor(logboot, hbnid, reeveapi)
					pret, err1 := proctorClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: procProg})
					logboot.Log(nil, "proctor returned %v err=%v", pret, err1)
					rl.SyncRS(sampleNOD, PicketRev, reeveapi, []string{PicketRev})
				}
			}
		}
		// hang out for a bit
		time.Sleep(1 * time.Second)
	}
}

// start up standard node services: "Dean", "Reeve", "Heartbeat", "Pastiche", "Picket"
func startNodeServices(logboot clog.Logger, net *flock.Flock, nod idutils.NodeIDT, reeveapi *reeve.StateT) (pb.DeanClient, idutils.NetIDT) {
	var c crux.Confab = net
	cc := &c

	// here is where we should start reeve

	// now that reeve is up, finish declaring the flocking gRPC server
	flockNOD := ReNOD(nod, FlockName, FlockAPI)
	logboot.Log(nil, "transform %s into %s", nod.String(), flockNOD.String())
	Flock1_0(net, nil, heartChan, nil, crux.SmallID(), logboot.With("focus", "flock"), flockNOD, FlockRev, reeveapi)

	// start up the picket service
	pickNOD := ReNOD(nod, PicketName, PicketAPI)
	pickNID := Picket1_0(nil, heartChan, &cc, crux.SmallID(), logboot.With("focus", PicketRev), pickNOD, PicketRev, reeveapi)
	psign := reeveapi.SelfSigner()
	_, err := ruck.ConnectPicket(pickNID, psign, logboot)
	crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "picket started!")

	// start up heartbeat. this is regretable, but dean needs a reliable way
	// to find out what is actually running. we could imagine sneaky ways to do this in a
	// half-hearted way for bootstrapping, but let's not.
	hbNOD := ReNOD(nod, HeartbeatName, HeartbeatAPI)
	hbNID := Heartbeat1_0(nil, heartChan, &cc, crux.SmallID(), logboot.With("focus", HeartbeatRev), hbNOD, HeartbeatRev, reeveapi)
	logboot.Log(nil, "started %s", HeartbeatRev)

	// start up dean
	deanNOD := ReNOD(nod, DeanName, DeanAPI)
	deanNID := Dean1_0(nil, heartChan, &cc, crux.SmallID(), logboot.With("focus", DeanRev), deanNOD, DeanRev, reeveapi)

	dsign := reeveapi.SelfSigner()
	deanClient, err := ruck.ConnectDean(deanNID, dsign, logboot)
	crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "dean started!\n")

	/*
		at this point, we have reeve, picket, and dean.
		we can now use dean but it relies on two tricks:
			1) if a symbol name
		is used BEFORE it appears in the muster maps, then it assumes the
		symbol comes from the current file. (this is awful, but not horrible.)
		we do dean in two parts; one before we start pastiche, and one after.
			2) if there is no evidence of heartbeat running, dean will start one
		hesitantly. there may be some weirdness here as this settles down.
	*/
	deanProg := fmt.Sprintf("pick(%s, 1, ALL)\n", DeanRev) +
		//fmt.Sprintf("pick(%s, 1, ALL)\n", "reeve1_0") + // ReeveRev is out of spec; TBD
		fmt.Sprintf("pick(%s, 1, ALL)\n", HeartbeatRev) +
		fmt.Sprintf("pick(%s, 1, ALL)\n", pastiche.PasticheRev) +
		fmt.Sprintf("pick(%s, 1, ALL)\n", PicketRev)
	dret, err1 := deanClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: deanProg})
	logboot.Log(nil, "deanspec is all!!! dret=%v err=%v", dret, err1)
	return deanClient, hbNID
}

func createAdmin(l clog.Logger, reeveapi *reeve.StateT, node string, conf *crux.Confab) {
	l.Log(nil, "create admin")
	(*conf).SetHorde(Admin)
}

func createHordes(l clog.Logger, reeveapi *reeve.StateT, hordespec string) {
	l.Log(nil, "create hordes(%s)", hordespec)
}

func findGenghis(l clog.Logger, hbNID idutils.NetIDT, reeveapi *reeve.StateT) pb.GenghisClient {
	nid, err := findService(l, GenghisName, hbNID, 30*time.Second, reeveapi)
	crux.FatalIfErr(l, err)
	sign := reeveapi.SelfSigner()
	gc, err := ruck.ConnectGenghis(nid, sign, l)
	crux.FatalIfErr(l, err)
	return gc
}

func findProctor(l clog.Logger, hbNID idutils.NetIDT, reeveapi *reeve.StateT) pb.ProctorClient {
	nid, err := findService(l, ProctorName, hbNID, 30*time.Second, reeveapi)
	crux.FatalIfErr(l, err)
	sign := reeveapi.SelfSigner()
	pc, err := ruck.ConnectProctor(nid, sign, l)
	crux.FatalIfErr(l, err)
	return pc
}

// BootstrapXX is the default registration process.
func BootstrapXX(port int, skey, ipname, ip, horde, beacon, networks, certdir string, visitor bool) {
	flockName := "flock"
	logboot := clog.Log.With("focus", "flock_boot", "node", ipname)

	logboot.Log("node", ipname, "in bootstrap")
	defer logboot.Log("node", ipname, "done bootstrap")

	// Step 1: start the Flock service, returns a *flock.Flock
	net := newFlock(port, skey, ipname, ip, beacon, networks, certdir, visitor)
	if ipname == "" {
		ipname = net.GetNames().Node
	}
	if ip == "" {
		ip = ipname
	}
	var c crux.Confab = net

	// Step 2: start reeve and steward
	reeveapi, err := reeve0_1(port, skey, ipname, ip, beacon, horde, networks, net)
	crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "reeveapi=%p", reeveapi)

	// Step 3: start heartbeating the bootstrap
	mydone := make(chan bool, 2)
	myUUID := crux.SmallID()
	(net.Monitor()) <- crux.MonInfo{
		Op:      crux.HeartBeatOp,
		Moniker: myUUID,
		T:       time.Now().UTC(),
		Oflock:  fmt.Sprintf("%s %s %s", ipname, "builtin", "Bootstrap"),
	}

	// Step3b: finish declare the flocking gRPC server
	flockNOD, err := idutils.NewNodeID(flockName, horde, ipname, FlockName, FlockAPI)
	crux.FatalIfErr(logboot, err)
	Flock1_0(net, nil, heartChan, nil, crux.SmallID(), logboot.With("focus", "flock"), flockNOD, FlockRev, reeveapi)

	// Step 4: start up the picket service
	cc := &c
	pickNOD, err := idutils.NewNodeID(flockName, horde, ipname, PicketName, PicketAPI)
	crux.FatalIfErr(logboot, err)

	pickNID := Picket1_0(nil, heartChan, &cc, crux.SmallID(), logboot.With("focus", PicketRev), pickNOD, PicketRev, reeveapi)
	psign, err := reeveapi.ClientSigner(PicketRev) // Signer for the Client side (does the signing on client calls out)
	crux.FatalIfErr(logboot, err)

	_, err = rl.DeclareClient(pickNOD, PicketRev, reeveapi)
	crux.FatalIfErr(logboot, err)

	pickClient, err := ruck.ConnectPicket(pickNID, psign, logboot)
	crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "picket started!")

	// Step 5: start up heartbeat. this is regretable, but dean needs a reliable way
	// to find out what is actually running. we could imagine sneaky ways to do this in a
	// half-hearted way for bootstrapping,
	hbNOD, err := idutils.NewNodeID(flockName, horde, ipname, HeartbeatName, HeartbeatAPI)
	crux.FatalIfErr(logboot, err)
	hbNID := Heartbeat1_0(nil, heartChan, &cc, crux.SmallID(), logboot.With("focus", HeartbeatRev), hbNOD, HeartbeatRev, reeveapi)
	logboot.Log(nil, "started %s", HeartbeatRev)

	// Step 6: start up dean
	deanNOD, err := idutils.NewNodeID(flockName, horde, ipname, DeanName, DeanAPI)
	crux.FatalIfErr(logboot, err)
	deanNID := Dean1_0(nil, heartChan, &cc, crux.SmallID(), logboot.With("focus", DeanRev), deanNOD, DeanRev, reeveapi)

	dsign := reeveapi.SelfSigner()

	// sync up, we're done with setup.
	rl.SyncRS(pickNOD, PicketRev, reeveapi, []string{PicketRev})

	deanClient, err := ruck.ConnectDean(deanNID, dsign, logboot)
	crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "dean started!\n")

	/*
		at this point, we have reeve/steward, picket, and dean.
		we can now use dean but it relies on two tricks:
			1) if a symbol name
		is used BEFORE it appears in the muster maps, then it assumes the
		symbol comes from the current file. (this is awful, but not horrible.)
		we do dean in two parts; one before we start pastiche, and one after.
			2) if there is no evidence of heartbeat running, dean will start one
		hesitantly. there may be some weirdness here as this settles down.
	*/
	deanProg := fmt.Sprintf("pick(%s, 1, ALL)\n", DeanRev) +
		//fmt.Sprintf("pick(%s, 1, ALL)\n", "reeve1_0") + // ReeveRev is out of spec; TBD
		fmt.Sprintf("pick(%s, 1, ALL)\n", HeartbeatRev) +
		fmt.Sprintf("pick(%s, 1, ALL)\n", pastiche.PasticheRev) +
		fmt.Sprintf("pick(%s, 1, ALL)\n", MetricRev) +
		fmt.Sprintf("pick(%s, 1, ALL)\n", PicketRev)
	dret, err1 := deanClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: deanProg})
	logboot.Log(nil, "deanspec is all!!! dret=%v err=%v", dret, err1)

	time.Sleep(10 * time.Second)
	logboot.Log(nil, "done sleeping")

	// we need our local pastiche
	// we should be able to do a local endpoints, but until then
	pasticheNID, err := findService(logboot, pastiche.PasticheName, hbNID, 10*time.Second, reeveapi)
	crux.FatalIfErr(logboot, err)

	// now tell pastiche about preloaded files
	dirs, err := ReadMuster(logboot)
	logboot.Log(nil, "readmuster: dirs=%s err=%v", dirs, err)
	crux.FatalIfErr(logboot, err)
	mep, err := ruck.ConnectPasticheSrv(pasticheNID, dsign, logboot.With("focus", "pasticheclient"))
	crux.FatalIfErr(logboot, err)
	logboot.Log(nil, "pastiche client connected")
	drep, err1 := mep.AddFilesFromDir(context.Background(), &pb.AddFilesFromDirRequest{Dirpath: dirs}) //FIXME:  no match for KeyID
	crux.FatalIfErr(logboot, crux.ErrE(err1))
	logboot.Log(nil, "pastiche addfiles returned: %+v", drep)

	// TODO:  add pastiche demo.  function taking client is preferrable.

	// finish loading dean
	// minor cheat until proctor works
	if ipname == "f5" {
		deanProg += fmt.Sprintf("pick(%s, 1, ALL)\n", HealthCheckRev)
		deanProg += fmt.Sprintf("pick(%s, 1, ALL)\n", GenghisRev) // this is a cheat for now
		deanProg += fmt.Sprintf("pick(%s, 1, ALL)\n", YurtRev)    // this is a cheat for now
		// here is where steward should go; see chris for details TBD
	}
	dret, err1 = deanClient.SetSpec(context.Background(), &pb.KhanSpec{Prog: deanProg})
	logboot.Log(nil, "deanspec is all2!!! dret=%v err=%v", dret, err1)

	// sync up, we're done.
	rl.SyncRS(pickNOD, PicketRev, reeveapi, []string{PicketRev})
	logboot.Log(nil, "deanspec is all3!!!")

	// norminally, we would start proctor up here in a way similiar to how we started dean.
	// proctor needs to add steward and healthcheck.
	// but not this day. TBD

	// at this point, this program can exit. the cluster is up and running.
	// instead, we'll do some random testing.

	// register us to use yurt
	yNID, err := findService(logboot, YurtName, hbNID, 10*time.Second, reeveapi)
	crux.FatalIfErr(logboot, err)
	yurtClient, err := ruck.ConnectYurt(yNID, dsign, logboot)
	crux.FatalIfErr(logboot, err)
	hr, errr := yurtClient.Who(context.Background(), &pb.Empty{})
	logboot.Log(nil, "yurt ret err=%v hr=%+v", errr, hr)

	// test Flock gRPC
	fNID, err := findService(logboot, FlockName, hbNID, 10*time.Second, reeveapi)
	crux.FatalIfErr(logboot, err)
	flockClient, err := ruck.ConnectFlock(fNID, dsign, logboot)
	crux.FatalIfErr(logboot, err)
	ns, errr := flockClient.Nodes(context.Background(), &pb.Empty{})
	logboot.Log(nil, "flock ret err=%v ns=%+v", errr, ns)

	_ = pickClient

	// kill time
	mydone <- true // turn off the bootstrap heartbeat
	time.Sleep(6 * time.Second)
	(net.Monitor()) <- crux.MonInfo{Op: crux.ExitOp, Moniker: "exit call", N: -1}
	time.Sleep(7 * time.Second)

	// we're done
	crux.Exit(0)
}
func findService(log clog.Logger, svc string, hbNID idutils.NetIDT, maxWait time.Duration, reeveapi *reeve.StateT) (idutils.NetIDT, *crux.Err) {
	log.Log(nil, "findService(%s) starts: hbNID=%s maxWait=%s", svc, hbNID.String(), maxWait.String())
	signer := reeveapi.SelfSigner()
	// connect to local heartbeat endpoint
	hbeat, err := ruck.ConnectHeartbeat(hbNID, signer, log)
	crux.FatalIfErr(log, err)

	// now loop
	ragnorak := time.Now().UTC().Add(maxWait)
	for time.Now().UTC().Before(ragnorak) {
		// get heartbeats
		hcrep, errr := hbeat.Heartbeats(context.Background(), &pb.Empty{})
		log.Log(nil, "heartbeats is %+v  errr=%v", hcrep, errr)
		if errr != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// rummage through for anything beginning with Pastiche
		for _, hr := range hcrep.List {
			if hr.NID == "" {
				continue
			}
			nid, err := idutils.NetIDParse(hr.NID)
			if err != nil {
				return idutils.NetIDT{}, err
			}
			log.Log(nil, "\tnid=%s svc=%s  ind=%d", nid.ServiceRev, svc, strings.Index(nid.ServiceRev, svc))
			if strings.Index(nid.ServiceRev, svc) == 0 {
				log.Log(nil, "findService(%s) found %s", svc, hr.NID)
				return nid, nil
			}
		}
		// pause to capture more heartbeats
		time.Sleep(500 * time.Millisecond)
	}
	log.Log(nil, "findService(%s) fails", svc)
	return idutils.NetIDT{}, crux.ErrF("couldn't find a %s", svc)
}

// BootstrapRipstop - is the follow-on tester for Burlap
// port :  the port for flocking UDP
// skey :  the flocking key (cmd argument --key)
// ipname :  is the hostname of the node we are on
// ip is : ip address of the host or resolvable hostname of the host we are on
// beacon : is an Address (ip:port) intended as the flock leader (cmd argument --beacon)
// horde : is the name of the horde on which this process's endpoints are running.
// networks: a list of CIDR networks to probe. If this is blank, we probe the local network of the given ip.
func BootstrapRipstop(port int, skey, ipname, ip, beacon, horde, networks, certdir string, visitor bool) {
	logboot := clog.Log.With("focus", "flock_boot", "node", ipname)
	logboot.Log("node", ipname, "in Ripstop bootstrap")
	defer logboot.Log("node", ipname, "done Ripstop bootstrap")
	// Start the Flock service, returns a *flock.Flock from
	// flock.NewFlockNode
	net := newFlock(port, skey, ipname, ip, beacon, networks, certdir, visitor)
	// This makes the *flock.Flock a crux.Confab interface
	// with methods to Get/Set stuff (Leader, Me, Register, Steward)
	if ipname == "" {
		ipname = net.GetNames().Node
	}
	if ip == "" {
		ip = ipname
	}
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
	var reeveapi rl.ReeveAPI
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
	/*      Reeve Documentation Demo Calls: service "bar", client "foo"
	barerr := sample.BarServiceStart(reeveapi, "flock", horde, me)
	if barerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - BarServiceStart failed : %v", barerr))
		crux.Exit(1)
	}
	fooerr := sample.FooClientExercise(reeveapi, "flock", horde, me)
	if fooerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - FooClientExercise failed : %v", fooerr))
		crux.Exit(1)
	}
	*/
	//------------------------------------
	// A typical service start - pastiche
	// which is both a server and client
	// ...
	// Part 1: Make the nodeid and netid for the service.
	// Set the nodeid
	/* obsolete now.
	pasticnod, eerr := idutils.NewNodeID("flock", horde, me, pastiche.PasticheName, pastiche.PasticheAPI)
	if eerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - invalid nodeid params for pastiche: %v", eerr))
		crux.Exit(1)
	}
	// Set the netid
	pasticheport := 50051
	pasticnid, eerr := idutils.NewNetID(pastiche.PasticheRev, principal, me, pasticheport)
	if eerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - invalid netid params for pastiche: %v", eerr))
		crux.Exit(1)
	}
	// ...
	// Part 2: Get the local security interfaces{} from reeve
	// Imp for the Server side (handles the inbound grpcsig whitelist lookups)
	pasticimp := reeveapi.SecureService(pastiche.PasticheRev)
	if pasticimp == nil {
		logboot.Log("node", ipname, "fatal", "failed reeveapi.SecureService for pastiche")
		crux.Exit(1)
	}
	// Signer for the Client side (does the signing on client calls out)
	pasticsign, perr := reeveapi.ClientSigner(pastiche.PasticheRev)
	if perr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("failed reeveapi.ClientSigner for pastiche %v", perr))
		crux.Exit(1)
	}
	// ...
	// Part 3: Pastiche Startup specific stuff - create a blob store
	// Get a NewServer and tell it where we will put files
	// We will use .muck/blob_dirs for now:

	blobdir := muck.BlobDir()
	var dirs []string
	dirs = append(dirs, blobdir)
	store, yerr := pastiche.NewServer(dirs)
	if yerr != nil {
		fmt.Fprintf(os.Stderr, "failed to create blob store %v\n", yerr)
		crux.Exit(1)
	}
	// Pass the parse-validated struct forms of the nodeid, netid, and the two interfaces
	// to start the Pastiche service.
	oerr := store.EasyStart(pasticnod, pasticnid, pasticimp, pasticsign)
	if oerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - starting pastiche %v", oerr))
		crux.Exit(1)
	}
	pastichestartedts := time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	// ...
	// Part 4: Advertise on the flock that our pastiche is a client and server ready
	// to do stuff.
	logboot.Log("node", ipname, "info", fmt.Sprintf("Registering pastiche with local reeve %v", oerr))
	// Get the self-signer
	selfsign := reeveapi.SelfSigner()
	// Dial the local gRPC client
	reeveclient, oerr := reeve.OpenGrpcReeveClient(reevenid, selfsign, logboot)
	if oerr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - local reeve grpc client failed - %v", oerr))
		crux.Exit(1)
	}
	// Construct what we need to RegisterEndpoint
	pasticheep := pb.EndpointInfo{
		Tscreated: pastichestartedts,
		Tsmessage: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Status:    pb.ServiceState_UP,
		Nodeid:   pasticnod.String(),
		Netid:     pasticnid.String(),
		Filename:  pastiche.PasticheRev, // Your plugin file hash goes here I think
	}
	// Make the gRPC call to local reeve
	ackPE, herr := reeveclient.RegisterEndpoint(context.Background(), &pasticheep)
	if herr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - RegisterEndpoint failed for pastiche: %v", herr))
		crux.Exit(1)
	}
	logboot.Log("node", ipname, "info", fmt.Sprintf("pastiche endpoint is registered with reeve: %v", ackPE))
	// Register the pastiche client
	pastichekid, pastichekeyjson := reeveapi.PubKeysFromSigner(pasticsign)
	pastichecl := pb.ClientInfo{
		Nodeid: pasticnod.String(),
		Keyid:   pastichekid,
		Keyjson: pastichekeyjson,
		Status:  pb.KeyStatus_CURRENT,
	}
	// Make the gRPC call to local reeve to register itself
	ackPC, werr := reeveclien.RegisterClient(context.Background(), &pastichecl)
	if werr != nil {
		logboot.Log("node", ipname, "fatal", fmt.Sprintf("error - RegisterClient failed for pastiche: %v", werr))
		crux.Exit(1)
	}
	logboot.Log("node", ipname, "info", fmt.Sprintf("pastiche client is registered with reeve: %v", ackPC))
	time.Sleep(20 * time.Second) // give reeve/steward if leader -  a chance to pick up updates
	// See the Catalog
	catrequest := pb.CatalogRequest{
		Nodeid: pasticnod.String(),
		Keyid:   pastichekid}
	pastichecatalog, ruerr := reeveclient.Catalog(context.Background(), &catrequest)
	if ruerr != nil {
		logboot.Log("node", ipname, "ERROR", fmt.Sprintf("error - Catalog failed for pastiche: %v", werr))
	}
	if pastichecatalog != nil {
		logboot.Log("node", ipname, "INFO", fmt.Sprintf("Catalog result: %v", pastichecatalog))
	}
	// See what endpoints are up for pastiche
	eprequest := pb.EndpointRequest{
		Nodeid: pasticnod.String(),
		Keyid:   pastichekid,
		Limit:   0}
	pasticheendpointsup, ruerr := reeveclient.EndpointsUp(context.Background(), &eprequest)
	if ruerr != nil {
		logboot.Log("node", ipname, "ERROR", fmt.Sprintf("error - EndpointsUp failed for pastiche: %v", werr))
	}
	otherServers := []string{}
	if pasticheendpointsup != nil {
		logboot.Log("node", ipname, "INFO", fmt.Sprintf("pastiche EndpointsUp result: %v", pasticheendpointsup))
		// Fish out the addresses for each pastiche endpoint, append to a list of strings
		for _, pep := range pasticheendpointsup.List {
			pepnid, _ := idutils.NetIDParse(pep.Netid)
			otherServers = append(otherServers, pepnid.Address())
		}
	}
	if len(otherServers) > 0 {
		store.AddOtherServers(otherServers)
		logboot.Log("node", ipname, "INFO", fmt.Sprintf("pastiche endpoints added to store -  %v", otherServers))
	}
	// Try the Pastiche DemoLocal functions on local client
	logpclient := clog.Log.With("focus", "pastiche-dial", "node", ipname)
	localPGcli, zerr := pastiche.OpenGrpcPasticheClient(pasticnid.Address(), pasticsign, logpclient)
	if zerr != nil {
		logboot.Log("node", ipname, "WARN", fmt.Sprintf("%v", zerr))
	}
	localPclient := pastiche.NewClient(localPGcli)
	loadfiles := false
	pherr := localPclient.AddDirToCache(blobdir, loadfiles)
	if pherr != nil {
		logboot.Log("node", ipname, "WARN", fmt.Sprintf("Couldn't add storage dir %s to server : %v", blobdir, pherr))
	}
	fakeHash := "fake-crypto-hash-1"
	bulkdata := "once upon a time there was a streaming protocol"
	dataRdr := bytes.NewBufferString(bulkdata)
	pastiche.DemoLocal(localPGcli, fakeHash, dataRdr)
	// Dial Remote Pastiche, run DemoRemote
	serverindex := 0
	if len(otherServers) > 3 {
		serverindex = 2
	}
	remotePGcli, zaerr := pastiche.OpenGrpcPasticheClient(otherServers[serverindex], pasticsign, logpclient)
	if zerr != nil {
		logboot.Log("node", ipname, "WARN", fmt.Sprintf("%v", zaerr))
	}
	remotePclient := pastiche.NewClient(remotePGcli)
	pherr = remotePclient.AddDirToCache(blobdir, loadfiles)
	if pherr != nil {
		logboot.Log("node", ipname, "WARN", fmt.Sprintf("Couldn't add storage dir %s to server : %v", blobdir, pherr))
	}
	fakeHash2 := "fake-crypto-hash-2-remote"
	bulkdata = "This file is to be placed first on the remote server, before being AddDataFromRemote()'d to local server"
	dataRdr = bytes.NewBufferString(bulkdata)
	pastiche.DemoRemote(localPGcli, remotePGcli, fakeHash2, dataRdr)
	*/
	// close down
	reeve.CloseGrpcReeveClient()
}
