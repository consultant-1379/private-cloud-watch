// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
)

// gRPC CLIENT Interceptors:

// UnaryClientInterceptor - installs the http-signatures ssh-agent signer on the client grpc unary data handler
// takes an active AgentSigner as argument
func UnaryClientInterceptor(signer *AgentSigner) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx, _ := SignDateHeader(ctx, signer)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor - installs the http-signatures ssh-agent signer on the client grpc stream data handler
// takes an active AgentSigner as argument
func StreamClientInterceptor(signer *AgentSigner, optFunc ...grpc.CallOption) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		newCtx, _ := SignDateHeader(ctx, signer)
		// must call streamer
		return streamer(newCtx, desc, cc, method, opts...)
	}
}

// gRPC SERVER Configurable Parameters:

// ImplementationT - Structure carrying runtime parameters for the server interceptor
// service including preferred KeyID lookup system, logging system, clockskew in seconds
type ImplementationT struct {
	LookupResource   string
	Service          string
	PubKeyLookupFunc PubKeyLookupFuncT
	Logger           clog.Logger // was c.ConfigurableLogger, but no
	ClockSkew        int
	Algorithms       []string
	Started          bool
	Certificate      *c.TLSCert // may be nil
}

// PubKeyLookupFuncT - Type for the lookup function that queries public key fingerprint,
// returns public key as unparsed openssh formatted string. First parameter is a resource
// string (e.g. filename or url), second parameter is the http-signatures KeyID
// The built-in default is the BoltDB PubKeysDBLookup().
type PubKeyLookupFuncT func(string, string) (string, *c.Err)

// DefaultAlgorithms - Algorithms supplied in this package
var DefaultAlgorithms = []string{"rsa-sha1", "ecdsa-sha512", "ed25519"}
var defaultServiceInitialized = false

// InitDefaultService - Starts up public key lookup database, default logger,
// user provides database filename for PubKeysDBLookup() and desired clockskew
func InitDefaultService(dbfile, servicename string, cert *c.TLSCert, logger clog.Logger, clockskew int, watch bool) (ImplementationT, *c.Err) {
	if defaultServiceInitialized {
		FiniPubKeyLookup()
	}
	if !PubKeyDBExists(dbfile) {
		derr := StartNewPubKeyDB(dbfile)
		if derr != nil {
			return ImplementationT{}, derr
		}
	}
	cerr := InitPubKeyLookup(dbfile, logger)
	if cerr != nil {
		return ImplementationT{}, c.ErrF("%s; Stack: %s", cerr.String(), cerr.Stack)
	}
	if clockskew <= 0 {
		return ImplementationT{}, c.ErrS("Cannot start grpcsig - Clockskew too small, must be > 0 seconds")
	}
	defaultServiceInitialized = true
	if watch {
		// Start the whitelist database watcher
		go DBWatcher(dbfile)
	}
	if logger == nil { // make one up so we don't leave this nil
		logger = clog.Log.With("focus", "srv_"+servicename)
	}
	cerr = CheckCertificate(cert, "InitDefaultService")
	if cerr != nil {
		return ImplementationT{}, cerr
	}
	return ImplementationT{
		PubKeyLookupFunc: PubKeysDBLookup,
		Service:          servicename,
		Logger:           logger,
		ClockSkew:        clockskew,
		LookupResource:   dbfile,
		Algorithms:       DefaultAlgorithms,
		Started:          true,
		Certificate:      cert,
	}, nil
}

// AddAnotherService - Adds another service after InitDefaultService has started.
func AddAnotherService(imp ImplementationT, servicename string, logger clog.Logger, clockskew int) (ImplementationT, *c.Err) {
	if clockskew <= 0 { // use the same one
		clockskew = imp.ClockSkew
	}
	if logger == nil { // use the same one
		logger = imp.Logger
	}
	return ImplementationT{
		PubKeyLookupFunc: imp.PubKeyLookupFunc,
		Service:          servicename,
		Logger:           logger,
		ClockSkew:        clockskew,
		LookupResource:   imp.LookupResource,
		Algorithms:       imp.Algorithms,
		Started:          imp.Started,
		Certificate:      imp.Certificate,
	}, nil
}

// FiniDefaultService - Shuts down Public Key database lookup BoltDB system
func FiniDefaultService() {
	FiniPubKeyLookup()
	defaultServiceInitialized = false
}

// gRPC SERVER Interceptors

// UnaryServerInterceptor - installs the http-signatures verification on the grpc unary data handler
// uses the implementation information as argument
func UnaryServerInterceptor(imp ImplementationT) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, service interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var newCtx context.Context
		var err error
		// VerifyAll returns grpc style errors
		newCtx, err = VerifyAll(ctx, &imp)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, service)
	}
}

//StreamServerInterceptor installs the http-signatures verification on the grpc stream data handler
//takes the external public key lookup function as argument
func StreamServerInterceptor(imp ImplementationT) grpc.StreamServerInterceptor {
	return func(service interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var newCtx context.Context
		var err error
		// VerifyAll returns grpc style errors
		newCtx, err = VerifyAll(stream.Context(), &imp)
		if err != nil {
			return err
		}
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(service, wrapped)
	}
}

// NewServer : grpc.NewServer with our conventional interceptors
func (imp ImplementationT) NewServer(opts ...grpc.ServerOption) *grpc.Server {
	_ = CheckCertificate(imp.Certificate, "NewServer")
	tlsOpt := ServerTLSOption(imp.Certificate)
	if tlsOpt != nil {
		opts = append(opts, tlsOpt)
	}

	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		UnaryServerInterceptor(imp),
		grpc_prometheus.UnaryServerInterceptor,
	)))

	opts = append(opts, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
		StreamServerInterceptor(imp),
		grpc_prometheus.StreamServerInterceptor,
	)))

	return grpc.NewServer(opts...)
}

// NewTLSServer : grpc.NewServer with TLS and prometheus interceptors only
func NewTLSServer(cert *c.TLSCert, opts ...grpc.ServerOption) *grpc.Server {
	_ = CheckCertificate(cert, "NewTLSServer")
	tlsOpt := ServerTLSOption(cert)
	if tlsOpt != nil {
		opts = append(opts, tlsOpt)
	}
	opts = append(opts, grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor))
	opts = append(opts, grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor))

	return grpc.NewServer(opts...)
}
