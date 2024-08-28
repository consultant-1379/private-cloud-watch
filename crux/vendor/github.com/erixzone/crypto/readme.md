# CRYPTO

this tree contains versions of standard golang and golang/x
crypto packages with support added for curve25519.

ed25519:

    scatter-gather for ed25519 signature functions.
    conversion of ed25519 public keys to curve25519 public keys, per libsodium.

tls:

    support ED25519 signatures.

x509:

    support Ed25519 cert's per FiloSottile 2017-12-20
    (https://github.com/golang/go/compare/master...FiloSottile:filippo/ed25519).

    add support for ed25519 to {Parse|Marshall}PKCS8PrivateKey.
    export MarshalPublicKey for keyid calculation per rfc3280.
    add DomainComponent to Certificate.Subject.

    rename the FetchPEMRoots function to avoid system software clash;
    this is a C function that only appears when building under darwin.
