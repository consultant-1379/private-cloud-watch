// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// reeve local user portion of api

package reeve

import (
	"fmt"
	"time"

	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
)

// Heartbeat -- TODO
func (s *server) Heartbeat(ctx context.Context, in *pb.HeartbeatReq) (*pb.HeartbeatReply, error) {

	/*
	   type HeartbeatReq struct {
	   }

	   type HeartbeatReply struct {
	   }
	*/

	return nil, grpc.Errorf(codes.Unimplemented, "TODO")
}

// acknowledgeNodeNetIDs - Process NodeID, NetID pair - due dilligence - return an Acknowledgment struct
func acknowledgeNodeNetIDs(nodeid string, netid string) (*idutils.NodeIDT, *idutils.NetIDT, *pb.Acknowledgement, error) {
	ack := pb.Acknowledgement{}
	ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	nod, ferr := idutils.NodeIDParse(nodeid)
	if ferr != nil {
		ack.Ack = pb.Ack_FAIL
		ack.Error = fmt.Sprintf("bad nodeid : %v", ferr)
		return nil, nil, &ack, fmt.Errorf(ack.Error)
	}
	nid, nerr := idutils.NetIDParse(netid)
	if nerr != nil {
		ack.Ack = pb.Ack_FAIL
		ack.Error = fmt.Sprintf("bad netid : %v", nerr)
		return nil, nil, &ack, fmt.Errorf(ack.Error)
	}
	ack.Localuuid = uuid.NewMD5(uuid.NIL, []byte(nid.String())).String()
	ack.Ack = pb.Ack_WORKING
	return &nod, &nid, &ack, nil
}

// acknowledgeNodeKeyKIDs - Process NodeID, KeyID pair
func acknowledgeNodeKeyIDs(nodeid string, keyid string) (*idutils.NodeIDT, *idutils.KeyIDT, *pb.Acknowledgement, error) {
	ack := pb.Acknowledgement{}
	ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	nod, ferr := idutils.NodeIDParse(nodeid)
	if ferr != nil {
		ack.Ack = pb.Ack_FAIL
		ack.Error = fmt.Sprintf("bad nodeid : %v", ferr)
		return nil, nil, &ack, fmt.Errorf(ack.Error)
	}
	kid, nerr := idutils.KeyIDParse(keyid)
	if nerr != nil {
		ack.Ack = pb.Ack_FAIL
		ack.Error = fmt.Sprintf("bad keyid : %v", nerr)
		return nil, nil, &ack, fmt.Errorf(ack.Error)
	}
	ack.Ack = pb.Ack_WORKING
	ack.Localuuid = uuid.NewMD5(uuid.NIL, []byte(kid.String())).String()
	return &nod, &kid, &ack, nil
}

// ensureLocal - ensure's that the signer's KeyID is on the list of local ones (including self key)
// (signature is already validated in the interceptor, this is used this to exclude non-locals from using Reeve)
func ensureLocal(ctx context.Context, ack pb.Acknowledgement, fname string) (pb.Acknowledgement, error) {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	claimkid, cerr := grpcsig.WhoSigned(ctx)
	if cerr != nil {
		msg0 := fmt.Sprintf("Reeve %s called from unparseable entity - : %v", fname, cerr)
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg0)
		ack.Error = msg0
		return ack, grpc.Errorf(codes.Unauthenticated, "Could not extract KeyID from context")
	}
	ckid, kerr := idutils.KeyIDParse(claimkid)
	if kerr != nil {
		msg1 := fmt.Sprintf("Reeve %s called from entity - could not parse KeyID - : %v", fname, kerr)
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg1)
		ack.Error = msg1
		return ack, grpc.Errorf(codes.Unauthenticated, "Could not extract KeyID from context")
	}
	if !LocalClientExists(&ckid) {
		msg2 := fmt.Sprintf("Reeve %s called from non-local client %s", fname, ckid.String())
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg2)
		ack.Error = msg2
		return ack, grpc.Errorf(codes.Unauthenticated, "Client is not local")
	}
	return ack, nil
}

// RegisterEndpoint -- Registers a new local endpoint; forwards to steward()
func (s *server) RegisterEndpoint(ctx context.Context, in *pb.EndpointInfo) (*pb.Acknowledgement, error) {
	// check parameters and return an appropriate Ack with
	// a timestamp and an MD5 generated localuuid - and if fails, the Ack is so marked.
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	nod, nid, ack, aerr := acknowledgeNodeNetIDs(in.Nodeid, in.Netid)
	if aerr != nil {
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("RegisterEndpoint - invalid  arguments : %v", aerr))
		return ack, grpc.Errorf(codes.InvalidArgument, "%v", aerr)
	}
	ack.Ack = pb.Ack_FAIL // Assume it goes badly
	// Serve to local clients only
	ack2, gerr := ensureLocal(ctx, *ack, "RegisterEndpoint")
	if gerr != nil {
		serr := EndpointFailed(*in, ack.Localuuid, "")
		if serr != nil {
			msg0 := fmt.Sprintf("RegisterEndpoint - EndpointFailed failed : %v", serr)
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg0)
		}
		return &ack2, gerr
	}
	// Save the endpoint in local pending map & checkpoint json file
	perr := EndpointPending(*in, ack.Localuuid)
	if perr != nil {
		msg1 := fmt.Sprintf("RegisterEndpoint - EndpointPending failed : %v", perr)
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
		return ack, grpc.Errorf(codes.Internal, "%v", msg1)
	}
	// Set up GRPC call to Steward
	epd := pb.EndpointData{}
	epd.Nodeid = nod.String()
	epd.Netid = nid.String()
	epd.Hash = in.Filename
	epd.Status = in.Status

	// Push this into the Reeve event loop to send to Steward
	ReeveEvents.IngestEndpoint(&epd)

	// Return the Ack to the caller
	ack.Ack = pb.Ack_DONE
	return ack, nil
}

