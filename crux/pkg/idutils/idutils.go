// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package idutils

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	c "github.com/erixzone/crux/pkg/crux"
)

const badID = "bad service identifier >%s<"

// NetIDT - The netID broken into parts
type NetIDT struct {
	ServiceRev string `json:"servicerev"`
	Principal  string `json:"principal"`
	Host       string `json:"host"`  // IPV4 or IPV6 host portion of address - see grpc.Dial()
	Port       string `json:"port"`  // port part of the address
	Query      bool   `json:"query"` // Is a field in this NetIDT a query regexp or wildcard?
}

// SplitPort - returns port part of an address
func SplitPort(address string) int {
	if address == "" {
		return 0
	}
	parts := strings.SplitAfter(address, ":")
	portstr := parts[len(parts)-1]
	portint, _ := strconv.Atoi(portstr)
	return portint
}

// SplitHost - returns host part of an address
func SplitHost(address string) string {
	if address == "" {
		return ""
	}
	parts := strings.Split(address, ":")
	hoststr := parts[0]
	return hoststr
}

// Address - returns the address string host + port
func (n *NetIDT) Address() string {
	return fmt.Sprintf("%s%s", n.Host, n.Port)
}

// String - returns the netID in string form
func (n *NetIDT) String() string {
	return fmt.Sprintf("/%s/%s/net/%s%s", n.ServiceRev, n.Principal, n.Host, n.Port)
}

// NewNetID - makes and validates a NetIDT from components
func NewNetID(servicerev, principal, host string, port int) (NetIDT, *c.Err) {
	trynid := NetIDT{
		ServiceRev: servicerev,
		Principal:  principal,
		Host:       host,
		Port:       fmt.Sprintf(":%d", port),
	}
	_, nerr := NetIDParse(trynid.String())
	if nerr != nil {
		return NetIDT{}, nerr
	}
	return trynid, nil
}

// NetIDParse - parses the netid string into component NetIDT parts
func NetIDParse(nid string) (NetIDT, *c.Err) {
	netid := NetIDT{}
	if nid == "" {
		return netid, c.ErrF("empty nid")
	}
	halvsies := strings.Split(nid, "/net/")
	if len(halvsies) != 2 {
		return netid, c.ErrF("netID failed to parse out 2 '/net/' delimited parts")
	}
	servicenames := halvsies[0]
	// remove leading /
	if servicenames[0:1] == "/" {
		servicenames = servicenames[1:]
	}
	services := strings.Split(servicenames, "/")
	if len(services) != 2 {
		return netid, c.ErrF("netID failed to parse out leading /servicerev/principal fields")
	}
	var query bool
	if services[0] == "*" || len(services[0]) == 0 {
		query = true
	}
	if services[1] == "*" || len(services[1]) == 0 {
		query = true
	}
	// Split out the Host and Port part of the address
	addy := halvsies[1]
	// Allow address to be a wildcard query, skip parsing out host/address
	if addy == "*" || len(addy) == 0 {
		netid.ServiceRev = services[0]
		netid.Principal = services[1]
		netid.Query = true
		return netid, nil
	}
	// No it is supposed to be a resolvable address, see if there is a real port number...
	lastcolon := strings.LastIndex(addy, ":") // Last : delimits the port number
	if lastcolon < 0 {
		return netid, c.ErrF("netID failed to find : delimited port string at the end of %s", addy)
	}
	port := addy[lastcolon:]         // maintains the :
	_, err := strconv.Atoi(port[1:]) // Is what trails : an integer?
	if err != nil {
		return netid, c.ErrF("netID port is not a number: %s", port)
	}
	host := addy[:lastcolon]
	if len(host) < 1 {
		return netid, c.ErrF("netID parsed IP Host '%s' too short to be resolvable", host)
	}
	// All good.
	netid.ServiceRev = services[0]
	if !goodID(netid.ServiceRev) {
		return netid, c.ErrF(badID, netid.ServiceRev)
	}
	netid.Principal = services[1]
	netid.Host = host
	netid.Port = port
	netid.Query = query
	return netid, nil
}

// NodeIDT - The nodeID
type NodeIDT struct {
	BlocName    string `json:"blocname"`
	HordeName   string `json:"hordename"`
	NodeName    string `json:"nodename"`
	ServiceName string `json:"servicename"`
	ServiceAPI  string `json:"serviceapi"`
}

// String - returns the nodeID in string form
func (n *NodeIDT) String() string {
	return fmt.Sprintf("/%s/%s/%s/%s/%s", n.BlocName, n.HordeName, n.NodeName, n.ServiceName, n.ServiceAPI)
}

