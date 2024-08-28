// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package register

import (
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/flock"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
)

// server - implements cruxgen.RegistryServer
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

// Register -- Implements the Register service function
func (s *server) Register(callbackenc *pb.CallBackEnc, stream pb.Registry_RegisterServer) error {
	pidstr, ts := grpcsig.GetPidTS()
	//	fmt.Printf("\nIn Register\n")

	// Decrypt the inbound info - (caller's reeve() details)
	nodeid, netid, pubkey, err := decryptCallBackEnc(flockkey, callbackenc)
	if err != nil {
		if stewardImp.Logger != nil {
			stewardImp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("unable to decrypt callback data: %v", err))
		}
		return grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("%s", err.Err))
	}

	if stewardImp.Logger != nil {
		stewardImp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("register request for nodeid:%s netid:%s", string(nodeid), string(netid)))
	}

	//	fmt.Printf("nodeid:%s\n",string(nodeid))
	//	fmt.Printf("netid:%s\n",string(netid))
	//	fmt.Printf("pubkey:%s\n",string(pubkey))

	// Send initial reply with Steward() info, public key, BUSY
	// receiving reeve sets up Steward() pubkey for the callback
	info := prepInfo(false, nil)
	stream.Send(info)

	// Do the reverse-http-signatures callback dance to the Registrant's reeve()

	nid, nerr := idutils.NetIDParse(string(netid))
	if nerr != nil {
		if stewardImp.Logger != nil {
			stewardImp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("cannot parse netid from nodeid:%s netid:%s : %v", string(nodeid), string(netid), nerr))
		}
		return grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("%s %s", nerr.Err, nerr.Stack))
	}

	pk, kerr := grpcsig.PubKeyFromJSON(pubkey)
	if kerr != nil {
		if stewardImp.Logger != nil {
			stewardImp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("cannot unmarshal provided pubkey from nodeid:%s netid:%s : %v", string(nodeid), string(netid), kerr))
		}
		return grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("%s", kerr.Err))
	}

	//	fmt.Printf("nid:%v\n",nid)
	//	fmt.Printf("pubkey:%v\n",pk)
	//	fmt.Printf("Calling Back Reeve: attempt 1\n")
	try := 0

	//  Set the inner reeve client callback to retry with 10 sec timeout
	start := time.Now()
	var total time.Duration
	for {
		try = try + 1
		cerr := reeve.ClientUpdate(nid.Address(), pk, stewardSigner, stewardImp)
		if cerr != nil {
			total = time.Now().Sub(start)
			if total > reeveTimeout {
				msg := fmt.Sprintf("registry update callback to reeve exceeded reeveTimeout, last error:%v", cerr)
				// Send a done + error down the stream with the error, caller can retry
				timerr := prepInfo(true, fmt.Errorf("%s", msg))
				stream.Send(timerr)
				// log the timeout here
				if stewardImp.Logger != nil {
					pidstr, ts = grpcsig.GetPidTS()
					stewardImp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("failed reeve callback to nodeid: %s netid:%s : %s", string(nodeid), string(netid), msg))
				}
				// return the error
				return grpc.Errorf(codes.Unauthenticated, msg)
			}
			// log the failed try
			if stewardImp.Logger != nil {
				pidstr, ts = grpcsig.GetPidTS()
				stewardImp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("try %d - registry update callback failed to nodeid: %s netid:%s : %v", try, string(nodeid), string(netid), cerr))
			}
			// try again until we timeout
			continue
		}
		break
	}
	done := prepInfo(true, nil)
	stream.Send(done)

	if stewardImp.Logger != nil {
		stewardImp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("registry accepted nodeid:%s netid:%s", string(nodeid), string(netid)))
	}

	return nil
}

// prepInfo - Creates the return message
func prepInfo(done bool, err error) *pb.RegisterInfo {
	// make a RegisterInfo object
	info := new(pb.RegisterInfo)
	info.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.000000Z07:00")
	if done {
		info.Status = pb.RegisterInfo_DONE
	}
	if err != nil {
		info.Error = err.Error()
	}
	// set its values to current Steward contact info
	info.Netid = stewardNetID
	info.Nodeid = stewardNodeID
	info.Pubkey = stewardPubKeyJSON
	return info
}