// RegisterClient -- Registers a client - forwards public keys to steward()
func (s *server) RegisterClient(ctx context.Context, in *pb.ClientInfo) (*pb.Acknowledgement, error) {
	// check parameters and return an appropriate Ack with
	// a timestamp and an MD5 generated localuuid - and if fails, the Ack is so marked.
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	nod, kid, ack, aerr := acknowledgeNodeKeyIDs(in.Nodeid, in.Keyid)
	if aerr != nil {
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("RegisterEndpoint - invalid  arguments : %v", aerr))
		return ack, grpc.Errorf(codes.InvalidArgument, "%v", aerr)
	}
	ack.Ack = pb.Ack_FAIL // Assume it goes badly
	// Serve to local clients only
	ack2, gerr := ensureLocal(ctx, *ack, "RegisterClient")
	if gerr != nil {
		serr := ClientFailed(*in, ack.Localuuid, "")
		if serr != nil {
			msg0 := fmt.Sprintf("RegisterClient  - ClientFailed failed : %v", serr)
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg0)
		}
		return &ack2, gerr
	}

	// Save the client in local pending map & checkpoint json file
	perr := ClientPending(*in, ack.Localuuid)
	if perr != nil {
		msg1 := fmt.Sprintf("RegisterClient - ClientPending failed : %v", perr)
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
		return ack, grpc.Errorf(codes.Internal, "%v", msg1)
	}

	// Set up GRPC call to Steward
	cld := pb.ClientData{}
	cld.Nodeid = nod.String()
	cld.Keyid = kid.String()
	cld.Keyjson = in.Keyjson
	cld.Status = in.Status

	// Push this into the Reeve event loop to send to Steward
	ReeveEvents.IngestClient(&cld)

	// Return the Ack to the caller
	ack.Ack = pb.Ack_DONE
	return ack, nil
}

// EndpointsUp -- Query for available endpoints for a given local client
func (s *server) EndpointsUp(ctx context.Context, in *pb.EndpointRequest) (*pb.Endpoints, error) {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	result := pb.Endpoints{}
	nod, kid, ack, aerr := acknowledgeNodeKeyIDs(in.Nodeid, in.Keyid)
	if aerr != nil {
		msg := fmt.Sprintf("EndpointsUp - invalid  arguments : %v", aerr)
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		result.Error = msg
		return &result, grpc.Errorf(codes.InvalidArgument, "%v", aerr)
	}
	ack.Ack = pb.Ack_FAIL // Assume it goes badly
	// Serve to local clients only
	_, gerr := ensureLocal(ctx, *ack, "EndpointsUp")
	if gerr != nil {
		msgA := fmt.Sprintf("%v", gerr)
		result.Error = msgA
		return &result, gerr
	}
	// We want a list of all servicerev within horde according to rules
	// First see if we have any rules yet. If not, we have no endpoints either...
	if ReeveState.rules == nil {
		msg0 := fmt.Sprintf("EndpointsUp - no endpoint data from steward yet")
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg0)
		result.Error = msg0
		return &result, nil
	}
	toservices := []string{}
	for _, rule := range ReeveState.rules {
		if rule.From == nod.ServiceName && rule.Horde == nod.HordeName {
			toservices = append(toservices, rule.To)
		}
	}
	endpoints, err := grpcsig.EndpointScan(toservices, nod.HordeName, kid.ServiceRev, int(in.Limit))
	if err != nil {
		msg1 := fmt.Sprintf("EndpointsUp - failed in EndpointScan : %v", err)
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
		result.Error = msg1
		return &result, grpc.Errorf(codes.Internal, msg1)
	}
	for _, e := range endpoints {
		epitem := pb.EpInfo{
			Nodeid:   e.NodeID,
			Netid:    e.NetID,
			Priority: e.Priority,
			Rank:     e.Rank}
		result.List = append(result.List, &epitem)
	}
	// Send an error if the list is empty, so the caller can figure out that they may
	// need to update the rules table (in pgk/registrydb/allowed.go)
	if len(result.List) == 0 {
		msg2 := fmt.Sprintf("EndpointsUp - no results - service rules for %s may prohibit you from seeing any entries", nod.String())
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg2)
		result.Error = msg2
		return &result, nil // result has Error message as a warning, this is not a system error
	}
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("EndpointsUp sent to %s %s", nod.String(), kid.String()))
	return &result, nil
}

