// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package crux

// RegisterClient  is the interface that implements node (reeve) registration protocol
// with the flock's register/steward.
type RegisterClient interface {
	AddAReeve(string, string, string) *Err
}

// AddAReeve takes registry address, encryption key, and reeve's public key as a json string
// In short - this is where the symmetric key is used to exchange the asymmetric public keys
// over a secondary channel and callback to our reeve. Provides for reeve-steward communication
// security with grpc-signatures by placing keys on both sides.
