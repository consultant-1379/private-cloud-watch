// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"google.golang.org/grpc"

	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
)

const (
	envFingerprint = "GRPCSIG_FINGERPRINT"
	envUser        = "GRPCSIG_USER"
	envSSHAuthSock = "SSH_AUTH_SOCK"
)

// DefaultEnvSigner - retrieves environment variables GRPCSIG_FINGERPRINT, GRPCSIG_USER
// to query/match ssh-agent. Down-converts internal crux errors (c.Err) to normal go errors
// with stack trace
func DefaultEnvSigner(servicerev string) (*AgentSigner, error) {
	fingerprint := os.Getenv(envFingerprint)
	if fingerprint == "" {
		return nil, fmt.Errorf("DefaultEnvSigner environment variable %s not set", envFingerprint)
	}
	principal := os.Getenv(envUser)
	if principal == "" {
		return nil, fmt.Errorf("DefaultEnvSigner environment Variable %s not set", envUser)
	}

	var err error
	err = nil
	keyid, kerr := idutils.NewKeyID(servicerev, principal, fingerprint)
	if kerr != nil {
		err = fmt.Errorf("DefaultEnvSigner - invalid arguments - cannot make KeyID : %s %s", kerr.Err, kerr.Stack)
	}
	agentSigner, cerr := NewAgent(keyid)
	if cerr != nil {
		err = fmt.Errorf("DefaultEnvSigner - failed connecting to ssh-agent : %s %s", cerr.Err, cerr.Stack)
	}
	return agentSigner, err
}

// SelfSigner - uses the self-key bootstrapping infrastructure, rather than environment variables
// Alternative to DefaultEnvSigner()
func SelfSigner(cert *c.TLSCert) (*AgentSigner, error) {
	fingerprint := GetSelfFingerprint()
	if fingerprint == "" {
		return nil, fmt.Errorf("missing fingerprint - probably because InitSelfSSHKeys not yet called")
	}
	principal := "self"
	servicerev := "self"
	var err error
	err = nil
	keyid, kerr := idutils.NewKeyID(servicerev, principal, fingerprint)
	if kerr != nil {
		return nil, fmt.Errorf("SelfSigner - cannot make KeyID : %s %s", kerr.Err, kerr.Stack)
	}
	agentSigner, cerr := NewAgent(keyid)
	agentSigner.Certificate = cert
	if cerr != nil {
		err = fmt.Errorf("SelfSigner - failed connecting to ssh-agent %s %s", cerr.Err, cerr.Stack)
	}
	return agentSigner, err
}

// ServiceSigner - uses an idutils.KeyIDT to get a signer
func ServiceSigner(keyID idutils.KeyIDT) (*AgentSigner, error) {
	if len(keyID.Principal) == 0 || len(keyID.ServiceRev) == 0 || len(keyID.Fingerprint) == 0 {
		return nil, fmt.Errorf("ServiceSigner - empty keyID")
	}
	serviceSigner, cerr := NewAgent(keyID)
	if cerr != nil {
		return nil, fmt.Errorf("%s %s", cerr.Err, cerr.Stack)
	}
	return serviceSigner, nil
}

// AgentSigner - HTTP Signature signer using an agent (an instance of ssh-agent)
// with this, private keys never enter userspace memory.
// but there are some tradeoffs in that you get the default algorithm
// that ssh-agent applies for a given key type.
type AgentSigner struct {
	KeyID       idutils.KeyIDT
	publicKey   ssh.PublicKey
	algorithm   string
	agent       agent.Agent
	Certificate *c.TLSCert // may be nil
}

// ClientSignerT - the agent signer and public key info for a given client
// This has the handle to the ssh-agent which can sign grpc-signature
// requests for your client to communicate.
type ClientSignerT struct {
	Signer     *AgentSigner
	PubKey     PubKeyT
	PubKeyJSON string
}

// TryAgent - looks in environment variables for "SSH_AUTH_SOCK"
func TryAgent() (string, *c.Err) {
	agent := os.Getenv(envSSHAuthSock)
	if agent == "" {
		return "", c.ErrF("Environment Variable %s not set. (See man ssh-agent and man ssh-add)", envSSHAuthSock)
	}
	return agent, nil
}

// AgentFingerprintMD5 -, returns key fingerprint in md5 format
// e.g.:  "e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22"
// for rsa, ed25519 and ecdsa keys
func AgentFingerprintMD5(key *agent.Key) string {
	md5hash := md5.New()
	md5hash.Write(key.Marshal())
	rawPrint := fmt.Sprintf("%x", md5hash.Sum(nil))
	fpFormatted := ""
	for i := 0; i < len(rawPrint); i = i + 2 {
		fpFormatted = fmt.Sprintf("%s%s:", fpFormatted, rawPrint[i:i+2])
	}
	fp := strings.TrimSuffix(fpFormatted, ":")
	return fp
}

