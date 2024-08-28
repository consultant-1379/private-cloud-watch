package rucklib

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// DeclareClient register this client w.r.t. the endpoint
func DeclareClient(cNodeID idutils.NodeIDT, eRev string, reeveapi ReeveAPI) (string, *crux.Err) {
	// get the signer for the endpoint
	eSigner, e1 := reeveapi.ClientSigner(eRev)
	if e1 != nil {
		return "", e1
	}

	// get the endpoint infomation we need
	eKeyID, eKeyJSON := reeveapi.PubKeysFromSigner(eSigner)
	ci := pb.ClientInfo{
		Nodeid:  cNodeID.String(),
		Keyid:   eKeyID,
		Keyjson: eKeyJSON,
		Status:  pb.KeyStatus_CURRENT,
	}

	// generate a signer for local reeve without any keys
	myReeve := reeveapi.SelfSigner()

	// connect to local reeve
	mylog := clog.Log.With("focus", cNodeID.ServiceName, "node", cNodeID.NodeName)
	_, reeveNIDtext, _, _, _ := reeveapi.ReeveCallBackInfo()
	reeveNID, _ := idutils.NetIDParse(reeveNIDtext)
	rve, e1 := reeve.OpenGrpcReeveClient(reeveNID, myReeve, mylog)

	ack, cerr := rve.RegisterClient(context.Background(), &ci)
	if cerr != nil {
		msg := fmt.Sprintf("DeclareClient.RegisterClient failed for %s: %v", ci.Nodeid, cerr)
		mylog.Log("error", msg)
		return "", crux.ErrS(msg)
	}
	reeve.CloseGrpcReeveClient()
	mylog.Log("info", fmt.Sprintf("%s->%s client is registered with reeve: %v", cNodeID.ServiceName, eRev, ack))
	SyncRS(cNodeID, eRev, reeveapi, []string{eRev})
	return eKeyID, nil
}

// AllEndpoints returns an EpInfo for all endpoints i can see
func AllEndpoints(cNodeID idutils.NodeIDT, eRev string, reeveapi ReeveAPI) ([]*pb.EpInfo, *crux.Err) {
	mylog := clog.Log.With("focus", "allendpoints", "node", cNodeID.NodeName)
	mylog.Log(nil, "allendpoints(%+v, %s) from %s", cNodeID.String(), eRev, crux.CallStack())
	// straightforward: get a catalog of possible endpoints, and then generate endpoint info for all their instances
	eSigner, e1 := reeveapi.ClientSigner(eRev)
	if e1 != nil {
		return nil, e1
	}

	// get the endpoint infomation we need
	eKeyID, _ := reeveapi.PubKeysFromSigner(eSigner)
	mylog.Log(nil, "ekeyid=%s", eKeyID)

	// generate a signer for local reeve without any keys
	myReeve := reeveapi.SelfSigner()

	// connect to local reeve
	_, reeveNIDtext, _, _, _ := reeveapi.ReeveCallBackInfo()
	reeveNID, _ := idutils.NetIDParse(reeveNIDtext)
	mylog.Log(nil, "connect to reeve at %s", reeveNIDtext)
	rve, e1 := reeve.OpenGrpcReeveClient(reeveNID, myReeve, mylog)
	mylog.Log(nil, "rve=%+v, e1=%v", rve, e1)
	if e1 != nil {
		msg := fmt.Sprintf("opengrpcreevecient fail: %v", e1)
		mylog.Log("node", cNodeID.NodeName, "ERROR", msg)
		return nil, crux.ErrS(msg)
	}

	cr := pb.CatalogRequest{
		Nodeid: cNodeID.String(),
		Keyid:  eKeyID,
	}
	// wait for how long? TBD
	cat, aerr := rve.Catalog(context.Background(), &cr)
	if aerr != nil {
		msg := fmt.Sprintf("AllEndpoints.Catalog failed for %s: %v", cr.Nodeid, aerr)
		mylog.Log("node", cNodeID.NodeName, "ERROR", msg)
		return nil, crux.ErrS(msg)
	}
	mylog.Log(nil, "catalog returns %+v", cat)

	// reeve works on established links, so lets go make them now
	var keys, eps []string
	for _, ce := range cat.List {
		/*
			serv, err11 := idutils.NodeIDParse(ce.Nodeid)
			if err11 != nil {
				return nil, err11
			}
		*/
		nerd, err11 := idutils.NetIDParse(ce.Netid)
		if err11 != nil {
			return nil, err11
		}
		eps = append(eps, nerd.ServiceRev)
		k, err11 := DeclareClient(cNodeID, nerd.ServiceRev, reeveapi)
		if err11 != nil {
			return nil, err11
		}
		keys = append(keys, k)
		mylog.Log(nil, "add key element: %s -> %s", cNodeID.String(), nerd.ServiceRev)
	}

	// wait for them to  settle
	SyncRS(cNodeID, eRev, reeveapi, eps)

	// now we simply ask for endpoints for each of the above keys
	mylog.Log(nil, "doing keys %+v", keys)
	var ret []*pb.EpInfo
	ndone := make(map[string]bool, len(keys))
	for _, k := range keys {
		if ndone[k] == true {
			continue
		}
		ndone[k] = true
		er := pb.EndpointRequest{
			Nodeid: cNodeID.String(),
			Keyid:  k,
			Limit:  0,
		}
		ep, uerr := rve.EndpointsUp(context.Background(), &er)
		if uerr != nil {
			msg := fmt.Sprintf("AllEndpoints.EndpointsUp failed for %s: %v", er.Nodeid, uerr)
			mylog.Log("ERROR", msg)
			return nil, crux.ErrS(msg)
		}
		ret = append(ret, ep.List...)
		mylog.Log(nil, "added pair %s -> %s", cNodeID.String(), k)
	}

	/*
		we could range thru the list as in
		for _, xx := range ep.List {
			nid, _ := idutils.NetIDParse(xx.Netid)
			// ping nid
		}
	*/
	mylog.Log(nil, "allendpoints done: ret %+v", ret)
	return ret, nil
}

