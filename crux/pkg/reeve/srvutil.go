// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// Starts the local reeve() server.
// the reeve() server provides updates to public keys via a
// reverse-http-signatures mechanism. reeve() is grpcsig protected.

package reeve

import (
	"encoding/json"
	"fmt"
	//	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	//	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
)

// implement gRPC interface for ReeveServer
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

// UpdatePubkeys -- call provided for register/steward to pull public keys from reeve() with
// reverse-signatures
func (s *server) UpdatePubkeys(ctx context.Context, signwith *pb.SignWith) (*pb.PubKeysUpdate, error) {
	// query the current pubkeys matching service specified in fingerprint
	// create a pb.PubKeysUpdate struct
	pku := pb.PubKeysUpdate{}
	pku.Keyid = signwith.Keyid // echo the input

	aheadername := "authorization"
	dheadername := "date"

	// Here we use the Service Signer - we know the self key is ok
	// TODO for the deprecated keys, need to prime the ssh-add first!!!

	// fingerprint is a keyId here.
	// Look in /current or /deprecated

	kid, kerr := idutils.KeyIDParse(pku.Keyid)
	if kerr != nil {
		return nil, fmt.Errorf("UpdatePubkeys - unable to parse keyid %s : %v", pku.Keyid, kerr)
	}

	signer, err := grpcsig.ServiceSigner(kid)
	if err != nil {
		return nil, fmt.Errorf("UpdatePubkeys - unable to make service signer with %s in UpdatePubkeys: %v", pku.Keyid, err)
	}
	signer.Certificate = ReeveState.GetCertificate()

	// Re-use the http-signatures signer with a blank context, return a temp context
	tempctx, cerr := grpcsig.SignDateHeader(context.Background(), signer)
	if err != nil {
		return nil, fmt.Errorf("UpdatePubkeys - unable to sign/date header in UpdatePubkeys: %s, %v", pku.Keyid, cerr)
	}
	// fmt.Printf("\ntempctx:\n%v\n\n", tempctx)

	// extract the signature header from the temp context
	signature := metautils.ExtractOutgoing(tempctx).Get(aheadername)
	if signature == "" {
		return nil, fmt.Errorf("UpdatePubkeys - missing 'authorization' header in UpdatePubkeys")
	}

	// extract the data stamp from the temp context
	date := metautils.ExtractOutgoing(tempctx).Get(dheadername)
	if date == "" {
		return nil, fmt.Errorf("UpdatePubkeys - unauthenticated - missing required 'date' header in UpdatePubkeys")
	}

	// pku looks like this
	// fingerPrint:"/self/self/keys/f2:10:05:ad:a2:e6:2b:58:5d:97:70:26:41:08:37:10"
	// date:"Thu, 05 Apr 2018 17:39:49 UTC"
	// signature:"Signature keyId=\"/self/self/keys/f2:10:05:ad:a2:e6:2b:58:5d:97:70:26:41:08:37:10\",algorithm=\"ed25519\",headers=\"Date\",signature=\"sVdC6jdY0XeMpcFK+9vz/yDtU8QZh2eLmhbtFIuzj02QTfUtRb9rpmHLjx89Htu6n+RUnpUnTJNAiB1FbhheAQ==\""

	pku.Date = date
	pku.Signature = signature

	// Assemble the public key payload
	// Current keys
	curkeys, kerr := GetCurrentPubkeys(false)
	if kerr != nil {
		return nil, fmt.Errorf("%v", kerr)
	}
	jcurrkeys, jerr := json.Marshal(curkeys)
	if jerr != nil {
		return nil, fmt.Errorf("%v", jerr)
	}
	pku.Current = string(jcurrkeys)
	// Deprecated keys
	deprkeys, derr := GetDeprecatedPubkeys()
	if derr != nil {
		return nil, fmt.Errorf("%v", derr)
	}
	jdeprkeys, lerr := json.Marshal(deprkeys)
	if lerr != nil {
		return nil, fmt.Errorf("%v", lerr)
	}
	pku.Deprecated = string(jdeprkeys)

	// Return the public key payload
	return &pku, nil
}
