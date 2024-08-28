package myriadca

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/myriad/pkg/crypt"
)

// message types
const (
	_ = iota
	TokenReq
	TokenResp
	CertReq
	CertResp
)

// MsgHdr : rhubarb
type MsgHdr struct {
	MsgType int    `json:"T"`
	UserId  string `json:"U"`
}

// Msg : rhubarb
type Msg struct {
	MsgHdr
	Payload []byte `json:"P"`
}

// MsgPayload : rhubarb
type MsgPayload struct {
	Token []byte   `json:"N,omitempty"`
	DER   [][]byte `json:"C,omitempty"`
}

// DecodeMsg : rhubarb
func DecodeMsg(msg *Msg, key *crypt.Key) (*MsgPayload, error) {
	hdrBits, err := json.Marshal(msg.MsgHdr)
	if err != nil {
		return nil, err
	}
	plaintext, err := crypt.DecryptWithAdd(msg.Payload, hdrBits, key)
	if err != nil {
		return nil, err
	}
	log.Printf("DecodeMsg payload: %d bytes\n", len(plaintext))
	pl := new(MsgPayload)
	err = json.Unmarshal(plaintext, pl)
	if err != nil {
		return nil, err
	}
	return pl, nil
}

// EncodeMsg : rhubarb
func EncodeMsg(msg *Msg, data *MsgPayload, key *crypt.Key) ([]byte, error) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	hdrBits, err := json.Marshal(msg.MsgHdr)
	if err != nil {
		return nil, err
	}
	msg.Payload, err = crypt.EncryptWithAdd(plaintext, hdrBits, key)
	if err != nil {
		return nil, err
	}
	return json.Marshal(msg)
}

// postExch : rhubarb
func postExch(url string, msgOut *Msg, plOut *MsgPayload, keyOut, keyIn *crypt.Key) (*MsgPayload, error) {
	data, err := EncodeMsg(msgOut, plOut, keyOut)
	if err != nil {
		return nil, fmt.Errorf("EncodeMsg: %s", err.Error())
	}
	log.Printf("send body %d bytes", len(data))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	data, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	log.Printf("recv body %d bytes", len(data))
	if len(data) > 8192 {
		return nil, fmt.Errorf("response body too long: %d", len(data))
	}
	msgIn := new(Msg)
	err = json.Unmarshal(data, msgIn)
	if err != nil {
		return nil, err
	}
	if msgIn.MsgType != msgOut.MsgType+1 {
		return nil, fmt.Errorf("protocol error: got MsgType %d, expected %d",
			msgIn.MsgType, msgOut.MsgType+1)
	}
	if msgIn.UserId != msgOut.UserId {
		return nil, fmt.Errorf("protocol error: got UserId %s, expected %s",
			msgIn.UserId, msgOut.UserId)
	}
	plIn, err := DecodeMsg(msgIn, keyIn)
	if err != nil {
		return nil, fmt.Errorf("DecodeMsg: %s", err.Error())
	}
	return plIn, nil
}

// RequestCert : rhubarb
func RequestCert(url, userId, passwd string, csr []byte) (*CertificateAuthority, error) {
	nonce, err := crypt.NewGCMNonce()
	if err != nil {
		return nil, fmt.Errorf("NewGCMNonce: %s", err.Error())
	}
	reqToken := nonce[:]

	msg := new(Msg)
	msg.MsgType = TokenReq
	msg.UserId = userId
	log.Printf("send: %#v", msg.MsgHdr)
	pl := new(MsgPayload)
	pl.Token = reqToken
	log.Printf("send token %x", reqToken)

	key0 := crypt.BytesToKey(userId, passwd)
	key1 := crypt.BytesToKey(userId, passwd, reqToken)
	pl, err = postExch(url, msg, pl, key0, key1)
	if err != nil {
		return nil, fmt.Errorf("postExch: %s", err.Error())
	}

	respToken := pl.Token
	log.Printf("recv token %x", respToken)

	msg.MsgType = CertReq
	log.Printf("send: %#v", msg.MsgHdr)
	pl = new(MsgPayload)
	pl.DER = append(pl.DER, csr)

	key2 := crypt.BytesToKey(userId, passwd, reqToken, respToken)
	pl, err = postExch(url, msg, pl, key2, key2)
	if err != nil {
		return nil, fmt.Errorf("postExch: %s", err.Error())
	}

	ca := new(CertificateAuthority)
	for _, data := range pl.DER {
		c, err := x509.ParseCertificate(data)
		if err != nil {
			return nil, fmt.Errorf("x509.ParseCertificate: %s", err.Error())
		}
		ca.Append(c, nil)
	}
	return ca, nil
}
