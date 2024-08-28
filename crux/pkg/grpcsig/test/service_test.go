package service

import (
	"fmt"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/clog"
	g "github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/muck"
)

func Test(t *testing.T) { TestingT(t) }

type AgentTester struct {
	dir string
}

var _ = Suite(&AgentTester{})

const badsig1 = "Signature keyId=\"/jettison/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22\",algorithm=\"rsa-sha1\",headers=\"Date\""
const badsig2 = "Signature keyId=\"/jettison//maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22\",headers=\"Date\",signature=\"WNFFjraEMiZQ2J0Seyx2PUPqw3fjT8Rwi+Pnl/Z3PU93tmSLQ6Ok9Uaa1fIT+4h4Yi/QZnPHx/f3lc/sjYTzDnEimHlZW04tpVc2TQnDJMO03XmOsicO9wSs52AOYAX0jHhLt0/d+81wdqjUQhD0S8XGajCeORK5vt68UEeL60AN8UVjKacaeEeTFMmQv6B+ANbmTOUdoOPtaCjKbu40n0NZ96nAPdcF+XL/eKrfaKXwOdngy+MwXJ9C0OcfIP17pt62l7nCaz6G2bAnjUYsHgzJtnzZlLz4E4VIceRs7WYpuEqbh0+K0Jug7xKJ/AvNUsHVu08JjlPYnqyAeZtCaA==\""
const badsig3 = "Signature algorithm=\"rsa-sha1\",headers=\"Date\",signature=\"WNFFjraEMiZQ2J0Seyx2PUPqw3fjT8Rwi+Pnl/Z3PU93tmSLQ6Ok9Uaa1fIT+4h4Yi/QZnPHx/f3lc/sjYTzDnEimHlZW04tpVc2TQnDJMO03XmOsicO9wSs52AOYAX0jHhLt0/d+81wdqjUQhD0S8XGajCeORK5vt68UEeL60AN8UVjKacaeEeTFMmQv6B+ANbmTOUdoOPtaCjKbu40n0NZ96nAPdcF+XL/eKrfaKXwOdngy+MwXJ9C0OcfIP17pt62l7nCaz6G2bAnjUYsHgzJtnzZlLz4E4VIceRs7WYpuEqbh0+K0Jug7xKJ/AvNUsHVu08JjlPYnqyAeZtCaA==\""
const badsig4 = "Signatureino keyId=\"/jettison/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22\",algorithm=\"rsa-sha1\",headers=\"Date\",signature=\"WNFFjraEMiZQ2J0Seyx2PUPqw3fjT8Rwi+Pnl/Z3PU93tmSLQ6Ok9Uaa1fIT+4h4Yi/QZnPHx/f3lc/sjYTzDnEimHlZW04tpVc2TQnDJMO03XmOsicO9wSs52AOYAX0jHhLt0/d+81wdqjUQhD0S8XGajCeORK5vt68UEeL60AN8UVjKacaeEeTFMmQv6B+ANbmTOUdoOPtaCjKbu40n0NZ96nAPdcF+XL/eKrfaKXwOdngy+MwXJ9C0OcfIP17pt62l7nCaz6G2bAnjUYsHgzJtnzZlLz4E4VIceRs7WYpuEqbh0+K0Jug7xKJ/AvNUsHVu08JjlPYnqyAeZtCaA==\""
const badsig5 = ""

