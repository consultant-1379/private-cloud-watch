// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	c "github.com/erixzone/crux/pkg/crux"
)

var (
	aheadername = "Authorization"
	dheadername = "date"
)

// SignatureParametersT  - verification data
type SignatureParametersT struct {
	KeyID      string
	Crypto     *CryptoT
	Headers    headerValues
	Date       string
	Timestamp  time.Time
	HeaderList []string
	Signature  string
}

// GetPidTS - utility function to get PID, Timestamp for logging calls
func GetPidTS() (string, string) {
	pidstr := fmt.Sprintf("%s", strconv.Itoa(os.Getpid()))
	ts := fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	return pidstr, ts
}

// WhoSigned returns the keyID from the authorization header.
func WhoSigned(ctx context.Context) (string, *c.Err) {
	// Do we have an HTTP signature?
	signature := metautils.ExtractIncoming(ctx).Get(aheadername)
	if signature == "" {
		return "", c.ErrF("unauthenticated - missing required http-signatures 'authentication' header")
	}
	// Is it parse-able?
	sigvals, cerr := SignatureParse(signature)
	if cerr != nil {
		return "", cerr
	}
	return sigvals.KeyID, nil
}

// VerifyAll - takes the context, processes it in stages, returns grpc formatted errors.
// If logging is implemented, logs failures as SEV=WARN, OK as SEV=INFO
func VerifyAll(ctx context.Context, imp *ImplementationT) (context.Context, error) {
	if !imp.Started {
		return nil, grpc.Errorf(codes.FailedPrecondition, "Error - server-side http-signatures authentication misconfiguration")
	}
	pidstr, ts := GetPidTS()
	sigparams, err := VerifyHeader(ctx, imp)
	if sigparams == nil || err != nil {
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("VerifyHeader failed: %v", err))
		}
		return nil, grpc.Errorf(codes.InvalidArgument, "Error - client-side http-signatures header: %v", err.Error())
	}
	err = VerifyClockskew(imp.ClockSkew, sigparams)
	if err != nil {
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("VerifyClockskew failed: %v", err))
		}
		return nil, grpc.Errorf(codes.DeadlineExceeded, "Error - : %v", err.Error())
	}
	err = VerifyAlgorithm(imp, sigparams)
	if err != nil {
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("VerifyAlgorithm failed: %v", err))
		}
		return nil, grpc.Errorf(codes.Unimplemented, "Error - : %v", err.Error())
	}
	err = VerifyService(imp, sigparams.KeyID)
	if err != nil {
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("VerifyService failed: %v", err))
		}
		return nil, grpc.Errorf(codes.NotFound, "Error - : %v", err.Error())
	}
	publickey := ""
	publickey, err = GetPublicKeyString(imp, sigparams.KeyID)
	if err != nil || len(publickey) == 0 {
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("GetPublicKeyString failed: %v", err))
		}
		return nil, grpc.Errorf(codes.NotFound, "Error - : %v", err.Error())
	}
	err = VerifyCrypto(sigparams, publickey)
	if err != nil {
		if imp.Logger != nil {
			imp.Logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, fmt.Sprintf("VerifyCrypto failed: %v", err))
		}
		return nil, grpc.Errorf(codes.Unauthenticated, "Error - : %v", err.Error())
	}
	if imp.Logger != nil {
		imp.Logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, fmt.Sprintf("grpcsig client authenticated: %s", sigparams.KeyID))
	}
	return ctx, nil
}

// VerifyHeader - takes inbound context, implementation, verifies the signature, clockskew, algorithm
// and parsing, returning parsed parameters and *c.Err types grouped by (grpc codes.InvalidArgument)
func VerifyHeader(ctx context.Context, imp *ImplementationT) (*SignatureParametersT, *c.Err) {
	// Do we have an HTTP signature?
	signature := metautils.ExtractIncoming(ctx).Get(aheadername)
	if signature == "" {
		return nil, c.ErrF("unauthenticated - missing required http-signatures 'authentication' header")
	}
	// Is it parse-able?
	sigvals, cerr := SignatureParse(signature)
	if cerr != nil {
		return nil, cerr
	}
	// Does timestamp exist?
	date := metautils.ExtractIncoming(ctx).Get(dheadername)
	if date == "" {
		return nil, c.ErrF("unauthenticated - missing required 'date' header for http-signatures")
	}
	// Is timestamp parse-able?
	hdrDate, err := time.Parse(time.RFC1123, date)
	if err != nil {
		return nil, c.ErrF("unauthenticated - cannot parse timestamp provided in http-signatures 'date' header, use RFC1123: '%s'", date)
	}
	sigvals.Date = date
	sigvals.Timestamp = hdrDate
	return &sigvals, nil
}

// VerifyClockskew - checks parsed time from client against "now", (grpc codes.DeadlineExceeded)
func VerifyClockskew(clockskew int, sigparams *SignatureParametersT) *c.Err {
	// Is timestamp within clockskew?
	dtime := (int)(time.Since(sigparams.Timestamp).Seconds())
	neg := "behind by "
	if dtime < 0 {
		dtime = -dtime
		neg = "ahead by "
	}
	if dtime > clockskew {
		return c.ErrF("unauthenticated - timestamp in http-signatures header exceeds server specified clock skew of +/- %d seconds, client is %s%d s", clockskew, neg, dtime)
	}
	return nil
}

