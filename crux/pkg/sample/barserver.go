package sample

import (
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/rucklib"
)

type server struct {
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
// and a test grpc error code and message for any other value
func (s *server) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error")
}

// SimpleStart - starts server Bar
func SimpleStart(fid idutils.NodeIDT, nid idutils.NetIDT, impif **grpcsig.ImplementationT) *crux.Err {

	imp := *impif

	// Start gRPC server with Interceptors for http-signatures inbound
	s := imp.NewServer()

	// Use your Protobuf Register function for server bar
	pb.RegisterBarServer(s, &server{})
	grpc_prometheus.Register(s)

	// Listen on the specified port
	lis, err := net.Listen("tcp", nid.Port)
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("error - bar net.Listen failed: %v", err)
		return crux.ErrS(msg)
	}

	// Print and/or log the serving message with the full NodeID and NetID
	msg = fmt.Sprintf("%s Serving %s", fid.String(), nid.String())
	fmt.Printf("%s\n", msg)
	imp.Logger.Log("INFO", msg) // imp carries a Logger, so use it.
	go s.Serve(lis)
	return nil
}

// BarServiceStart - can be called from e.g. /ruck/bootstrap.go after pastiche
func BarServiceStart(reeveapi rucklib.ReeveAPI, blocName, hordeName, nodeName string) *crux.Err {
	barName := "bar"
	barAPI := "bar_1"
	barRev := "bar_1_0"
	barNodeID, _ := idutils.NewNodeID(blocName, hordeName, nodeName, barName, barAPI)
	principal, _ := muck.Principal()
	barNetID, _ := idutils.NewNetID(barRev, principal, nodeName, 51010)
	logbar := clog.Log.With("focus", "bar_service", "node", nodeName)
	barImp := reeveapi.SecureService(barRev)
	if barImp == nil {
		msg := "failed reeveapi.SecureService for bar"
		logbar.Log("node", nodeName, "error", msg)
		return crux.ErrS(msg)
	}
	err := SimpleStart(barNodeID, barNetID, barImp)
	if err != nil {
		msg := fmt.Sprintf("error - SimpleStart failed for bar: %v", err)
		logbar.Log("error", msg)
		return crux.ErrS(msg)
	}
	ts := time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	barEI := pb.EndpointInfo{
		Tscreated: ts,
		Tsmessage: ts,
		Status:    pb.ServiceState_UP,
		Nodeid:    barNodeID.String(),
		Netid:     barNetID.String()}
	selfsign := reeveapi.SelfSigner()
	_, reevenetid, _, _, _ := reeveapi.ReeveCallBackInfo()
	reeveNID, _ := idutils.NetIDParse(reevenetid)
	reeveclient, gerr := reeve.OpenGrpcReeveClient(reeveNID, selfsign, logbar)
	if gerr != nil {
		msg := fmt.Sprintf("error - OpenGrpcReeveClient failed for bar: %v", gerr)
		logbar.Log("error", msg)
		return crux.ErrS(msg)
	}
	ack, rerr := reeveclient.RegisterEndpoint(context.Background(), &barEI)
	if rerr != nil {
		msg := fmt.Sprintf("error - RegisterEndpoint failed for bar: %v", rerr)
		logbar.Log("error", msg)
		return crux.ErrS(msg)
	}
	logbar.Log("info", fmt.Sprintf("bar server is registered with reeve: %v", ack))
	return nil
}
