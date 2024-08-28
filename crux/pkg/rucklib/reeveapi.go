// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package rucklib

import (
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"time"
)

// ReeveAPI is the interface used for the ssh-agent signing functions
// and grpc-signatured components for the reeve API that cannot be called via gRPC
//   i.e. we cannot send pointers over the network,
//   even to ourself, not even with unsafe, not even with interfaces.
// TODO: rename to something like SignatureAPI. ( Reeve.StateT implements)
type ReeveAPI interface {
	SetEndPtsHorde(string) string
	ReeveCallBackInfo() (string, string, string, string, **grpcsig.ImplementationT)
	LocalPort() int
	// SecureService(string) interface{}
	// ClientSigner(string) (interface{}, *Err)
	// SelfSigner() interface{}
	// PubKeysFromSigner(interface{}) (string, string)
	SecureService(string) **grpcsig.ImplementationT // Note: ImplementationT has no methods.
	ClientSigner(string) (**grpcsig.ClientSignerT, *crux.Err)
	SelfSigner() **grpcsig.ClientSignerT
	PubKeysFromSigner(**grpcsig.ClientSignerT) (string, string)
	StartStewardIO(time.Duration) *crux.Err
	StopStewardIO()
	GetCertificate() *crux.TLSCert
}

// Reeve Information Calls:
// SetEndPtsHorde(string) - sets Horde name in persistant storage for all endpoints of a Reeve
// ReeveCallBackInfo() - returns string forms of reeveNodeID, reeveNetID, reeveKeyID, reevePubKeyJSON,
// and reeve's grpcsig.ImplementationT as an interface
// LocalPort() - Returns the gRPC port to connect to reeve on localhost
//
// Bootstrapping - connect a newly started reeve to steward:
// StartStewardIO(time.Duration) starts reeve's Ingest Events so it can forward information to Steward.
// with the timeout you provide.
//
// Start grpcsignatures on a grpc service:
// SecureService(serviceRev string) - returns interface to a *grpcsig.ImplementationT
//
// Get a signer for a grpc client:
// ClientSigner(serviceRev string) - returns interface to a *reeve.ClientSignerT
// SelfSigner() - returns interface to a reeve.ClientSignerT - for in-process grpc calls (i.e. calling your reeve)
// PubKeysFromSigner(interface{}) - utility - takes a ClientSigner interface, returns its KeyID and PubKeyJSON