//rsa-sha1
const testdate1 = "Mon, 27 Nov 2017 22:26:06 UTC"
const testsig1 = "Signature keyId=\"/jettison/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22\",algorithm=\"rsa-sha1\",headers=\"Date\",signature=\"WNFFjraEMiZQ2J0Seyx2PUPqw3fjT8Rwi+Pnl/Z3PU93tmSLQ6Ok9Uaa1fIT+4h4Yi/QZnPHx/f3lc/sjYTzDnEimHlZW04tpVc2TQnDJMO03XmOsicO9wSs52AOYAX0jHhLt0/d+81wdqjUQhD0S8XGajCeORK5vt68UEeL60AN8UVjKacaeEeTFMmQv6B+ANbmTOUdoOPtaCjKbu40n0NZ96nAPdcF+XL/eKrfaKXwOdngy+MwXJ9C0OcfIP17pt62l7nCaz6G2bAnjUYsHgzJtnzZlLz4E4VIceRs7WYpuEqbh0+K0Jug7xKJ/AvNUsHVu08JjlPYnqyAeZtCaA==\""
const pubkey1 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa6Jdf0FADJs6t5xW6RZXGN7mN0bWJYbIKqUrLVtFfZX06WLiynChWmV7QvO4n2Ae08skMDHRaONd1nTcJonh8HMe75OTfQS/4xeCFKIt18BqwT85T7i8vnZvc6pDqLSLgQcFWMqo/51PQomZGdID+mlXK5oZnfsAabwZGcY+tbWUdnI3yzGo2XgBckQCu+nGtqCYyjlNayFs3AQ6tIhMdHmOeg9cyoqvVIb3wQW0wDgxf8rhcGoGO3Tiy4N9BCzdy9NoZd70Uq7jNkmWRS6Zg0IRW4HgIZ63mfM8Ai5HtJIoBDXdUi98OA1DXsV999wLF8JZQ169DfJJMy0bLQbhZ test-rsa-key"
const wrongkey1 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD4UH2KZ1+LvAMOuJyIuQrVzVhFX3cNQDVpT9F0VLJ3UstZni4rNwtZZgGhIx0hy2xfXy29XYAqWnv+QwX3PzyjT6fLEv5umDhuzcbPzJZO9oofrlCGClspXTVHTIKsQ1cgppzyicGubgOKnE1wVrGD6XYPfHqkGLHG99xXNPya0sl3JuPnUJ5BstjHDGoR6UnUSPcm84DAQOUmwbtCgvx2F8ZDFJFwqZiOOAUummAJrp7BwiXeTmzi2F5ZtlRSCt7T8FQ2imbRVWwS3lpF2mmud1ODaz86kwQ8SkaCH4BFfI+QCjbN/CbeFAcUtJVclzb34+rRcFSvx6mZAGEt/leT wrong-rsa-key"

//ecdsa
const testdate2 = "Tue, 28 Nov 2017 00:33:43 UTC"
const testsig2 = "Signature keyId=\"/jettison/bobo/keys/2c:f3:65:41:56:97:6c:2d:aa:08:ef:34:f3:ef:e3:c8\",algorithm=\"ecdsa-sha512\",headers=\"Date\",signature=\"AAAAQgEJbDiFwbMGVTVH7NyzpHlI+TmoSGmV+vofsrRwAUuMToMeeSPyJrb8Asxop+pOOO3oWmQX6toOonsNkW1/CtXXagAAAEIBPaOYhyUbMRdeLNvFWCpD12/i1nec1QB0XNnkloSZT4y3t0l7l71As/T9HsTEzLcl6Qex3LVvdsJ+yhyyjAFsqos=\""
const pubkey2 = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBABkFVCEb50dR3fFdq6n6gvHpa+s1iYB9tDMX38KIHaE/HEi3eK6zD9ND+E+PfkXVUkeieNBytDuh0wGycoZb/smVQBZcBy+jZgmkFn3snnSiMyQZOdRTPvXx5f4JR5aSmr4e1UOGIumNEHU/qkV/EwA7AM+ex5RJRIVK6y+l2Jb+pXhwA== test-ecdsa-key"
const wrongkey2 = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAFYsfX1NwY8KRXoyJn4BAS8qYuQSaeLAwtXyvfazygIARErMG6BM2YvPbItnY18Oq2ZDLAwj+YDPEnXAYCY3k+xfgAZXTO6JG03X/N9DnFCp8Y9rkxKatWcqTxYONPsYZ3Fyi2AF4/fqTxJ5ZX8hq0kkJRDuLTOAsJ43+6oH1jNmHqfkg== wrong-ecdsa-key"

//ed25119
const testdate3 = "Tue, 28 Nov 2017 00:37:40 UTC"
const testsig3 = "Signature keyId=\"/jettison/bobo/keys/6f:d4:de:43:e2:6c:de:7c:44:2d:33:4a:d1:35:ab:8a\",algorithm=\"ed25519\",headers=\"Date\",signature=\"M4Phx3TY0AjDJVjpFhc2b37tLvoWknN/J1eI+l8hCAbLYQrBM/hLkoKMcBBqeXIozO8SlJP0XYaX60vvOt4WBQ==\""
const pubkey3 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINyH0WROU2WyByo3Mq0yUYaI1uaQvTLAL1OLMVLOytQ2 test-ed25519-key"
const wrongkey3 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBCVUTJzFqzt48PzqADMVs4EcpViEKXU4KTN1hnCPuh0 wrong-ed25519-key"

const futuredate = "Mon, 22 Nov 2027 22:26:06 UTC"
const emptykey = ""

const fingerprint1 = "e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22"
const username1 = "maude"
const service1 = "jettison"

const fingerprint2 = "2c:f3:65:41:56:97:6c:2d:aa:08:ef:34:f3:ef:e3:c8"
const username2 = "bobo"
const service2 = "jettison"

const fingerprint3 = "6f:d4:de:43:e2:6c:de:7c:44:2d:33:4a:d1:35:ab:8a"
const username3 = "bobo"
const service3 = "jettison"

