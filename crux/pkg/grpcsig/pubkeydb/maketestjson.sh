#!/bin/bash
./ecdsapubkeytojson.sh ../test/testdata/test-key-ecdsa.pub phlogiston bobo
./ecdsapubkeytojson.sh ../test/testdata/test-key-ecdsa.pub jettison bobo
./ed25519pubkeytojson.sh ../test/testdata/test-key-ed25519.pub jettison bobo
./rsapubkeytojson.sh ../test/testdata/test-key-rsa.pub phlogiston maude
./rsapubkeytojson.sh ../test/testdata/test-key-rsa.pub jettison maude
