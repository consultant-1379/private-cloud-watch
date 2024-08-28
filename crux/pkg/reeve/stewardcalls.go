// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// portion of the reeve server api that is restricted to updates from steward

package reeve

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
)

// ensureSteward -  ensures calls to this part of the API originate from the steward of record
func ensureSteward(ctx context.Context, ack pb.Acknowledgement, fname string) (pb.Acknowledgement, error) {
	// Find out who is calling. A insider may make a Steward client and send an fake update to this reeve.
	// Require the caller (and hence signer) matches exactly the registered steward server.
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	claimKeyID, cerr := grpcsig.WhoSigned(ctx)
	if cerr != nil {
		// unlikely to hit this as WhoSigned re-runs the same parser that was used in the interceptor
		msg0 := fmt.Sprintf("Reeve %s called from possible non-steward entity - : %v", fname, cerr)
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg0)
		ack.Error = msg0
		return ack, grpc.Errorf(codes.Unauthenticated, "Could not extract KeyID from context")
	}
	// Ensure this call arises only from the steward of record
	// whose KeyID was captured in .muck in the registration process.
	expect := muck.StewardKeyID("")
	if claimKeyID != expect {
		msg1 := fmt.Sprintf("Reeve %s call rejected - KeyID mismatch with known steward server", fname)
		logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("%s - got %s expected %s", msg1, claimKeyID, expect))
		ack.Error = msg1
		return ack, grpc.Errorf(codes.Unauthenticated, "%s", msg1)
	}
	return ack, nil
}

// UpdateCatalog -- call provided for steward to push updates to the in-memory catalog
// that is served up in the Catalog call to local clients
func (s *server) UpdateCatalog(ctx context.Context, in *pb.CatalogList) (*pb.Acknowledgement, error) {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	ack := pb.Acknowledgement{}
	ack.Ts = ts
	ack.Ack = pb.Ack_FAIL // Assume it goes badly
	ack2, gerr := ensureSteward(ctx, ack, "UpdateCatalog")
	if gerr != nil {
		return &ack2, gerr
	}
	newcat := []pb.CatalogInfo{}
	for _, cat := range in.List {
		catitem := pb.CatalogInfo{
			Nodeid:   cat.Nodeid,
			Netid:    cat.Netid,
			Filename: cat.Filename}
		nid, nerr := idutils.NetIDParse(catitem.Netid)
		if nerr != nil {
			msg2 := "UpdateCatalog rejected - Bad NetID in supplied catalog -"
			logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("%s %v", msg2, nerr))
			ack.Error = msg2
			return &ack, grpc.Errorf(codes.InvalidArgument, "%s", msg2)
		}
		// Filter out any reeve or steward listings in the catalog
		// If this node is not in the same horde steward - it is not listed
		// The preferred reeve is our own, so we swap it in
		if !(nid.ServiceRev == ReeveName || nid.ServiceRev == StewardName) {
			newcat = append(newcat, catitem)
		}
	}
	// Append on our Local Reeve to the catalog
	reeveitem := pb.CatalogInfo{
		Nodeid:   ReeveState.nodeid,
		Netid:    ReeveState.netid,
		Filename: ""}
	newcat = append(newcat, reeveitem)
	// Append on our official Steward as recorded during registration
	// also provides the steward info when our node not in same horde
	stewnod := muck.StewardNodeID("")
	stewnid := muck.StewardNetID("")
	stewarditem := pb.CatalogInfo{
		Nodeid:   stewnod,
		Netid:    stewnid,
		Filename: ""}
	newcat = append(newcat, stewarditem)
	ReeveState.catalog = newcat
	// Populate ReeveState.Rules   []RuleInfo
	ReeveState.rules = nil
	for _, rule := range in.Allowed {
		ruleitem := pb.RuleInfo{
			Rule:  rule.Rule,
			Horde: rule.Horde,
			From:  rule.From,
			To:    rule.To,
			Owner: rule.Owner}
		ReeveState.rules = append(ReeveState.rules, ruleitem)
	}
	pidstr, ts = grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "UpdateCatalog and rules update received from steward")
	ack.Ack = pb.Ack_DONE
	return &ack, nil
}

// WlState -- call provided for steward() to query reeve whitelist subscription state
func (s *server) WlState(ctx context.Context, in *pb.StateId) (*pb.Acknowledgement, error) {
	/*
	   type StateId struct {
	           State int32 `protobuf:"varint,1,opt,name=state" json:"state,omitempty"`
	   }

	*/
	// TODO - make a maxstate and a list of any gaps - that can be filled in

	return nil, grpc.Errorf(codes.Unimplemented, "WlState not implemented")

}

