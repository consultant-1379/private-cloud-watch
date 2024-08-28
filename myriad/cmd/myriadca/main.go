package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/myriad/pkg/crypt"
	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/x509ca"
)

var serverAddress string

const Timeout = 5.0 // sec

var certAuth *myriadca.CertificateAuthority

type UserEntry struct {
	sync.Mutex
	Passwd    string    // secret shared between server and client
	Tstamp    time.Time // time of last TokenReq
	ReqToken  []byte    // sent by the client
	RespToken []byte    // sent by the server
}

var userDict map[string]*UserEntry // map user id to UserEntry

func bailout(format string, args ...interface{}) []byte {
	log.Printf("  "+format, args...)
	return nil
}

func tokenReqHandler(msg *myriadca.Msg, user *UserEntry) []byte {
	key := crypt.BytesToKey(msg.UserId, user.Passwd)
	pl, err := myriadca.DecodeMsg(msg, key)
	if err != nil {
		return bailout("tokenReqHandler: %s", err.Error())
	}
	user.ReqToken = pl.Token
	log.Printf("  recv token %x", user.ReqToken)
	user.RespToken = nil // assume the worst
	nonce, err := crypt.NewGCMNonce()
	if err != nil {
		return bailout("NewGCMNonce: %s", err.Error())
	}
	pl.Token = nonce[:]
	key = crypt.BytesToKey(msg.UserId, user.Passwd, user.ReqToken)
	msg.MsgType = myriadca.TokenResp
	resp, err := myriadca.EncodeMsg(msg, pl, key)
	if err != nil {
		return bailout("TokenReq: %s", err.Error())
	}
	user.RespToken = pl.Token
	user.Tstamp = time.Now()
	log.Printf("  send token %x", user.RespToken)
	return resp
}

func certReqHandler(msg *myriadca.Msg, user *UserEntry) []byte {
	if user.ReqToken == nil {
		return bailout("certReqHandler: no request token")
	}
	if user.RespToken == nil {
		return bailout("certReqHandler: no response token")
	}
	if time.Since(user.Tstamp).Seconds() > Timeout {
		return bailout("certReqHandler: request timed out")
	}
	key := crypt.BytesToKey(msg.UserId, user.Passwd, user.ReqToken, user.RespToken)
	pl, err := myriadca.DecodeMsg(msg, key)
	if err != nil {
		return bailout("certReqHandler: %s", err.Error())
	}
	user.ReqToken = nil // single use
	user.RespToken = nil
	if len(pl.DER) != 1 {
		return bailout("missing or multiple CSRs (%d)", len(pl.DER))
	}
	cert, err := x509ca.SignCACert(pl.DER[0], 12,
		certAuth.Last().Cert, certAuth.Last().PrivKey)
	if err != nil {
		return bailout("SignCACert: %s", err.Error())
	}
	pl.DER = nil
	for _, link := range certAuth.Chain {
		pl.DER = append(pl.DER, link.Cert.Raw)
	}
	pl.DER = append(pl.DER, cert.Raw)
	msg.MsgType = myriadca.CertResp
	resp, err := myriadca.EncodeMsg(msg, pl, key)
	if err != nil {
		return bailout("CertResp: %s", err.Error())
	}
	return resp
}

func xHandler(w http.ResponseWriter, req *http.Request) []byte {
	log.Printf("Req from %s", req.RemoteAddr)
	if req.URL.Path != "/jsonrpc/" {
		return bailout("bad path %s", req.URL.Path)
	}
	if req.ContentLength <= 0 || req.ContentLength > 4096 {
		return bailout("content length out of range: %d", req.ContentLength)
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return bailout("reading req.Body: %s", err.Error())
	}
	msg := new(myriadca.Msg)
	err = json.Unmarshal(body, msg)
	if err != nil {
		return bailout("json.Unmarshal: %s", err.Error())
	}
	user, ok := userDict[msg.UserId]
	if !ok {
		return bailout("unknown user: %s", msg.UserId)
	}
	if msg.Payload == nil {
		return bailout("empty payload")
	}
	user.Lock()
	defer user.Unlock()
	switch msg.MsgType {
	case myriadca.TokenReq:
		return tokenReqHandler(msg, user)
	case myriadca.CertReq:
		return certReqHandler(msg, user)
	}
	return bailout("unknown MsgType %d", msg.MsgType)
}

func handler(w http.ResponseWriter, req *http.Request) {
	resp := xHandler(w, req)
	w.Header().Add("X-Clacks-Overhead", "GNU Lars Magnus Ericsson")
	if resp != nil {
		w.Write(resp)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func bailOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func readConfig(fname string) {
	data, err := ioutil.ReadFile(fname)
	bailOnErr(err)
	jsondict := new(map[string]string)
	err = json.Unmarshal(data, jsondict)
	bailOnErr(err)
	userDict = make(map[string]*UserEntry)
	var ourCert *x509.Certificate
	var ourKey interface{}
	extDataMask := 0
	for k, v := range *jsondict {
		fmt.Printf(`"%s" => "%s"`+"\n", k, v)
		switch k {
		case "caChain":
			certAuth = new(myriadca.CertificateAuthority)
			for _, f := range strings.Split(v, ",") {
				link, err := x509ca.ReadCertPEMFile(f)
				bailOnErr(err)
				certAuth.Append(link, nil)
			}
			extDataMask |= 1
		case "caCert":
			ourCert, err = x509ca.ReadCertPEMFile(v)
			bailOnErr(err)
			extDataMask |= 2
		case "caKey":
			ourKey, err = x509ca.ReadKeyPEMFile(v)
			bailOnErr(err)
			extDataMask |= 4
		case "serverAddress":
			serverAddress = v
		default:
			userDict[k] = &UserEntry{Passwd: v}
		}
	}
	switch extDataMask {
	case 7: // all cert info supplied
		certAuth.Append(ourCert, ourKey)
		return
	case 0: // no cert info supplied
		certAuth, err = myriadca.MakeCertificateAuthorityTopLevels()
		if err != nil {
			bailOnErr(fmt.Errorf("MakeCertificateAuthorityTopLevels: %s", err.Error()))
		}
	default:
		bailOnErr(fmt.Errorf("Incomplete cert info in config file"))
	}
}

func main() {
	log.SetFlags(log.Ltime)
	log.SetPrefix("server: ")
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s config.json", os.Args[0])
	}
	readConfig(os.Args[1])
	if serverAddress == "" {
		log.Fatalf("no server address in config file")
	}
	http.HandleFunc("/", handler)
	log.Printf("Listening for http://%s", serverAddress)
	err := http.ListenAndServe(serverAddress, nil)
	log.Fatal(err)
}