// Get1Endpoint returns the access stuff for a single endpoint
func Get1Endpoint(cNodeID idutils.NodeIDT, eRev string, reeveapi ReeveAPI) (idutils.NetIDT, **grpcsig.ClientSignerT, *crux.Err) {
	mylog := clog.Log.With("focus", cNodeID.ServiceName, "node", cNodeID.NodeName)
	mylog.Log(nil, "get1endpoint(%+v, %s): %s", cNodeID, eRev, crux.CallStack())
	var erk idutils.NetIDT
	// == copied from above
	// get the signer for the endpoint
	eSigner, e1 := reeveapi.ClientSigner(eRev)
	if e1 != nil {
		return erk, nil, e1
	}

	// get the endpoint infomation we need
	eKeyID, _ := reeveapi.PubKeysFromSigner(eSigner)

	// generate a signer for local reeve without any keys
	myReeve := reeveapi.SelfSigner()

	// connect to local reeve
	_, reeveNIDtext, _, _, _ := reeveapi.ReeveCallBackInfo()
	reeveNID, _ := idutils.NetIDParse(reeveNIDtext)
	rve, e1 := reeve.OpenGrpcReeveClient(reeveNID, myReeve, mylog)
	if e1 != nil {
		msg := fmt.Sprintf("opengrpcreevecient fail: %v", e1)
		mylog.Log(nil, "ERROR", msg)
		return erk, nil, crux.ErrS(msg)
	}
	// === end of copied from above
	_, err11 := DeclareClient(cNodeID, eRev, reeveapi)
	if err11 != nil {
		return erk, nil, err11
	}

	cr := pb.CatalogRequest{
		Nodeid: cNodeID.String(),
		Keyid:  eKeyID,
	}
	mylog.Log(nil, "cr=%+v", cr)
	// wait for how long? TBD
	cat, aerr := rve.Catalog(context.Background(), &cr)
	if aerr != nil {
		msg := fmt.Sprintf("Get1Endpoint.Catalog failed for %+v: %v", cr, aerr)
		mylog.Log("ERROR", msg)
		return erk, nil, crux.ErrS(msg)
	}
	nid, _ := idutils.NetIDParse(cat.List[0].Netid)
	fmt.Printf("gorp11: cat=%+v  nid=%+v\n", cat, nid)
	SyncRS(cNodeID, eRev, reeveapi, []string{eRev})
	return nid, eSigner, nil
}

// PingSleep - blocks, pings with delay intervals, until we get a response from server
// or total time exceeds timeout
func PingSleep(client pb.BarClient, delay time.Duration, timeout time.Duration) error {
	var total time.Duration
	// Make a ping
	ping := &pb.Ping{Value: pb.Pingu_PING}
	// Ping until we get the server response
	for {
		resp, cerr := client.PingTest(context.Background(), ping)
		if cerr != nil {
			total = total + delay
			time.Sleep(delay)
		} else {
			if resp.Value == pb.Pingu_PONG {
				return nil
			}
		}
		if total > timeout {
			return fmt.Errorf("PingSleep blocking exceeded timeout; last error: %v", cerr)
		}
	}
}