// canSee - checks a catalog item against a rule.
// If the rule allows, the caller can see it.
// e.g. with initial implementaiton of rules
// pastiche only sees pastiche in same horde
// steward client sees reeves in any horde - not anything else
// reeve sees steward in any horde - not other reeves or anything else
// add some logic to improve error messages:
//	matchH: true if it would have match except for the horde
//	matchF: true if horde and To match, but From doesn't
//	matchT: true if horde and From match, but To doesn't
func canSee(nod *idutils.NodeIDT, item *pb.CatalogInfo) (match, matchH, matchF, matchT bool) {
	itemnod, err := idutils.NodeIDParse(item.Nodeid)
	if err != nil {
		return
	}
	for _, rule := range ReeveState.rules {
		if nod.HordeName == rule.Horde && nod.ServiceName == rule.From && itemnod.ServiceName == rule.To {
			match = true
			return
		}
		if nod.HordeName != rule.Horde && nod.ServiceName == rule.From && itemnod.ServiceName == rule.To {
			matchH = true
		}
		if nod.HordeName == rule.Horde && itemnod.ServiceName == rule.To {
			matchF = true
		}
		if nod.HordeName == rule.Horde && nod.ServiceName == rule.From {
			matchT = true
		}
	}
	return
}

// Catalog -- call provided for a query of available endpoint service types + example + plugin
// caller must be found in local list of clients, or self. Catalog provided is screened
// by rules, so the catalog - seen by a client - is only what is is allowed to connect to.
func (s *server) Catalog(ctx context.Context, in *pb.CatalogRequest) (*pb.CatalogReply, error) {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	nod, kid, _, aerr := acknowledgeNodeKeyIDs(in.Nodeid, in.Keyid)
	if aerr != nil {
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("Catalog requested - invalid CatalogRequest arguments : %v", aerr))
		return nil, grpc.Errorf(codes.InvalidArgument, "%v", aerr)
	}
	// Who is requesting catalog? Catalog only serves local clients.
	// currently, this seems to error incorrectly. andrew TBD
	if false && !LocalClientExists(kid) {
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("Catalog non local client request from %s %s-  : %v", nod.String(), kid.String(), aerr))
		return nil, grpc.Errorf(codes.Unauthenticated, "Client is not local")
	}
	logger.Log(nil, "Catalog(%+v): nod=%s", *in, nod)

	/*      Least strict Catalog
	// This is the whole catalog, with local reeve and registered steward
	cat := pb.CatalogReply{}
	for _, c := range ReeveState.catalog {
		catitem := pb.CatalogInfo{
			Nodeid:  c.Nodeid,
			Netid:    c.Netid,
			Filename: c.Filename}
		cat.List = append(cat.List, &catitem)
	}
	*/

	// More strict catalog check nod by the rules.
	// The Catalog filtered by the rules that apply for the calling nod/kid
	var match, matchH, matchF, matchT, mH, mF, mT bool
	cat := pb.CatalogReply{}
	for _, c := range ReeveState.catalog {
		catitem := pb.CatalogInfo{
			Nodeid:   c.Nodeid,
			Netid:    c.Netid,
			Filename: c.Filename}
		match, matchH, matchF, matchT = canSee(nod, &catitem)
		fmt.Printf("cansee(%+v, %+v) returns %v,%v,%v,%v  >%s %s %s<\n", nod, catitem, match, matchH, matchF, matchT, nod.HordeName, nod.ServiceName, c.Nodeid)
		mH, mF, mT = mH || matchH, mF || matchF, mT || matchT
		if match {
			cat.List = append(cat.List, &catitem)
		}
	}

	/*
		// Strictest possible catalog - check kid and nod
		The nod is used to check the rules table.
		The nod is not checked against local list, so it can be spoofed by a local user
		to get a catalog with someone else's ServiceName.
		a nod of //hordename//servicename// would suffice.
		To make this exactly strict, nod must also be sent to LocalClientExists(kid, nod)
	*/

	// Send an error if the catalog is empty, so the caller can figure out that they
	// need to update the rules table (in pgk/registrydb/allowed.go)
	if len(cat.List) == 0 {
		msg := "Catalog empty: "
		if mT {
			msg = msg + fmt.Sprintf("rules match horde and source, but not target %s", nod.ServiceName)
		} else if mF {
			msg = msg + fmt.Sprintf("rules match horde and target, but not source %s", nod.ServiceName)
		} else if mH {
			msg = msg + fmt.Sprintf("rules match source and target, but not horde %s", nod.HordeName)
		} else {
			msg = msg + fmt.Sprintf("no matching rules for horde=%s source=%s target=%s", nod.HordeName, nod.ServiceName, nod.ServiceName)
		}
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		return nil, grpc.Errorf(codes.InvalidArgument, msg)
	}

	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("Catalog sent to %s %s", in.Nodeid, in.Keyid))
	return &cat, nil
}
