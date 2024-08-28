// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package register

import (
	"fmt"
	"time"

	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
)

// AddAReeve - does the one shot registration
func AddAReeve(registryaddress string,
	enckey string,
	reevepubkeyjson string,
	reevenodeid string,
	reevenetid string,
	pinginterval time.Duration,
	contimeout time.Duration,
	cbtimeout time.Duration,
	imp *grpcsig.ImplementationT) *c.Err {

	// Dial the gRPC registry server; no grpcsig whitelisting
	_ = grpcsig.CheckCertificate(imp.Certificate, "AddAReeve "+registryaddress)
	dialOpts := []grpc.DialOption{grpcsig.ClientTLSOption(imp.Certificate, registryaddress)}
	dialOpts = grpcsig.PrometheusDialOpts(dialOpts...)
	//dialOpts = append(dialOpts, grpc.WithBlock(), grpc.WithTimeout(6*time.Second))
	conn, err := grpc.Dial(registryaddress, dialOpts...)
	if err != nil {
		// you will never see this error unless grpc.WithBlock(), grpc.WithTimeout(#)...
		// but we will catch any failed concurrent Dial while in PingSleep
		return c.ErrF("AddAReeve - could not connect to registry at %s : %v", registryaddress, err)
	}
	defer conn.Close()

	client := pb.NewRegistryClient(conn)

	// Blocks until Ping response from Registry (i.e. returns nil when Registry server is up,
	// and concurrent grpc.Dial() worked)
	pidstr, ts := grpcsig.GetPidTS()
	imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("AddAReeve connecting to Registry at %s", registryaddress))

	pidstr, ts = grpcsig.GetPidTS()
	imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("AddAReeve In PingSleep"))
	perr := PingSleep(client, pinginterval, contimeout)
	if perr != nil {
		msg := fmt.Sprintf("AddAReeve - could not connect to registry - PingSleep timed out: %v", err)
		pidstr, ts = grpcsig.GetPidTS()
		imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}

	// Proceed with registration - encrypt our reeve callback information
	cb, cerr := prepCallBackEnc(enckey, reevenodeid, reevenetid, reevepubkeyjson)
	if cerr != nil {
		pidstr, ts := grpcsig.GetPidTS()
		imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, fmt.Sprintf("%v", cerr))
		return cerr
	}

	// Try to register with reasonable cbtimeout (inner timeout on reeve callback is 10 sec)
	// so this can try 2-3 times before failing out when set to 30 seconds.
	// Note that we saw a PingTest grpc message already so this should not longer values

	pidstr, ts = grpcsig.GetPidTS()
	imp.Logger.Log("SEV", "INFO", "TS", ts, fmt.Sprintf("AddAReeve calling getRegisteredTimeout"))
	rerr := getRegisteredTimeout(client, cb, imp, cbtimeout)
	if rerr != nil {
		msg2 := fmt.Sprintf("AddAReeve failed to register - timeout : %v", rerr)
		pidstr, ts := grpcsig.GetPidTS()
		imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg2)
		return c.ErrF("%v", msg2)
	}
	pidstr, ts = grpcsig.GetPidTS()
	imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "AddAReeve --- REEVE IS REGISTERED ---")
	return nil
}
