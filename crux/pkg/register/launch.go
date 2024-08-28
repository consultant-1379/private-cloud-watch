// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package register

import (
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/reeve"
)

func registryLaunchError(msg string, logger clog.Logger) *c.Err {
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
	return c.ErrF(msg)
}

// RegistryLaunch - starts the registry infrastructure in "organza", required for running the Registry/Steward services
// Pass in the address (string as spec'd in grpc.Dial) for the registry service and steward service
// running on this machine. Assumes reeve is already started.
func RegistryLaunch(registrynod idutils.NodeIDT, registryaddress string, stewardaddress string, enckey string,
	rtimeout time.Duration, impinterface **grpcsig.ImplementationT, stopch *chan bool) *c.Err {
	// log this startup function to RegistryLaunch
	logger := clog.Log.With("focus", "RegistryLaunch")

	stewardImp = *impinterface

	pid, derr := muck.Principal()
	if derr != nil {
		msg2 := fmt.Sprintf("RegistryLaunch() failed in muck GetPrincipal : %v", derr)
		return registryLaunchError(msg2, logger)
	}

	// Set the NetID for steward (retrieves from .muck if there)
	stewardnid, nerr := idutils.NewNetID(reeve.StewardRev, pid, idutils.SplitHost(stewardaddress), idutils.SplitPort(stewardaddress))
	if nerr != nil {
		msg3 := fmt.Sprintf("RegistryLaunch() failed - invalid netid for steward : %v", nerr)
		return registryLaunchError(msg3, logger)
	}
	stewardNetID = stewardnid.String()

	// Set the NodeID for steward on same host as registry
	stewardnod, serr := idutils.NewNodeID(registrynod.BlocName, registrynod.HordeName, registrynod.NodeName, reeve.StewardName, reeve.StewardAPI)
	if serr != nil {
		msgS := fmt.Sprintf("RegistryLaunch() failed - invalid flock for steward : %v", serr)
		return registryLaunchError(msgS, logger)
	}
	stewardNodeID = stewardnod.String()

	// local for Serving information log line:
	registrynid, rerr := idutils.NewNetID(RegistryRev, pid, idutils.SplitHost(registryaddress), idutils.SplitPort(registryaddress))
	if rerr != nil {
		msg4 := fmt.Sprintf("RegistryLaunch() failed - invalid netid for registry : %v", rerr)
		return registryLaunchError(msg4, logger)
	}

	// PubKey json string for steward (retrieves from .muck if already there)
	// and Registry to talk to reeve()s
	pk, kerr := reeve.MakeServiceKeys(reeve.ReeveRev, "", false)
	if kerr != nil {
		msg5 := fmt.Sprintf("RegistryLaunch() failed - in MakeServiceKeys for steward's reeve client : %v", kerr)
		return registryLaunchError(msg5, logger)
	}
	pkjson, jerr := grpcsig.PubKeyToJSON(pk)
	if jerr != nil {
		msg6 := fmt.Sprintf("RegistryLaunch() failed - in PubKeyToJSON : %v", jerr)
		return registryLaunchError(msg6, logger)
	}

	// Set the reeve callback access pubkey (shared with  (and signer)
	qerr := SetStewardPubkey(pkjson)
	if qerr != nil {
		msg7 := fmt.Sprintf("RegistryLaunch() failed - in SetStewardPubkey : %v", qerr)
		return registryLaunchError(msg7, logger)
	}

	// Set the current flock encrytion key
	SetFlockKey(enckey)

	// Set the reeve timeout
	reeveTimeout = rtimeout

	// Start gRPC server for Registry inbound with no grpcsig whitelisting
	_ = grpcsig.CheckCertificate(stewardImp.Certificate, "RegistryLaunch")
	s := grpcsig.NewTLSServer(stewardImp.Certificate)

	pb.RegisterRegistryServer(s, &server{})
	grpc_prometheus.Register(s)

	// extract port from registryaddress
	lis, lerr := net.Listen("tcp", registrynid.Port)
	if lerr != nil {
		msg8 := fmt.Sprintf("RegistryLaunch() failed - in net.Listen : %v", lerr)
		return registryLaunchError(msg8, logger)
	}
	// log that registry has started
	msg9 := fmt.Sprintf("%s Serving %s", registrynod.String(), registrynid.String())
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg9)
	go s.Serve(lis)
	stopfn := func(server *grpc.Server, nod idutils.NodeIDT, nid idutils.NetIDT, logger clog.Logger, stop *chan bool) {
		msg10 := fmt.Sprintf("%s GracefulStop Service  %s", nod.String(), nid.String())
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg10)
		<-*stop
		server.GracefulStop()
		lis.Close()
		msg11 := fmt.Sprintf("%s Service Stopped  %s", nod.String(), nid.String())
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg11)
	}
	go stopfn(s, registrynod, registrynid, logger, stopch)

	return nil
}
