// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package register

import (
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/flock"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
)

// ClientT - constant info for registering a node
// and an interface for fulcum bootstrapping
type ClientT struct {
	reevenodeid  string
	reevenetid   string
	pinginterval time.Duration
	contimeout   time.Duration
	cbtimeout    time.Duration
	imp          *grpcsig.ImplementationT
}

// AddAReeve - satisfies the crux interface for fulcrum node register call
// It is provided with the variable parts required for the call & callback
func (r *ClientT) AddAReeve(registryaddress string,
	enckey string,
	reevepubkeyjson string) *c.Err {

	err := AddAReeve(registryaddress,
		enckey,
		reevepubkeyjson,
		r.reevenodeid,
		r.reevenetid,
		r.pinginterval,
		r.contimeout,
		r.cbtimeout,
		r.imp)

	return err

}

// NewClient - for connecting to Register a node with the reeve callback info
// stashed here. Note this client does not need a signer, so it is a one-off.
func NewClient(reevenodeid string,
	reevenetid string,
	pinginterval time.Duration,
	contimeout time.Duration,
	cbtimeout time.Duration,
	impinterface **grpcsig.ImplementationT) *ClientT {

	// Here we stash the node's constant part of what is required for callback
	rcli := ClientT{}
	rcli.reevenodeid = reevenodeid
	rcli.reevenetid = reevenetid
	rcli.pinginterval = pinginterval
	rcli.contimeout = contimeout
	rcli.cbtimeout = cbtimeout
	rcli.imp = *impinterface
	return &rcli
}

// PingSleep - blocks, pings with delay intervals, until we get a response from server
// or total time exceeds timeout
func PingSleep(client pb.RegistryClient, delay time.Duration, timeout time.Duration) *c.Err {
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
			return c.ErrF("PingSleep blocking exceeds timeout")
		}
	}
}

// prepCallBackEnc - prepares for the registration callback by encrypting the message
func prepCallBackEnc(key string, nodeid string, netid string, pubkey string) (*pb.CallBackEnc, *c.Err) {
	if len(key) == 0 || len(nodeid) == 0 || len(netid) == 0 || len(pubkey) == 0 {
		return nil, c.ErrF("error - in prepCallBackEnc - missing parameters")
	}
	enckey, err := flock.String2Key(key)
	if err != nil {
		return nil, c.ErrF("error parsing key :%v", err)
	}
	nodeidenc, ferr := flock.Encrypt([]byte(nodeid), &enckey)
	if ferr != nil {
		return nil, c.ErrF("cannot encrypt nodeid :%v", ferr)
	}
	netidenc, nerr := flock.Encrypt([]byte(netid), &enckey)
	if nerr != nil {
		return nil, c.ErrF("cannot encrypt netid :%v", nerr)
	}
	pubkeyenc, perr := flock.Encrypt([]byte(pubkey), &enckey)
	if perr != nil {
		return nil, c.ErrF("cannot encrypt pubkey :%v", perr)
	}
	cb := new(pb.CallBackEnc)
	cb.NodeidEnc = base64.StdEncoding.EncodeToString(nodeidenc)
	cb.NetidEnc = base64.StdEncoding.EncodeToString(netidenc)
	cb.PubkeyEnc = base64.StdEncoding.EncodeToString(pubkeyenc)
	return cb, nil
}

// getRegisteredTimeout - does the registration... imp is needed because we hit the database
func getRegisteredTimeout(client pb.RegistryClient, cb *pb.CallBackEnc, imp *grpcsig.ImplementationT, to time.Duration) *c.Err {
	start := time.Now()
	var total time.Duration
	try := 0
	for {
		try = try + 1
		msg := fmt.Sprintf("Registering... try %d", try)
		pidstr, ts := grpcsig.GetPidTS()
		imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg)
		err := getRegistered(client, cb, imp)
		if err != nil {
			msg = fmt.Sprintf("getRegisteredTimeout try %d failed in getRegistered : %v", try, err)
			pidstr, ts := grpcsig.GetPidTS()
			imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
			total = time.Now().Sub(start)
			if total > to {
				tomsg := fmt.Sprintf("getRegisteredTimeout - timeout after %d attempts; last error : %v ", try, err)
				pidstr, ts := grpcsig.GetPidTS()
				imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, tomsg)
				return c.ErrF("%s", tomsg)
			}
			time.Sleep(2 * time.Second) // Don't flood the Registry with requests
			continue
		}
		break
	}
	return nil
}

func getRegisteredError(msg string, logger clog.Logger) *c.Err {
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
	return c.ErrF("%s", msg)
}

