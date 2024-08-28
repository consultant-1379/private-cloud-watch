package grpcsig

import (
	"context"
	"encoding/asn1"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crypto/pkg/tls"
	"github.com/erixzone/crypto/pkg/x509/pkix"
)

// CheckCertificate : rhubarb
func CheckCertificate(cert *crux.TLSCert, src string) *crux.Err {
	if cert == nil {
		msg := fmt.Sprintf("TLS %s: Certificate is nil", src)
		crux.GetLogger().Log("SEV", "INFO", msg)
		return nil
	}
	msg := fmt.Sprintf("TLS CheckCertificate %s: %s", src, cert.Leaf.Leaf.Subject.String())
	crux.GetLogger().Log("SEV", "INFO", msg)
	return nil
}

// gratefully cribbed from: grpc/credentials/credentials.go (mostly).
// this dissociates the vendor grpc from our local tls and x509,
// which have support added for ED25519.

func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{}
	}
	return cfg.Clone()
}

// tlsInfo contains the auth information for a TLS authenticated connection.
// It implements the AuthInfo interface.
type tlsInfo struct {
	State tls.ConnectionState
}

// AuthType returns the type of tlsInfo as a string.
func (t tlsInfo) AuthType() string {
	return "tls"
}

// tlsCreds is the credentials required for authenticating a connection using TLS.
type tlsCreds struct {
	// TLS configuration
	config *tls.Config
}

func (c tlsCreds) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		SecurityVersion:  "1.2",
		ServerName:       c.config.ServerName,
	}
}

func (c *tlsCreds) ClientHandshake(ctx context.Context, authority string, rawConn net.Conn) (_ net.Conn, _ credentials.AuthInfo, err error) {
	// use local cfg to avoid clobbering ServerName if using multiple endpoints
	cfg := cloneTLSConfig(c.config)
	if cfg.ServerName == "" {
		colonPos := strings.LastIndex(authority, ":")
		if colonPos == -1 {
			colonPos = len(authority)
		}
		cfg.ServerName = authority[:colonPos]
	}
	conn := tls.Client(rawConn, cfg)
	errChannel := make(chan error, 1)
	go func() {
		errChannel <- conn.Handshake()
	}()
	select {
	case err := <-errChannel:
		if err != nil {
			return nil, nil, err
		}
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	return conn, tlsInfo{conn.ConnectionState()}, nil
}

func (c *tlsCreds) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn := tls.Server(rawConn, c.config)
	if err := conn.Handshake(); err != nil {
		return nil, nil, err
	}
	return conn, tlsInfo{conn.ConnectionState()}, nil
}

func (c *tlsCreds) Clone() credentials.TransportCredentials {
	return newTLS(c.config)
}

func (c *tlsCreds) OverrideServerName(serverNameOverride string) error {
	c.config.ServerName = serverNameOverride
	return nil
}

// alpnProtoStr are the specified application level protocols for gRPC.
var alpnProtoStr = []string{"h2"}

// newTLS uses c to construct a TransportCredentials based on TLS.
func newTLS(c *tls.Config) credentials.TransportCredentials {
	tc := &tlsCreds{cloneTLSConfig(c)}
	tc.config.NextProtos = alpnProtoStr
	return tc
}

// end of gratefully cribbed section

// TCT : a wrapper to simplify logging
type TCT struct {
	serverName string
	tc         credentials.TransportCredentials
}

// ClientHandshake : rhubarb
func (tct TCT) ClientHandshake(a context.Context, b string, c net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn, authInfo, err := tct.tc.ClientHandshake(a, b, c)
	//msg := fmt.Sprintf("TLS ClientHandshake %s (%s) returns: Conn=%+v, AuthInfo=%+v, error=%+v", tct.serverName, b, conn, authInfo, err)
	msg := fmt.Sprintf("TLS ClientHandshake %s:", tct.serverName)
	if authInfo != nil {
		state := authInfo.(tlsInfo).State
		msg += fmt.Sprintf(" CipherSuite=%#04x", state.CipherSuite)
	}
	if err != nil {
		msg += fmt.Sprintf(" error=%+v", err)
	}
	crux.GetLogger().Log("SEV", "INFO", msg)
	return conn, authInfo, err
}

