// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package reeve

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
)

var conn *grpc.ClientConn

// OpenGrpcReeveClient - dials reeve and returns a client. Typical use is to pass it
// the SelfSigner and call within the same process
func OpenGrpcReeveClient(reevenid idutils.NetIDT, signer **grpcsig.ClientSignerT, clilog clog.Logger) (pb.ReeveClient, *c.Err) {
	if signer == nil {
		return nil, c.ErrF("OpenGrpcReeveClient - no grpcsig.AgentSigner provided")
	}

	var pSigner *grpcsig.ClientSignerT
	pSigner = *signer
	if len(reevenid.Address()) == 0 {
		return nil, c.ErrF("OpenGrpcReeveClient - no address provided")
	}
	// Note the intentional use of WithBlock() and WithTimeout() here so we can quickly
	// expose any Dial failures.
	pidstr, ts := grpcsig.GetPidTS()
	clilog.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "OpenGrpcReeveClient - connecting to reeve")
	var err error
	conn, err = pSigner.Signer.Dial(reevenid.Address(),
		grpc.WithBlock(),
		grpc.WithTimeout(dialTimeout))
	if err != nil {
		msg := fmt.Sprintf("dial failed to connect to reeve server at %s, %v", reevenid.Address(), err)
		pidstr, ts := grpcsig.GetPidTS()
		clilog.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		return nil, c.ErrF("error - " + msg)
	}
	// Create a new ReeveClient
	client := pb.NewReeveClient(conn)
	// Log communication established
	pidstr, ts = grpcsig.GetPidTS()
	clilog.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("OpenGrpcReeveClient communication established with reeve at %s", reevenid.Address()))
	return client, nil
}

// CloseGrpcReeveClient - closes the connection to local reeve
func CloseGrpcReeveClient() {
	conn.Close()
}

// Implementation of the reverse-http-signatures callback
// to a reeve() server to get its public key updates.

// Timeout short timeout values for ClientUpdate.
var dialTimeout = 4 * time.Second
var pingsleepTimeout = 8 * time.Second
var pingsleepInterval = 1000 * time.Millisecond

