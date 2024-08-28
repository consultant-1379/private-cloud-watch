package grpcsvc

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/rucklib"
)

// GrpcServerStarter : what we need to start a grpc whitelist server
type GrpcServerStarter interface {
	RegisterServer(*grpc.Server)
	Name() string
}

// all the error returns herein need to be re-evaluated.  crux.Exit(1) for now

// StartGrpcServer : start and register the server contained in the GrpcServerStarter interface
func StartGrpcServer(fid *idutils.NodeIDT, serviceRev, address string, port int,
	xxx GrpcServerStarter, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	logger := clog.Log.With("node", fid.NodeName, "service", fid.ServiceName, "regserve", fid.ServiceAPI)
	// EasyStart - The grpc server will start listening for requests with grpc signatures security
	grpcStart := func(fid *idutils.NodeIDT, nid idutils.NetIDT, impif interface{}, xxx GrpcServerStarter) (string, *grpc.Server, *crux.Err) {
		// Sort out that interface for logging errors and stuff
		ptrimp, ok := impif.(**grpcsig.ImplementationT)
		logger.Log(nil, "%s(%s, %s): ok=%v\\n", xxx.Name(), fid.String(), nid.String(), ok)
		if !ok {
			return "", nil, crux.ErrF("bad interface{} passed to easyStart, not a **grpcsig.ImplementationT")
		}
		imp := *ptrimp

		// Start gRPC server with Interceptors for http-signatures inbound
		s := imp.NewServer()
		xxx.RegisterServer(s)
		grpc_prometheus.Register(s)
		lis, oerr := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if oerr != nil {
			msg11 := fmt.Sprintf("Register%sServer failed in net.Listen : %v", xxx.Name(), oerr)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg11)
			return "", nil, crux.ErrF("%s", msg11)
		}
		// Ready to serve
		go s.Serve(lis)
		// Pause for handy MacOS X firewall dialogue to appear & clicky-clicky
		if runtime.GOOS == "darwin" {
			time.Sleep(4 * time.Second)
		}
		quitfn := func(xxx GrpcServerStarter, server *grpc.Server, qchan <-chan bool) {
			logger.Log(nil, "%s waiting for quit chan", xxx.Name())
			<-qchan
			server.GracefulStop()
			lis.Close()
			// something to reeve?
			logger.Log(nil, "%s got quit", xxx.Name())
		}
		go quitfn(xxx, s, quit)
		logger.Log(nil, "%s %s Serving %s at %s", xxx.Name(), fid.String(), nid.String(), lis.Addr())

		return lis.Addr().String(), s, nil
	}

	principal, derr := muck.Principal()
	if derr != nil {
		logger.Log("fatal", fmt.Sprintf("error Principal: %v", derr))
		crux.Exit(1)
	}
	// Part 1: Make the netid for the service.
	nid, eerr := idutils.NewNetID(serviceRev, principal, address, port)
	if eerr != nil {
		logger.Log("fatal", fmt.Sprintf("invalid netid params: %v", eerr))
		crux.Exit(1)
	}
	// Part 2: Get the local security interfaces{} from reeve
	_, reevenetid, _, _, _ := reeveapi.ReeveCallBackInfo()
	reevenid, ierr := idutils.NetIDParse(reevenetid)
	if ierr != nil {
		logger.Log("fatal", "failed to parse reevenetid : %v", ierr)
		crux.Exit(1)
	}
	imp := reeveapi.SecureService(serviceRev)
	if imp == nil {
		logger.Log("fatal", "failed reeveapi.SecureService")
		crux.Exit(1)
	}

	// Pass the parse-validated struct forms of the nodeid, netid, and the two interfaces to start the service.
	addr, _, oerr := grpcStart(fid, nid, imp, xxx)
	if oerr != nil {
		logger.Log("fatal", fmt.Sprintf("failed to start service %s (nid=%s fid=%s): %v", xxx.Name(), nid.String(), fid.String(), oerr))
		crux.Exit(1)
	}
	startedts := time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")

	// craft port address by hand
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		logger.Fatal("address(%s) has bad format", addr)
		crux.Exit(1)
	}
	nid.Port = ":" + parts[len(parts)-1]

	// Part 4: Advertise on the flock that our server is ready to do stuff.
	logger.Log("info", fmt.Sprintf("Registering %s with local reeve", xxx.Name()))
	// Get the self-signer
	selfsign := reeveapi.SelfSigner()
	// Dial the local gRPC client
	reeveclient, oerr := reeve.OpenGrpcReeveClient(reevenid, selfsign, logger)
	if oerr != nil {
		logger.Log("fatal", fmt.Sprintf("local reeve grpc client failed: %v", oerr))
		crux.Exit(1)
	}

	// Construct what we need to RegisterEndpoint
	ep := pb.EndpointInfo{
		Tscreated: startedts,
		Tsmessage: time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"),
		Status:    pb.ServiceState_UP,
		Nodeid:    fid.String(),
		Netid:     nid.String(),
		Filename:  fid.ServiceAPI, // how on earth can this be correct?? TBD
	}
	// Make the gRPC call to local reeve
	ackPE, herr := reeveclient.RegisterEndpoint(context.Background(), &ep)
	if herr != nil {
		logger.Log("fatal", fmt.Sprintf("RegisterEndpoint failed: %v", herr))
		crux.Exit(1)
	}

	logger.Log(nil, "endpoint is registered with reeve: %v (addr=%s, nid=%s, ep=%+v)", ackPE, addr, nid.String(), ep)
	return nid
}

// NewGrpcClient : get a client connection to wrap in a service
func NewGrpcClient(dest idutils.NetIDT, signer interface{}, clilog clog.Logger, svcname string) (*grpc.ClientConn, *crux.Err) {
	if signer == nil {
		return nil, crux.ErrF("Connect%s - no grpcsig.AgentSigner provided", svcname)
	}
	pclisigner, ok := signer.(**grpcsig.ClientSignerT)
	if !ok {
		return nil, crux.ErrF("Connect%s - client signer interface sent is not a signer type", svcname)
	}
	var pSigner *grpcsig.ClientSignerT
	pSigner = *pclisigner
	if len(dest.Address()) == 0 {
		return nil, crux.ErrF("Connect%s - no address provided", svcname)
	}
	// Note the intentional use of WithBlock() and WithTimeout() here so we can quickly
	// expose any Dial failures.
	pidstr, ts := grpcsig.GetPidTS()
	clilog.Log("SEV", "INFO", "PID", pidstr, "TS", ts, nil, "GetPidTS(%s) returned: dial(%s) at %s", svcname, dest.Address(), crux.CallStack())
	conn, err := pSigner.Signer.Dial(dest.Address(),
		grpc.WithBlock(),
		grpc.WithTimeout(crux.DialTimeout),
	)
	if err != nil {
		msg := fmt.Sprintf("dial %s server at %s failed: %v", svcname, dest.Address(), err)
		pidstr, ts := grpcsig.GetPidTS()
		clilog.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		return nil, crux.ErrF("error - " + msg)
	}
	// Log communication established
	pidstr, ts = grpcsig.GetPidTS()
	clilog.Log("SEV", "INFO", "PID", pidstr, "TS", ts, nil, "Connect%s at %p thru %s", svcname, conn, dest.Address())
	// caller will (without fail) wrap the connection in a new client,
	// e.g., pb.NewDeanClient(conn)
	return conn, nil
}
