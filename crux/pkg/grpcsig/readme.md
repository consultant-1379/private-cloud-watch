# grpcsig - Golang grpc http-signatures

Implementation of http-signatures for gRPC client/server systems, see
`https://tools.ietf.org/html/draft-cavage-http-signatures-08#section-3.1`

Limitations: only signs the Date header, not additional headers, nor the entire grpc contents.

## To run the /test/client/server demo

Start with `./pubkeydb/readme.md`

	# Make the test key database `pubkeys.db`.
	$ cd pubkeydb
	$ ./maketestjson.sh > pubkeys.json
	$ go run pubkeydb.go
	Loading pubkeys.db from pubkeys.json
	Finished
	# Optional - Check
	$ go run pubkeydb.go -j -o ./pubkeys.db
	{"name":"bobo"
	# Move the test key database to `grpc/server/`
	$ mv pubkeys.db ../test/server/pubkeys_test.db
	$ cd ..

Run the server

	$ cd ./test/server
	$ go run main.go

Start another terminal session

Use ssh-add commands to set up the ssh-agent needed by the client

	$ ssh-add -k test/testdata/test-key-rsa
	Identity added: test/testdata/test-key-rsa (test/testdata/test-key-rsa)
	# Check
	$ ssh-add -l
	2048 SHA256:bvSuxkEkGhYBzmgSVdknDDimyHisFyBzGcC1vN1ngd8 test/testdata/test-key-rsa (RSA)

Set up the environments variables

	$ source ./test/client/envars_test
	# Check
	$ env | grep GRPCSIG_
	GRPCSIG_FINGERPRINT=e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22
	GRPCSIG_USER=maude

Run the client

	$ go run ./test/client/main.go

When you are done, clear out the ssh-agent with

	$ ssh-add -D
	All identities removed.
	$ ssh-add -l
	The agent has no identities.

And shut down the server with `control-c`


To Test with ecdsa or ed25519 keys - substitute these test keys for the ssh-agent

	$ ssh-add -k test/testdata/test-key-ed25519
	Identity added: test/testdata/test-key-ed25519 (test-ed25519-key)

	$ ssh-add -k test/testdata/test-key-ecdsa
	Identity added: test/testdata/test-key-ecdsa (test/testdata/test-key-ecdsa)

Then - Comment/uncomment the line in `./client/envars_test` to change the corresponding
environment variables to match.

## Middleware

Code implemented as an [Interceptor](github.com/grpc-ecosystem/go-grpc-middleware) pattern.

## Compiling the protos in ./test/

protoc -I test/protos/ test/protos/sigtest.proto --go_out=plugins=grpc:test/gen/sigtest
