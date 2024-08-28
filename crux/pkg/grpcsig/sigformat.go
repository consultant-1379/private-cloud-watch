// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"

	c "github.com/erixzone/crux/pkg/crux"
)

const (
	dateTag     = "date:" // As seen on server side as a header string
	dateName    = "Date"  // As seen on client side in header struct
	aheaderName = "Authorization"
)

// SignDateHeader - client side function that updates a context to add the http-signatures header
// Here, a single function consolidates all the header formatting.
// For gRPC implementation, only the Date header is used,
// (the http-signatures spec expands this to signing a list of headers)
func SignDateHeader(ctx context.Context, s *AgentSigner) (context.Context, *c.Err) {
	// Create the date header string as it will be seen on the recieving end
	dateValue := time.Now().UTC().Format(time.RFC1123)
	dateHeader := fmt.Sprintf("%s %s", dateTag, dateValue)
	// Sign the date header - via the AgentSigner
	sig, err := s.Sign(dateHeader)
	if err != nil {
		return ctx, c.ErrF("Cannot Sign header: %v\n", err)
	}
	// Produce the authHeader from the sig
	authHeader := fmt.Sprintf(`Signature keyId="%s",algorithm="%s",headers="%s",signature="%s"`,
		s.KeyID.String(),
		sig.HashAlg,
		dateName,
		sig.Base64Sig)
	// Package up the gRPC http headers
	md1 := metadata.Pairs(dateName, dateValue)
	md2 := metautils.NiceMD(md1).Add(aheaderName, authHeader)
	// Update the context
	fCtx := metautils.NiceMD(md2).ToOutgoing(ctx)
	return fCtx, nil
}

// SignatureFormatToAlg2 - converts agent reported format to signature format
func SignatureFormatToAlg2(format string) (string, *c.Err) {
	trunc := format
	if len(format) > 11 {
		trunc = format[:11]
	}
	alg := ""
	switch trunc {
	case "ssh-rsa":
		alg = "rsa"
	case "ssh-ed25519":
		alg = "ed25519"
	case "ecdsa-sha2-":
		alg = "ecdsa"
	}
	if alg == "" {
		return alg, c.ErrS(fmt.Sprintf("Key Format %s is unknown/not implemented.", format))
	}
	return alg, nil
}
