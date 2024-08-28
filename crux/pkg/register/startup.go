// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package register

import (
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/reeve"
)

func registryInitError(msg string, logger clog.Logger) *c.Err {
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "FATAL", "PID", pidstr, "TS", ts, msg)
	return c.ErrF(msg)
}

// RegistryInit - starts the registry infrastructure in "ripstop", required for running the Registry/Steward services
// Pass in the address (string as spec'd in grpc.Dial) for the registry service and steward service
// running on this machine. Assumes reeve is already started.
func RegistryInit(registrynod idutils.NodeIDT, registryaddress string, stewardaddress string, enckey string,
	rtimeout time.Duration, impinterface **grpcsig.ImplementationT) *c.Err {

	stewardImp = *impinterface

	// log this startup function to RegistryRev
	logger := clog.Log.With("focus", "RegistryRev", "mode", "registry-init")

	pid, derr := muck.Principal()
	if derr != nil {
		msg2 := fmt.Sprintf("RegistryInit failed in muck GetPrincipal : %v", derr)
		return registryInitError(msg2, logger)
	}

	// Set the NetID for steward (retrieves from .muck if there)
	stewardnid, nerr := idutils.NewNetID(reeve.StewardRev, pid, idutils.SplitHost(stewardaddress), idutils.SplitPort(stewardaddress))
	if nerr != nil {
		msg3 := fmt.Sprintf("RegistryInit failed - invalid netid for steward : %v", nerr)
		return registryInitError(msg3, logger)
	}
	stewardNetID = stewardnid.String()

	// Set the NodeID for steward on same host as registry
	stewardnod, serr := idutils.NewNodeID(registrynod.BlocName, registrynod.HordeName, registrynod.NodeName, reeve.StewardName, reeve.StewardAPI)
	if serr != nil {
		msgS := fmt.Sprintf("RegistryInit failed - invalid flock for steward : %v", serr)
		return registryInitError(msgS, logger)
	}
	stewardNodeID = stewardnod.String()

	// local for Serving information log line:
	registrynid, rerr := idutils.NewNetID(RegistryRev, pid, idutils.SplitHost(registryaddress), idutils.SplitPort(registryaddress))
	if rerr != nil {
		msg4 := fmt.Sprintf("RegistryInit failed - invalid netid for registry : %v", rerr)
		return registryInitError(msg4, logger)
	}

	// PubKey json string for steward (retrieves from .muck if already there)
	// and Registry to talk to reeve()s
	pk, kerr := reeve.MakeServiceKeys(reeve.ReeveRev, "", false)
	if kerr != nil {
		msg5 := fmt.Sprintf("RegistryInit failed - in MakeServiceKeys for steward's reeve client : %v", kerr)
		return registryInitError(msg5, logger)
	}
	pkjson, jerr := grpcsig.PubKeyToJSON(pk)
	if jerr != nil {
		msg6 := fmt.Sprintf("RegistryInit failed - in PubKeyToJSON : %v", jerr)
		return registryInitError(msg6, logger)
	}

	// Set the reeve callback access pubkey (shared with  (and signer)
	qerr := SetStewardPubkey(pkjson)
	if qerr != nil {
		msg7 := fmt.Sprintf("RegistryInit failed - in SetStewardPubkey : %v", qerr)
		return registryInitError(msg7, logger)
	}

	// Set the current flock encrytion key
	SetFlockKey(enckey)

	// Set the reeve timeout
	reeveTimeout = rtimeout

	// Start gRPC server for Registry inbound with no grpcsig whitelisting
	_ = grpcsig.CheckCertificate(stewardImp.Certificate, "RegistryInit")
	s := grpcsig.NewTLSServer(stewardImp.Certificate)

	pb.RegisterRegistryServer(s, &server{})
	grpc_prometheus.Register(s)

	// extract port from registryaddress
	lis, lerr := net.Listen("tcp", registrynid.Port)
	if lerr != nil {
		msg8 := fmt.Sprintf("RegistryInit failed - in net.Listen : %v", lerr)
		return registryInitError(msg8, logger)
	}
	// log that registry has started
	msg9 := fmt.Sprintf("%s Serving %s", registrynod.String(), registrynid.String())
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg9)
	go s.Serve(lis)
	return nil
}