const fingerprint4 = "8f:d4:de:43:e2:6c:de:7c:44:2d:33:4a:d1:35:ab:8a"
const username4 = "dada"
const service4 = "phlogiston"

func (s *AgentTester) SetUpTest(c *C) {
	s.dir = c.MkDir()
	//s.dir = "./"
	fmt.Printf("Working Directory: %s", s.dir)
}

func (s *AgentTester) TestService(c *C) {
	logger := clog.Log.With("focus", "service_test")
	fmt.Printf("\nTest Default Public Key DB service\n")
	dbname := "server/pubkeys_test.db"
	imp1, err := g.InitDefaultService(dbname, service1, nil, logger, 300, false)
	fmt.Printf("InitDefaultService Errors?  [%v]\n", err)
	c.Assert(err, IsNil)

	fmt.Printf("Test Public Key Lookup\n")
	keylookup := fmt.Sprintf("/%s/%s/keys/%s", service1, username1, fingerprint1)
	keystring, serr := g.GetPublicKeyString(&imp1, keylookup)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey1)

	keylookup = fmt.Sprintf("/%s/%s/keys/%s", service1, username4, fingerprint1)
	keystring, serr = g.GetPublicKeyString(&imp1, keylookup)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)

	keylookup = fmt.Sprintf("/%s/%s/keys/%s", service2, username2, fingerprint2)
	keystring, serr = g.GetPublicKeyString(&imp1, keylookup)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey2)

	keylookup = fmt.Sprintf("/%s/%s/keys/%s", service3, username3, fingerprint3)
	keystring, serr = g.GetPublicKeyString(&imp1, keylookup)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey3)

	keylookup = fmt.Sprintf("/%s/%s/keys/%s", service4, username4, fingerprint4)
	keystring, serr = g.GetPublicKeyString(&imp1, keylookup)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)

	fmt.Printf("\nTest Clock Skew Detection\n")
	sigparams := g.SignatureParametersT{}
	sigparams.Timestamp, _ = time.Parse(time.RFC1123, testdate1)
	terr := g.VerifyClockskew(300, &sigparams)
	c.Assert(terr, Not(IsNil))
	fmt.Printf("  %v\n", terr)

	sigparams.Timestamp, _ = time.Parse(time.RFC1123, futuredate)
	terr = g.VerifyClockskew(300, &sigparams)
	c.Assert(terr, Not(IsNil))
	fmt.Printf("  %v\n", terr)

	fmt.Printf("\nTest Signature Parsing\n")
	sigparams, serr = g.SignatureParse(badsig1)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)
	sigparams, serr = g.SignatureParse(badsig2)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)
	sigparams, serr = g.SignatureParse(badsig3)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)
	sigparams, serr = g.SignatureParse(badsig4)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)
	sigparams, serr = g.SignatureParse(badsig5)
	c.Assert(serr, Not(IsNil))
	fmt.Printf("  %v\n", serr)

	fmt.Printf("\nTest Crypto Algorithm restrictions\n")
	sigparams, serr = g.SignatureParse(testsig1)
	c.Assert(serr, IsNil)
	aerr := g.VerifyAlgorithm(&imp1, &sigparams)
	c.Assert(aerr, IsNil)
	imp1.Algorithms = []string{"wobbocrypt", "", "bobs-easycrypto", "rsa-sha1-and-friends"}
	aerr = g.VerifyAlgorithm(&imp1, &sigparams)
	c.Assert(aerr, Not(IsNil))
	fmt.Printf("  %v\n", aerr)

	fmt.Printf("\nTest rsa-sha1 signature verification with db retrieved public key and wrong keys\n")
	sigparams, serr = g.SignatureParse(testsig1)
	c.Assert(serr, IsNil)
	sigparams.Date = testdate1
	keystring, serr = g.GetPublicKeyString(&imp1, sigparams.KeyID)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey1)
	verr := g.VerifyCrypto(&sigparams, keystring)
	c.Assert(verr, IsNil)
	verr = g.VerifyCrypto(&sigparams, wrongkey1)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, wrongkey2)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, wrongkey3)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, emptykey)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)

	fmt.Printf("\nTest ecdsa signature verification with db retrieved public key and wrong keys\n")
	sigparams, serr = g.SignatureParse(testsig2)
	c.Assert(serr, IsNil)
	sigparams.Date = testdate2
	keystring, serr = g.GetPublicKeyString(&imp1, sigparams.KeyID)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey2)
	verr = g.VerifyCrypto(&sigparams, keystring)
	c.Assert(verr, IsNil)
	verr = g.VerifyCrypto(&sigparams, wrongkey1)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, wrongkey2)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, wrongkey3)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, emptykey)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)

	fmt.Printf("\nTest ed25519 signature verification with db retrieved public key and wrong keys\n")
	sigparams, serr = g.SignatureParse(testsig3)
	c.Assert(serr, IsNil)
	sigparams.Date = testdate3
	keystring, serr = g.GetPublicKeyString(&imp1, sigparams.KeyID)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey3)
	verr = g.VerifyCrypto(&sigparams, keystring)
	c.Assert(verr, IsNil)
	verr = g.VerifyCrypto(&sigparams, wrongkey1)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, wrongkey2)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, wrongkey3)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)
	verr = g.VerifyCrypto(&sigparams, emptykey)
	c.Assert(verr, Not(IsNil))
	fmt.Printf("  %v\n", verr)

	fmt.Printf("\n\nSelf ssh-keygen testing\n")

	kerr := g.SelfKeyFileExists()
	fmt.Printf("SelfKeyFileExists() Errors?: [%v]\n\n", kerr)
	c.Assert(kerr, Not(IsNil))

	fmt.Printf("\nCalling InitSelfSSHKeys()\n")
	kerr = g.InitSelfSSHKeys(true)
	fmt.Printf("InitSelfSSHKeys() Errors?: [%v]\n", kerr)
	c.Assert(kerr, Not(IsNil))

	fmt.Printf("Starting Muck")
	merr := muck.InitMuck(s.dir+"/"+".muck", "")
	fmt.Printf("IntiMuck() Errors?: [%v]\n", merr)
	c.Assert(merr, IsNil)

	fmt.Printf("\nCalling InitSelfSSHKeys()\n")
	kerr = g.InitSelfSSHKeys(true)
	fmt.Printf("InitSelfSSHKeys() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	pubkey := g.GetSelfPubKey()
	fmt.Printf("GetSelfPubKey(): [%v]\n", pubkey)
	c.Assert(pubkey, Not(IsNil))

	kerr = g.SelfKeyFileExists()
	fmt.Printf("SelfKeyFileExists() Errors?: %v\n", kerr)
	c.Assert(kerr, IsNil)

	fmt.Printf("\nCalling ListKeysFromAgent()\n")
	_, kerr = g.ListKeysFromAgent(true) // Does ssh-agent have it?
	fmt.Printf("ListKeysFromAgent() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// Add our shiny new self public key to a Bolt db
	fmt.Printf("\nBoltDB public key database test add\n")
	testdb := s.dir + "/" + " testing.db"
	oerr := g.StartNewPubKeyDB(testdb)
	c.Assert(oerr, IsNil)
	derr := g.InitPubKeyLookup(testdb, logger)
	c.Assert(derr, IsNil)

	kerr = g.AddPubKeyToDB(pubkey)
	fmt.Printf("AddPubKeyToDB() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// Try to look up our self-key
	fmt.Printf("\nTest Self Key Lookup\n")
	imp2, err := g.AddAnotherService(imp1, "self", nil, 0)

	keylookup = pubkey.KeyID
	keystring, serr = g.GetPublicKeyString(&imp2, keylookup)
	fmt.Printf("GetPublicKeyString() Errors?: [%v]\n", err)
	c.Assert(serr, IsNil)
	c.Assert(keystring, Equals, pubkey.PubKey)

	// Remove it from the same Bolt db
	fmt.Printf("\nBoltDB public key database test remove\n")
	kerr = g.RemoveSelfPubKeysFromDB()
	fmt.Printf("RemoveSelfPubKeysFromDB() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	fmt.Printf("\nRemove key files, and keys from ssh-agent\n")
	// Remove that shiny new private key from ssh-agent, and the file(s) themselves
	kerr = g.FiniSelfSSHKeys(true)
	fmt.Printf("FiniSelfSSHKeys() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	// Other tests for NewKeyPair, ecdsa format and errors
	fmt.Printf("\nTest NewKeyPair ecdsa, error messages\n")
	var pkt *g.PubKeyT
	pkt, kerr = g.NewKeyPair("ecdsa", s.dir+"/.muck/go-test-key-ecdsa"+g.GetSelfPidStr(), "MyPassPhrase", g.GetSelfPidStr(), true) // ecdsa with passphrase
	fmt.Printf("%v\n", pkt)
	fmt.Printf("NewKeyPair() Errors?: [%v]\n", kerr)
	c.Assert(kerr, IsNil)

	_, kerr = g.NewKeyPair("bobsyourunclkecrypto", s.dir+"/.muck/bob-test-rsa", "", "comment", true) // ssh-keygen error
	fmt.Printf("NewKeyPair() Errors?: [%v]\n", kerr)
	c.Assert(kerr, Not(IsNil))

	fmt.Printf("Completed.\n")
}