// ClientUpdate - calls the reeve() service at address, signs with AgentSigner,
// and updates its local whitelist database (in imp.LookupResource).
// A one shot call to a reeve() service to get updates for its Public Keys
// Closes the connection if any errors - and is on a short timeout leash.
// Wrap calls to this with retry as needed.
// pk argument is the PubKeyT that the reeve() server is to use
// to sign its return value (reverse-http-signature)
func ClientUpdate(address string, pk *grpcsig.PubKeyT,
	signer *grpcsig.AgentSigner, imp *grpcsig.ImplementationT) *c.Err {

	if len(address) == 0 {
		return c.ErrF("error - no address provided")
	}

	if imp == nil {
		return c.ErrF("error - no grpcsig.ImplementationT provided")
	}

	if signer == nil {
		return c.ErrF("error - no grpcsig.AgentSigner provided")
	}

	if pk == nil {
		return c.ErrF("error - no public key provided for the server to sign with")
	}

	// Note the intentional use of WithBlock() and WithTimeout() here so we can quickly
	// expose any Dial failures.
	conn, err := signer.Dial(address,
		grpc.WithBlock(),
		grpc.WithTimeout(dialTimeout))
	defer conn.Close() // Any errors returned will shut down this connection
	if err != nil {
		msg := fmt.Sprintf("ClientUpdate grpc.Dial failed to connect to reeve server at %s, %v", address, err)
		if imp.Logger != nil {
			pidstr, ts := grpcsig.GetPidTS()
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Create a new ReeveClient
	client := pb.NewReeveClient(conn)

	// RvePingSleep until we get a fully qualified, non-error response (i.e. give that reeve() a chance
	// to update its grpcsig database if that was recently sent in the Register() mechanism)
	perr := RvePingSleep(client, pingsleepInterval, pingsleepTimeout) // Blocks until Ping response
	if perr != nil {
		msg := fmt.Sprintf("communication timeout to reeve server at %s: %v", address, perr)
		if imp.Logger != nil {
			pidstr, ts := grpcsig.GetPidTS()
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Log communication established
	if imp.Logger != nil {
		pidstr, ts := grpcsig.GetPidTS()
		imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("communication established with reeve at %s", address))
	}

	// Now do the real update work
	return getReeveUpdate(client, address, pk, imp)
}

// getStruUpdate - calls the RPC method  UpdatePubkeys of ReeveServer
// errors and logs.
func getReeveUpdate(client pb.ReeveClient, address string, pk *grpcsig.PubKeyT, imp *grpcsig.ImplementationT) *c.Err {
	pidstr, ts := grpcsig.GetPidTS()
	// argument pk must
	if len(pk.KeyID) == 0 {
		msg := fmt.Sprintf("malformed signwith public key, no KeyID")
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}
	signwith := &pb.SignWith{Keyid: pk.KeyID}

	// gRPC Call for the update package containing the reverse-http-signature
	resp, err := client.UpdatePubkeys(context.Background(), signwith)
	if err != nil {
		msg := fmt.Sprintf("could not get public key updates from reeve service at %s, %v", address, err)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Parse the response Signature
	sigparams, cerr := grpcsig.SignatureParse(resp.Signature)
	if cerr != nil {
		msg := fmt.Sprintf("reeve service at %s provided malformed signature: %v", address, cerr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}
	sigparams.Date = resp.Date

	// Is timestamp parse-able?
	hdrDate, terr := time.Parse(time.RFC1123, sigparams.Date)
	if terr != nil {
		msg := fmt.Sprintf("reeve service at %s provided malformed date string: %v", address, terr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}
	sigparams.Timestamp = hdrDate

	// Check the clock skew
	verr := grpcsig.VerifyClockskew(imp.ClockSkew, &sigparams)
	if verr != nil {
		msg := fmt.Sprintf("reeve service at %s exceeds clockskew: %v", address, verr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Check the crypto algorithm is supported
	verr = grpcsig.VerifyAlgorithm(imp, &sigparams)
	if verr != nil {
		msg := fmt.Sprintf("reeve service at %s suggesting unimplemented algorithm: %v", address, verr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Verify the KeyID received is the KeyID required as sent in signwith
	if sigparams.KeyID != signwith.Keyid {
		msg := fmt.Sprintf("reeve service at %s used wrong public key: %v", address, verr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Verify the service is the same service
	verr = grpcsig.VerifyService(imp, sigparams.KeyID)
	if verr != nil {

		// TODO REMOVE
		msg := fmt.Sprintf("reeve service at %s failed service verification: expected:imp[%v] saw:keyID[%s]  %v", address, imp, sigparams.KeyID, verr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Verify the signature with our in-memory version of the reeve's PubKeyT
	serr := grpcsig.VerifyCrypto(&sigparams, pk.PubKey)
	if serr != nil {
		msg := fmt.Sprintf("reeve service at %s failed signature verification: %v", address, serr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Log the authentication
	if imp.Logger != nil {
		imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("reeve at %s is authenticated", address))
	}

	// Update the Registration Database
	// TODO this all below changes with backend database implementation
	// for now we are just using the BoltDB whitelist system

	// Parse current keys
	currkeys := []grpcsig.PubKeyT{}
	jerr := json.Unmarshal([]byte(resp.Current), &currkeys)
	if jerr != nil {
		msg := fmt.Sprintf("reeve service at %s provided malformed json public keys: %v", address, jerr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Add the updated keys to whitelist db -
	dbname := imp.LookupResource
	dberr := grpcsig.AddPubKeyBlockUpdateToDB(currkeys)
	if dberr != nil {
		msg := fmt.Sprintf("reeve client unable to update pubkey database %s : %v", dbname, dberr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Parse deprecated keys
	deprkeys := []string{}
	jerr = json.Unmarshal([]byte(resp.Deprecated), &deprkeys)
	if jerr != nil {
		msg := fmt.Sprintf("reeve server %s provided malformed json deprecated keys: %v", dbname, jerr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Remove deprecated keys from our whitelist db
	dberr = grpcsig.RemovePubKeys(deprkeys)
	if dberr != nil {
		msg := fmt.Sprintf("reeve client unable to remove deprecated keys from db %s: %v", dbname, dberr)
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg)
		}
		return c.ErrF("error - " + msg)
	}

	// Databases updated - all good - log the update
	if imp.Logger != nil {
		pidstr, ts = grpcsig.GetPidTS() // reflect elapsed time
		imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("reeve at %s data updated", address))
	}

	return nil
}

// RvePingSleep - blocks, pings with delay intervals, until we get a response from server
// or total time exceeds timeout
func RvePingSleep(client pb.ReeveClient, delay time.Duration, timeout time.Duration) error {
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
			return fmt.Errorf("PingIt blocking exceeded timeout")
		}
	}
}