// decryptCallBackEnc - decrypts the sender's information with flock key
func decryptCallBackEnc(key string, cb *pb.CallBackEnc) ([]byte, []byte, []byte, *c.Err) {
	empty := []byte{}
	if len(key) != 0 {
		enckey, err := flock.String2Key(key)
		if err != nil {
			return empty, empty, empty, c.ErrF("error parsing key :%v", err)
		}
		fid, serr := base64.StdEncoding.DecodeString(cb.NodeidEnc)
		if serr != nil {
			return empty, empty, empty, c.ErrF("base64 error decrypting nodeid: %v", serr)
		}
		nodeid, ferr := flock.Decrypt(fid, &enckey)
		if ferr != nil {
			return empty, empty, empty, c.ErrF("error decrypting nodeid: %v", ferr)
		}
		nid, terr := base64.StdEncoding.DecodeString(cb.NetidEnc)
		if terr != nil {
			return empty, empty, empty, c.ErrF("base64 error decrypting netid: %v", terr)
		}
		netid, nerr := flock.Decrypt(nid, &enckey)
		if nerr != nil {
			return empty, empty, empty, c.ErrF("error decrypting netid: %v", nerr)
		}
		pkey, uerr := base64.StdEncoding.DecodeString(cb.PubkeyEnc)
		if uerr != nil {
			return empty, empty, empty, c.ErrF("base64 error decrypting pubkey: %v", uerr)
		}
		pubkey, perr := flock.Decrypt(pkey, &enckey)
		if perr != nil {
			return empty, empty, empty, c.ErrF("error decrypting pubkey: %v", perr)
		}
		return nodeid, netid, pubkey, nil
	}
	return empty, empty, empty, c.ErrF("no encryption key provided")
}

// Internal copies of current server and flock credentials
var reeveTimeout time.Duration          // constrains callback to reeve
var stewardRev string                   // service name of steward
var stewardNetID string                 // netID of Steward()
var stewardNodeID string                // nodeID of Steward()
var stewardPubKey *grpcsig.PubKeyT      // PubKeyT of Steward()
var stewardPubKeyJSON string            // JSON version of above
var stewardSigner *grpcsig.AgentSigner  // ssh-agent signer
var stewardImp *grpcsig.ImplementationT // Steward's database and logger
var flockkey string                     // symmetric flocking key

// Helper functions for updating internal credentials
// when or if they are rotated

// GetStewardSigner - register holds the signer for steward, steward needs it.
func GetStewardSigner() *grpcsig.AgentSigner {
	return stewardSigner
}

// GetStewardImp - register holds the Imp for steward, steward needs it.
func GetStewardImp() *grpcsig.ImplementationT {
	return stewardImp
}

// SetFlockKey -  Set/Update the flocking key used inside register()
func SetFlockKey(key string) {
	flockkey = key
}

// SetStewardPubkey - Set the Steward PubKey used inside register()
// takes json marshalled steward PubKeyT as a string.
// Allows key rotation to update the settings after startup
func SetStewardPubkey(pubkey string) *c.Err {
	stewardpk, err := grpcsig.PubKeyFromJSON([]byte(pubkey))
	if err != nil {
		return err
	}

	// If rotating, remove old key from agent.
	if stewardSigner != nil {
		rerr := reeve.RemoveCurrentKeyFromAgent(stewardPubKey.KeyID, true)
		if rerr != nil {
			return c.ErrF("SetStewardPubkey - failed to remove old key: %v", rerr)
		}
	}

	// Hook up the new key to the agent
	nerr := reeve.AddCurrentKeyToAgent(stewardpk.KeyID, false)
	if nerr != nil {
		return c.ErrF("failed to add new key to agent: %v", nerr)
	}

	kid, kerr := idutils.KeyIDParse(stewardpk.KeyID)
	if kerr != nil {
		return c.ErrF("SetStewardPubkey - unable to parse keyid %s : %v", stewardpk.KeyID, kerr)
	}

	// Make a new signer
	newsigner, serr := grpcsig.ServiceSigner(kid)
	if serr != nil {
		return c.ErrF("SetStewardPubkey - steward ssh agent initialization failed: %v", serr)
	}

	// Set the internals
	stewardPubKeyJSON = pubkey
	stewardPubKey = stewardpk
	stewardSigner = newsigner
	stewardSigner.Certificate = stewardImp.Certificate

	return nil
}

// getFlockKey - Return the flocking key used inside register()
func getFlockKey() string {
	return flockkey
}

// GetStewardPubkeyJSON - Return the Steward Pubkey used inside register() in json format
func GetStewardPubkeyJSON() string {
	return stewardPubKeyJSON
}

// GetStewardKeyID - Return the Steward Pubkey used inside register()
func GetStewardKeyID() string {
	return stewardPubKey.KeyID
}