func getRegistered(client pb.RegistryClient, cb *pb.CallBackEnc, imp *grpcsig.ImplementationT) *c.Err {
	// calls Registry, gets streaming reply
	stream, err := client.Register(context.Background(), cb)
	if err != nil {
		msg0 := fmt.Sprintf("getRegistered failed in client.Register : %v", err)
		return getRegisteredError(msg0, imp.Logger)
	}

	pkinserted := false
	regstatus := pb.RegisterInfo_BUSY
	regts := ""
	msg := ""
	pk := &grpcsig.PubKeyT{}
	pidstr, ts := grpcsig.GetPidTS()

	cleanup := false
	// Streaming reply ends on io.EOF
	for {
		info, ierr := stream.Recv()
		if ierr == io.EOF {
			break
		}
		if ierr != nil {
			msg = fmt.Sprintf("getRegistered failed - bad registry stream with client %v : %v", client, ierr)
			cleanup = true
			break
		}
		// Fields in info:
		//		fmt.Printf("info.GetNodeid() is:%v\n", info.GetNodeid())  // for server's steward
		//		fmt.Printf("info.GetNetid() is:%v\n", info.GetNetid())      // for server's steward
		//		fmt.Printf("info.GetPubkey() is:%v\n", info.GetPubkey())  // for client's reeve
		//		fmt.Printf("info.GetStatus() (BUSY|DONE):%v\n", info.GetStatus())
		//		fmt.Printf("info.GetError():%s\n", info.GetError())
		//		fmt.Printf("info.GetTs():%s\n", info.GetTs())
		// Parse verify steward's informaiton
		regts = info.GetTs()
		if info.GetError() != "" {
			msg = fmt.Sprintf("getRegistered failed on stream.Recv error : %s at %s", info.GetError(), info.GetTs())
			cleanup = true
			break
		}
		stewardNetID := info.GetNetid()
		stewardNodeID := info.GetNodeid()
		_, nerr := idutils.NetIDParse(stewardNetID)
		if nerr != nil {
			msg = fmt.Sprintf("getRegistered recieved bad netid from steward at %s/%s: %v", stewardNodeID, stewardNetID, nerr)
			cleanup = true
			break
		}
		_, ferr := idutils.NodeIDParse(stewardNodeID)
		if ferr != nil {
			msg = fmt.Sprintf("getRegistered recieved bad nodeid from steward at %s/%s: %v", stewardNodeID, stewardNetID, nerr)
			cleanup = true
			break
		}
		regstatus = info.GetStatus()
		// we should see the pubkey for register/steward to access our reeve in
		// the first stream message, and insert it into our whitelist,
		// so it can complete the reeve() callback with grpcsig protection
		if !pkinserted {
			var perr *c.Err
			pk, perr = grpcsig.PubKeyFromJSON([]byte(info.GetPubkey()))
			if perr != nil {
				msg = fmt.Sprintf("getRegistered failed in PubKeyFromJSON unmarshall : %v", perr)
				return getRegisteredError(msg, imp.Logger)
			}
			stewardKeyID := pk.KeyID
			// Parse Verify
			_, kerr := idutils.KeyIDParse(stewardKeyID)
			if kerr != nil {
				msg = fmt.Sprintf("getRegistered recieved bad keyid from steward at %s/%s: %v", stewardNodeID, stewardNetID, nerr)
				return getRegisteredError(msg, imp.Logger)
			}
			dberr := grpcsig.AddPubKeyToDB(pk)
			if dberr != nil {
				msg = fmt.Sprintf("getRegistered failed failed in grpcsig AddPubKeyToDB : %v", dberr)
				return getRegisteredError(msg, imp.Logger)
			}
			pkinserted = true
			// save steward FID, NID, KID in ./muck/register/
			_ = muck.StewardKeyID(stewardKeyID)
			_ = muck.StewardNodeID(stewardNodeID)
			_ = muck.StewardNetID(stewardNetID)
			// Log the insert
			pidstr, ts := grpcsig.GetPidTS()
			imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("getRegistered added registry/steward public key to whitelist: %s", pk.KeyID))
		}
	}
	if regstatus != pb.RegisterInfo_DONE {
		cleanup = true
	}

	if !cleanup {
		pidstr, ts = grpcsig.GetPidTS()
		imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("getRegistered completed"))
		return nil
	}

	extra := msg
	if pkinserted {
		if pk != nil {
			derr := grpcsig.RemovePubKeyFromDB(pk)
			if derr != nil {
				msgC := fmt.Sprintf("getRegistered - could not remove failed steward public key from whitelist DB : %v", derr)
				pidstr, ts = grpcsig.GetPidTS()
				imp.Logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msgC)
				extra = extra + "; could not clean up whitelist DB"
			}
		}
		_ = muck.StewardKeyID("")
		_ = muck.StewardNodeID("")
		_ = muck.StewardNetID("")
	}
	msg6 := fmt.Sprintf("registration is incomplete - stream closed at %s %s", regts, extra)
	return getRegisteredError(msg6, imp.Logger)
}