// NewAgent - Takes an md5 format query fingerprint, username and service name
// contacts the ssh-agent at SSH_AUTH_SOCK, matches the query
// fingerprint to a key, and returns an active AgentSigner
func NewAgent(keyid idutils.KeyIDT) (*AgentSigner, *c.Err) {
	agentstr, err := TryAgent() // This is a crux Err
	if err != nil {
		return nil, err
	}
	sock, gerr := net.Dial("unix", agentstr) // Here a gerr is a conventional go error
	if gerr != nil {
		return nil, c.ErrF("NewAgent - failed to connect to SSH agent - %v", gerr)
	}
	sshagent := agent.NewClient(sock) // New SSH agent
	// get the list of public keys
	keys, gerr := sshagent.List()
	if gerr != nil {
		return nil, c.ErrF("NewAgent - ssh agent could not list keys  - %v ", gerr)
	}
	// Find key matching fingerprint in md5 format
	var foundKey *agent.Key
	for _, key := range keys {
		agentfp := AgentFingerprintMD5(key)
		if agentfp == keyid.Fingerprint {
			foundKey = key
			break
		}
	}
	if foundKey == nil {
		return nil, c.ErrF("NewAgent - fingerprint %s not found in keys managed by ssh-agent.", keyid.Fingerprint)
	}
	agentsigner := &AgentSigner{
		KeyID:     keyid,
		publicKey: foundKey,
		agent:     sshagent,
	}
	alg, cerr := agentsigner.findAlg()
	if cerr != nil {
		return nil, c.ErrF("NewAgent - cannot sign using ssh-agent: %s", cerr.Error())
	}
	agentsigner.algorithm = alg
	return agentsigner, nil
}

// TODO provide ssh-agent call to remove key ??

// findAlg - Uses the agent's public key to sign some arbitrary data, then extracts the
// algorithm that the agent used.
func (s *AgentSigner) findAlg() (string, *c.Err) {
	signature, err := s.agent.Sign(s.publicKey, []byte("FEEDB0B0"))
	if err != nil {
		return "", c.ErrF("findAlg - cannot sign test signature: %v", err)
	}
	return SignatureFormatToAlg(signature.Format)
}

// SignatureFormatToAlg converts agent reported format to signature format
func SignatureFormatToAlg(format string) (string, *c.Err) {
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
		return alg, c.ErrF("SignatureFormatToAlg - key format %s is unknown", format)
	}
	return alg, nil
}

// SigPairT - holds algorithm and signed results
type SigPairT struct {
	HashAlg   string
	Base64Sig string
}

func (s *SigPairT) String() string {
	// todo - switch on HashAlg
	return s.Base64Sig
}

// Sign - Signs data, returns an interface with the results and algorithm used
func (s *AgentSigner) Sign(data string) (SigPairT, *c.Err) {
	var sigpair SigPairT
	signature, err := s.agent.Sign(s.publicKey, []byte(data))
	if err != nil {
		return sigpair, c.ErrF("Sign - agent cannot sign data: %v", err)
	}
	alg, cerr := SignatureFormatToAlg(signature.Format)
	if cerr != nil {
		return sigpair, c.ErrF("Sign - unexpected agent signature format: %v", cerr)
	}
	switch alg {
	case "rsa":
		sigpair = SigPairT{
			HashAlg:   "rsa-sha1",
			Base64Sig: base64.StdEncoding.EncodeToString(signature.Blob),
		}
	case "ecdsa":
		sigpair, cerr = EcdsaSignature(signature.Blob)
		if cerr != nil {
			return SigPairT{}, cerr
		}
	case "ed25519":
		sigpair = SigPairT{
			HashAlg:   "ed25519",
			Base64Sig: base64.StdEncoding.EncodeToString(signature.Blob),
		}
	default:
		return SigPairT{}, c.ErrF("Sign - unexpected algorithm format %s", alg)
	}
	return sigpair, nil
}

// Dial - grpc.Dial with our conventional interceptors
func (s *AgentSigner) Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	err := CheckCertificate(s.Certificate, "Dial "+target)
	if err != nil {
		return nil, err
	}
	opts = append(opts, ClientTLSOption(s.Certificate, target))

	opts = append(opts, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
		UnaryClientInterceptor(s),
		grpc_prometheus.UnaryClientInterceptor,
	)))

	opts = append(opts, grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
		StreamClientInterceptor(s),
		grpc_prometheus.StreamClientInterceptor,
	)))

//	opts = append(opts, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(UnaryClientInterceptor(s))))
//	opts = append(opts, grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(StreamClientInterceptor(s))))
	return grpc.Dial(target, opts...)
}

// PrometheusDialOpts - []grpc.DialOption for dialing without a signer
func PrometheusDialOpts(opts ...grpc.DialOption) []grpc.DialOption {
	opts = append(opts, grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor))
	opts = append(opts, grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor))
	return opts
}