// VerifyAlgorithm - checks parsed algorithm name against provided list of supported crypto algorithms.
// (grpc codes.Unimplemented)
func VerifyAlgorithm(imp *ImplementationT, sigparams *SignatureParametersT) *c.Err {
	// Is the specified algorithm supported?
	AlgorithmOk := false
	for _, algorithm := range imp.Algorithms {
		if sigparams.Crypto.Name == algorithm {
			AlgorithmOk = true
			break
		}
	}
	if !AlgorithmOk {
		return c.ErrF("unauthenticated - encryption algorithm '%s' in http-signatues header is not supported", sigparams.Crypto.Name)
	}
	return nil
}

// VerifyService - checks parsed service name - first item in in keyID prefix /service/user/keys/
// is the same as the name of this implementation's service
// Rationale - as there is only one BoltDB, and we want to run multiple servers on a single process,
// all the public keys go in one database. This provides a point to reject access to a service for
// a key that may be in the pubkey database, but is not named the same as the offered service.
// The /self/self/ prefixed key arises from the transient SelfPubkey implementation. This would
// be signature arising a process's own client calling its own server. We allow a process to use
// an internal client to talk to any service the process provides,
// so it is passed to the signature verification step.
func VerifyService(imp *ImplementationT, keyID string) *c.Err {
	if keyID == "" {
		return c.ErrF("unauthenticated - keyID is empty string")
	}
	terms := strings.Split(keyID, "/")
	if len(terms) < 3 {
		return c.ErrF("unauthenticated - malformed keyID provided '%s'", keyID)
	}
	if terms[1] == "self" && terms[2] == "self" {
		return nil // Pass self-key prefix /self/self/ on to the next stage
	}
	if imp.Service != terms[1] {
		return c.ErrF("unauthenticated - this is not service '%s'", terms[1])
	}
	return nil
}

// GetPublicKeyString - Attempt to get the openssh single line formatted Public Key by provided KeyID with service
// provided in ImplementationT
// (grpc codes.NotFound)
func GetPublicKeyString(imp *ImplementationT, keyID string) (string, *c.Err) {
	pubkeystring, err := imp.PubKeyLookupFunc(imp.LookupResource, keyID)
	if err != nil {
		return "", c.ErrF("unauthenticated - server cannot find public key with provided http-signatures header keyId '%s' : %v", keyID, err)
	}
	return pubkeystring, nil
}

// VerifyCrypto - packages and calls the provided cryptography algorithm to compare the signature with the
// reconstructed date header and the provided public key openssh format string.
// (grpc codes.Unauthenticated
func VerifyCrypto(sigparams *SignatureParametersT, key string) *c.Err {
	sigbytes := []byte(sigparams.Signature)
	dateheader := fmt.Sprintf("%s: %s", "date", sigparams.Date)
	dhbytes := []byte(dateheader)
	pubkey := []byte(key)

	return sigparams.Crypto.Verify(&pubkey, dhbytes, &sigbytes)
}

type headerValues map[string]string

// SignatureParse - parses the inbound signature string fields
func SignatureParse(in string) (SignatureParametersT, *c.Err) {
	s := SignatureParametersT{}
	if len(in) < 11 {
		return s, c.ErrF("authentication header contents too short %s to be valid http-signatures header", in)
	}
	// "Signature " must be start of string
	preamble := in[0:10]
	if preamble != "Signature " {
		return s, c.ErrF("authentication header contents mislabled as %s in http-signatures header", preamble)
	}
	var key, value string
	signatureRegex := regexp.MustCompile(`(\w+)="([^"]*)"`)
	for _, m := range signatureRegex.FindAllStringSubmatch(in, -1) {
		key = m[1]
		value = m[2]
		if key == "keyId" {
			s.KeyID = value
		} else if key == "algorithm" {
			crypto, cerr := CryptoFromString(value)
			if cerr != nil {
				return s, cerr
			}
			s.Crypto = crypto
		} else if key == "headers" {
			s.parseHeaders(value)
		} else if key == "signature" {
			s.Signature = value
		}
		// ignore unspecified parameters
	}
	if len(s.HeaderList) == 0 { // only the date header is supported, so explicit list not needed.
		s.HeaderList = []string{"date"}
		s.Headers = headerValues{}
	}
	if len(s.Signature) == 0 {
		return s, c.ErrF("missing required 'signature' parameter in http-signatures header")
	}
	if len(s.KeyID) == 0 {
		return s, c.ErrF("missing required 'keyId' in http-signatures header")
	}
	if s.Crypto == nil {
		return s, c.ErrF("missing required 'algorithm' in http-signatures header")
	}
	return s, nil
}

// parseHeaders - fills HeaderList
func (s *SignatureParametersT) parseHeaders(list string) {
	if len(list) == 0 {
		return
	}
	list = strings.TrimSpace(list)
	headers := strings.Split(strings.ToLower(string(list)), " ")
	for _, header := range headers {
		s.HeaderList = append(s.HeaderList, header)
	}
}