// EpState -- call provided for steward() to query reeve endpoint subscription state
func (s *server) EpState(ctx context.Context, in *pb.StateId) (*pb.Acknowledgement, error) {

	/*
	   type StateId struct {
	           State int32 `protobuf:"varint,1,opt,name=state" json:"state,omitempty"`
	   }
	*/
	// TODO - make this return a maxstate and a list of any gaps - that can be filled in

	return nil, grpc.Errorf(codes.Unimplemented, "EpState not implemented")
}

// WlUpdate -- call provided for steward to push whitelist pubkeys subscription update to reeve
func (s *server) WlUpdate(ctx context.Context, in *pb.WlList) (*pb.Acknowledgement, error) {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	ack := pb.Acknowledgement{}
	ack.Ts = ts
	ack.State = in.State
	ack.Ack = pb.Ack_FAIL // Assume it goes badly
	ack2, gerr := ensureSteward(ctx, ack, "WlUpdate")
	if gerr != nil {
		return &ack2, gerr
	}
	pubkeys := []grpcsig.PubKeyT{}
	for _, pubkeyjson := range in.Add {
		key, err := grpcsig.PubKeyFromJSON([]byte(pubkeyjson.Json))
		if err != nil {
			msg2 := fmt.Sprintf("WlUpdate - failed to parse pubkey : %v", err)
			ack.Ack = pb.Ack_FAIL
			ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
			ack.Error = msg2
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ack.Ts, msg2)
			return &ack, grpc.Errorf(codes.InvalidArgument, msg2)
		}
		key.StateAdded = in.State
		pubkeys = append(pubkeys, *key)
	}
	uerr := grpcsig.AddPubKeyBlockUpdateToDB(pubkeys)
	if uerr != nil {
		msg3 := fmt.Sprintf("WlUpdate failed in AddPubKeyBlockUpdateToDB : %v", uerr)
		ack.Ack = pb.Ack_FAIL
		ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ack.Ts, msg3)
		ack.Error = msg3
		return &ack, grpc.Errorf(codes.Internal, msg3)
	}

	// TODO DEL LIST

	msg4 := fmt.Sprintf("WlUpdate WHITELIST UPDATED with %d keys", len(pubkeys))
	ack.Ack = pb.Ack_DONE
	ack.State = in.State
	ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ack.Ts, msg4)

	// TODO save the last state updated in muck
	// Update the local client in pending map & checkpoint json file

	return &ack, nil
}

// EpUpdate -- call provided for steward to push endpoint subscription update to reeve
func (s *server) EpUpdate(ctx context.Context, in *pb.EpList) (*pb.Acknowledgement, error) {
	logger := ReeveState.imp.Logger
	pidstr, ts := grpcsig.GetPidTS()
	ack := pb.Acknowledgement{}
	ack.Ts = ts
	ack.State = in.State
	ack.Ack = pb.Ack_FAIL // Assume it goes badly
	ack2, gerr := ensureSteward(ctx, ack, "EpUpdate")
	if gerr != nil {
		return &ack2, gerr
	}
	endpoints := []grpcsig.EndPointT{}
	for _, ep := range in.Add { // EpInfo
		endpt := grpcsig.EndPointT{
			NodeID:     ep.Nodeid,
			NetID:      ep.Netid,
			Priority:   ep.Priority,
			Rank:       ep.Rank,
			StateAdded: in.State,
		}
		endpoints = append(endpoints, endpt)
	}
	uerr := grpcsig.AddEndPointBlockUpdateToDB(endpoints)
	if uerr != nil {
		msg3 := fmt.Sprintf("EpUpdate failed in AddEndPointBlockUpdateToDB : %v", uerr)
		ack.Ack = pb.Ack_FAIL
		ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ack.Ts, msg3)
		ack.Error = msg3
		return &ack, grpc.Errorf(codes.Internal, msg3)
	}

	// TODO DEL LIST

	msg4 := fmt.Sprintf("EpUpdate ENDPOINT UPDATED with %d endpoints = %+v", len(endpoints), endpoints)
	ack.Ack = pb.Ack_DONE
	ack.State = in.State
	ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ack.Ts, msg4)

	// TODO save the last state updated in muck
	// Update the local endpoint in pending map & checkpoint json file

	return &ack, nil

}