// WakeUpBar - Client call - Dials Bar endpoint, does PingSleep every 1s.
// Returns nil when PingSleep works with an
// authenticated connection to Bar,
// Returns an error if PingSleep times out (10s), with the last gRPC error seen.
func WakeUpBar(signer *grpcsig.AgentSigner, nid idutils.NetIDT) error {
	conn, err := signer.Dial(nid.Address())
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("WakeUpBar: grpc.Dial %s failed : %v", nid.String(), err)
	}
	barcli := pb.NewBarClient(conn)
	return PingSleep(barcli, 1*time.Second, 10*time.Second)
}

// SyncRS polls until the given client and all the endpoints have propogated to the client (us).
// if anything fails, just wait.
func SyncRS(cNodeID idutils.NodeIDT, eRev string, reeveapi ReeveAPI, epts []string) {
	mylog := clog.Log.With("focus", "syncrs", "node", cNodeID.NodeName)
	nowTime := time.Now().UTC()
	returnTime := time.Now().UTC().Add(10 * time.Second)
	time.Sleep(returnTime.Sub(nowTime))
	clog.Log.Log(nil, "SyncRS returns (after %s) from %s", returnTime.Sub(nowTime).String(), crux.CallStack())
	if false {

		pause := func(e *crux.Err) {
			if e != nil {
				clog.Log.Log(nil, "SyncRS error: %+v", e)
				time.Sleep(returnTime.Sub(time.Now().UTC()))
				clog.Log.Log(nil, "returning after SyncRS error: %+v", e)
			}
		}
		// get the signer for the endpoint
		eSigner, e1 := reeveapi.ClientSigner(eRev)
		if e1 != nil {
			pause(e1)
			return
		}

		// get the endpoint infomation we need
		eKeyID, _ := reeveapi.PubKeysFromSigner(eSigner)

		// generate a signer for local reeve without any keys
		myReeve := reeveapi.SelfSigner()
		_, reevenetid, _, _, _ := reeveapi.ReeveCallBackInfo()
		reeveNID, _ := idutils.NetIDParse(reevenetid)
		reeveclient, rerr := reeve.OpenGrpcReeveClient(reeveNID, myReeve, mylog)
		if rerr != nil {
			pause(crux.ErrF("error - OpenGrpcReeveClient failed for SyncRS: %v", rerr))
			return
		}

		// get the catalog
		catrequest := pb.CatalogRequest{
			Nodeid: cNodeID.String(),
			Keyid:  eKeyID,
		}

		mycat, aerr := reeveclient.Catalog(context.Background(), &catrequest)
		if aerr != nil {
			pause(crux.ErrF("Catalog failed: %v", aerr))
			return
		}
		if mycat != nil {
			mylog.Log(nil, "Catalog result: %v", mycat)
		}

		// get the endpoints
		eprequest := pb.EndpointRequest{
			Nodeid: cNodeID.String(),
			Keyid:  eKeyID,
			Limit:  0,
		}

		// Get the Endpoints
		epl, uerr := reeveclient.EndpointsUp(context.Background(), &eprequest)
		if uerr != nil {
			pause(crux.ErrF("EndpointsUp failed: %v (ep=%+v)", uerr, eprequest))
			return
		}

		barClientSigner := **eSigner
		barAgentSigner := barClientSigner.Signer

		// gRPC Ping all the bar Endpoints with http-signatures enabled
		if epl != nil {
			mylog.Log(nil, "EndpointsUp result: %+v", epl)
			for _, ep := range epl.List {
				netID, _ := idutils.NetIDParse(ep.Netid)
				// Ping Each Bar server - returns on timeout,  error or success
				werr := WakeUpBar(barAgentSigner, netID)
				if werr != nil {
					pause(crux.ErrF("WakeUpBar failed on endpoint %s: %v", netID.String(), werr))
					return
				}
				mylog.Log(nil, "SyncRS endpoint %s is pongs", netID.String())
			}
		}
		mylog.Log(nil, "SyncRS took %s", time.Now().UTC().Sub(nowTime).String())
	}
}

// DisplayEndpoints - for tests and debug
// TODO: show actual service names. - pastiche, Steward, etc.
func DisplayEndpoints(endPts *pb.Endpoints) {
	//endPts is a list of EpInfo's with nodeid and netid
	if endPts != nil {
		for _, endPt := range endPts.List {
			endPtID, _ := idutils.NetIDParse(endPt.Netid)
			// otherServers = append(otherServers, endPtID.Address())
			fmt.Printf("endpoint netID: %s \n", endPtID.String())
		}
	}
}
