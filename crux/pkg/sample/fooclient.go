package sample

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/rucklib"
)

// FooClientExercise - exercises the Foo client which Pings all the Bar service nodes in the same horde.
func FooClientExercise(reeveapi rucklib.ReeveAPI, blocName, hordeName, nodeName string) *crux.Err {
	fooName := "foo"
	fooAPI := "foo1"
	barRev := "bar_1_0"

	fooNodeID, _ := idutils.NewNodeID(blocName, hordeName, nodeName, fooName, fooAPI)
	barSignerIf, _ := reeveapi.ClientSigner(barRev)
	fooKeyID, fooKeyJSON := reeveapi.PubKeysFromSigner(barSignerIf)
	fooCI := pb.ClientInfo{
		Nodeid:  fooNodeID.String(),
		Keyid:   fooKeyID,
		Keyjson: fooKeyJSON,
		Status:  pb.KeyStatus_CURRENT,
	}
	selfsign := reeveapi.SelfSigner()
	logfoo := clog.Log.With("focus", "foo_client", "node", nodeName)
	_, reevenetid, _, _, _ := reeveapi.ReeveCallBackInfo()
	reeveNID, _ := idutils.NetIDParse(reevenetid)
	reeveclient, rerr := reeve.OpenGrpcReeveClient(reeveNID, selfsign, logfoo)
	if rerr != nil {
		msg := fmt.Sprintf("error - OpenGrpcReeveClient failed for foo: %v", rerr)
		logfoo.Log("error", msg)
		return crux.ErrS(msg)
	}
	ack, cerr := reeveclient.RegisterClient(context.Background(), &fooCI)
	if cerr != nil {
		msg := fmt.Sprintf("error - RegisterClient failed for foo: %v", cerr)
		logfoo.Log("error", msg)
		return crux.ErrS(msg)
	}
	logfoo.Log("info", fmt.Sprintf("foo client is registered with reeve: %v", ack))

	// Delay until client's public keys are distributed...

	time.Sleep(35 * time.Second)

	// Get the Catalog for the foo client

	catrequest := pb.CatalogRequest{
		Nodeid: fooNodeID.String(),
		Keyid:  fooKeyID}

	fooCatalog, aerr := reeveclient.Catalog(context.Background(), &catrequest)
	if aerr != nil {
		msg := fmt.Sprintf("error - Catalog failed for foo: %v", aerr)
		logfoo.Log("node", nodeName, "ERROR", msg)
		return crux.ErrS(msg)
	}
	if fooCatalog != nil {
		logfoo.Log("node", nodeName, "INFO", fmt.Sprintf("Catalog result: %v", fooCatalog))
	}

	eprequest := pb.EndpointRequest{
		Nodeid: fooNodeID.String(),
		Keyid:  fooKeyID,
		Limit:  0,
	}

	// Get the Endpoints for the foo client

	fooEndpoints, uerr := reeveclient.EndpointsUp(context.Background(), &eprequest)
	if uerr != nil {
		msg := fmt.Sprintf("error - EndpointsUp failed for foo: %v", uerr)
		logfoo.Log("node", nodeName, "ERROR", msg)
		return crux.ErrS(msg)
	}

	pbarSigner := barSignerIf
	barClientSigner := **pbarSigner
	barAgentSigner := barClientSigner.Signer

	// gRPC Ping all the bar Endpoints with http-signatures enabled

	if fooEndpoints != nil {
		logfoo.Log("node", nodeName, "INFO", fmt.Sprintf("EndpointsUp result: %v", fooEndpoints))
		for _, fooEp := range fooEndpoints.List {
			fooEpNetID, _ := idutils.NetIDParse(fooEp.Netid)
			// Ping Each Bar server - returns on timeout,  error or success
			werr := WakeUpBar(barAgentSigner, fooEpNetID)
			if werr != nil {
				msg := fmt.Sprintf("error - foo WakeUpBar failed on endpoint %s: %v", fooEpNetID.String(), werr)
				logfoo.Log("node", nodeName, "ERROR", msg)
			}
			logfoo.Log("node", nodeName, "INFO", fmt.Sprintf("foo endpoint %s is awake :)", fooEpNetID.String()))
		}
	}
	return nil
}