// KeyIDT - the keyID broken into fields
type KeyIDT struct {
	ServiceRev  string `json:"servicerev"`
	Principal   string `json:"principal"`
	Fingerprint string `json:"fingerprint"`
}

// String - returns the keyID in string form
func (k *KeyIDT) String() string {
	return fmt.Sprintf("/%s/%s/keys/%s", k.ServiceRev, k.Principal, k.Fingerprint)
}

// NewKeyID - makes and validates a KeyIDT from components
func NewKeyID(servicerev, principal, fingerprint string) (KeyIDT, *c.Err) {
	trykid := KeyIDT{
		ServiceRev:  servicerev,
		Principal:   principal,
		Fingerprint: fingerprint,
	}
	_, kerr := KeyIDParse(trykid.String())
	if kerr != nil {
		return KeyIDT{}, kerr
	}
	return trykid, nil
}

// NewNodeID - makes and validates a NodeIDT from components
func NewNodeID(blocname, hordename, nodename, servicename, serviceapi string) (NodeIDT, *c.Err) {
	tryfid := NodeIDT{
		BlocName:    blocname,
		HordeName:   hordename,
		NodeName:    nodename,
		ServiceName: servicename,
		ServiceAPI:  serviceapi,
	}
	_, ferr := NodeIDParse(tryfid.String())
	if ferr != nil {
		return NodeIDT{}, ferr
	}
	return tryfid, nil
}

// NodeIDParse - parses the nodeid string into component NodeIDT parts
func NodeIDParse(fid string) (NodeIDT, *c.Err) {
	nodeid := NodeIDT{}
	if fid == "" {
		return nodeid, c.ErrF("empty nodeID input")
	}
	// remove leading /
	if fid[0:1] == "/" {
		fid = fid[1:]
	}
	parts := strings.Split(fid, "/")
	if len(parts) != 5 {
		return nodeid, c.ErrF("NodeIDParse fail -  delimiter count mismatch")
	}
	nodeid.BlocName = parts[0]
	nodeid.HordeName = parts[1]
	nodeid.NodeName = parts[2]
	nodeid.ServiceName = parts[3]
	if !goodID(nodeid.ServiceName) {
		return nodeid, c.ErrF(badID, nodeid.ServiceName)
	}
	nodeid.ServiceAPI = parts[4]
	if !goodID(nodeid.ServiceAPI) {
		return nodeid, c.ErrF(badID, nodeid.ServiceAPI)
	}
	return nodeid, nil
}

// KeyIDParse - parses the keyid string into component KeyIDT parts
func KeyIDParse(kid string) (KeyIDT, *c.Err) {
	keyid := KeyIDT{}
	if kid == "" {
		return keyid, c.ErrF("empty keyID input")
	}
	halvsies := strings.Split(kid, "/keys/")
	if len(halvsies) != 2 {
		return keyid, c.ErrF("keyID failed to parse out 2 '/keys/' delimited parts")
	}
	servicenames := halvsies[0]
	// remove leading /
	if servicenames[0:1] == "/" {
		servicenames = servicenames[1:]
	}
	services := strings.Split(servicenames, "/")
	if len(services) != 2 {
		return keyid, c.ErrF("keyID failed to parse out leading /servicerev/userid fields")
	}
	// Split out the Fingerprint part
	fingerprint := halvsies[1]
	if fingerprint == "*" || len(fingerprint) == 0 {
		// we are parsing a query string
		keyid.ServiceRev = services[0]
		keyid.Principal = services[1]
		return keyid, nil
	}
	// No it is supposed to be a resolvable fingerprint...
	// See if our last colon is in the right character position
	// e.g.: e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22
	lastcolon := strings.LastIndex(fingerprint, ":") // Last :
	if lastcolon != 44 {
		return keyid, c.ErrF("keyID failed to parse fingerprint : %s", fingerprint)
	}
	// All good.
	keyid.ServiceRev = services[0]
	if !goodID(keyid.ServiceRev) {
		return keyid, c.ErrF(badID, keyid.ServiceRev)
	}
	keyid.Principal = services[1]
	keyid.Fingerprint = fingerprint
	return keyid, nil
}

func goodID(id string) bool {
	if id == "self" {
		return true
	}
	r := []rune(id)
	if len(r) < 1 {
		return true
	}
	if !unicode.IsUpper(r[0]) {
		return false
	}
	for _, c := range r {
		ok := unicode.IsUpper(c) || unicode.IsLower(c) || unicode.IsNumber(c) || (c == '_')
		if !ok {
			return false
		}
	}
	return true
}