// ServerHandshake : rhubarb
func (tct TCT) ServerHandshake(c net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn, authInfo, err := tct.tc.ServerHandshake(c)
//	msg := fmt.Sprintf("TLS ServerHandshake returns: Conn=%+v, AuthInfo=%+v, error=%+v", conn, authInfo, err)
	msg := fmt.Sprintf("TLS ServerHandshake: RemoteAddr=%s", c.RemoteAddr().String())
	if authInfo != nil {
		state := authInfo.(tlsInfo).State
		msg += fmt.Sprintf(", ServerName=%s, CipherSuite=%#04x",
			state.ServerName, state.CipherSuite)
	}
	if err != nil {
		msg += fmt.Sprintf(" error=%+v", err)
	}
	crux.GetLogger().Log("SEV", "INFO", msg)
	return conn, authInfo, err
}

// Info : rhubarb
func (tct TCT) Info() credentials.ProtocolInfo {
	msg := fmt.Sprintf("TLS Info %s", tct.serverName)
	crux.GetLogger().Log("SEV", "INFO", msg)
	return tct.tc.Info()
}

// Clone : rhubarb
func (tct TCT) Clone() credentials.TransportCredentials {
	msg := fmt.Sprintf("TLS Clone %s", tct.serverName)
	crux.GetLogger().Log("SEV", "INFO", msg)
	return tct.tc.Clone()
}

// OverrideServerName : rhubarb
func (tct TCT) OverrideServerName(s string) error {
	msg := fmt.Sprintf("TLS OverrideServerName %s", tct.serverName)
	crux.GetLogger().Log("SEV", "INFO", msg)
	return tct.tc.OverrideServerName(s)
}

// ServerTLSOption : option parameter for grpc.NewServer()
func ServerTLSOption(cert *crux.TLSCert) grpc.ServerOption {
	if cert == nil {
		return nil
	}
	tlsConfig := tls.Config{
		ClientAuth:       tls.RequireAndVerifyClientCert,
		Certificates:     []tls.Certificate{cert.Leaf},
		ClientCAs:        cert.Pool,
		CipherSuites:     []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		CurvePreferences: []tls.CurveID{tls.X25519},
	}
	transportCreds := newTLS(&tlsConfig)
	tct := TCT{
		serverName: "<server side>",
		tc:         transportCreds,
	}
	return grpc.Creds(tct)
}

// ParseName : extract a DN from a DER sequence
func ParseName(data []byte) (*pkix.Name, []byte, error) {
	seq := new(pkix.RDNSequence)
	tail, err := asn1.Unmarshal(data, seq)
	if err != nil {
		return nil, data, err
	}
	name := new(pkix.Name)
	name.FillFromRDNSequence(seq)
	return name, tail, nil
}

// ClientTLSOption : option parameter for grpc.Dial()
func ClientTLSOption(cert *crux.TLSCert, serverName string) grpc.DialOption {
	if cert == nil {
		return grpc.WithInsecure()
	}
	tlsConfig := tls.Config{
		RootCAs:          cert.Pool,
		CipherSuites:     []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		CurvePreferences: []tls.CurveID{tls.X25519},
	}
	tct := TCT{
		serverName: serverName,
	}
	tlsConfig.GetClientCertificate =
		func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			msg := fmt.Sprintf("TLS GetClientCertificate %s: AcceptableCAs=[",
				tct.serverName)
			for _, d := range cri.AcceptableCAs {
				if name, _, _ := ParseName(d); name != nil {
					msg += fmt.Sprintf("[%s]", name.String())
				}
			}
			msg += fmt.Sprintf("] SignatureSchemes=%x", cri.SignatureSchemes)
			crux.GetLogger().Log("SEV", "INFO", msg)
			return &cert.Leaf, nil
		}

	tct.tc = newTLS(&tlsConfig)
	return grpc.WithTransportCredentials(tct)
}
