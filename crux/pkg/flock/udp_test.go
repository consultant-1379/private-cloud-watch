package flock

import (
	"net"
	"testing"
)

func TestParseNetworks(t *testing.T) {
	emptyNets := ","
	goodNets := "10.0.1.0/24,15.14.13.0/20"
	badNets := "not a network, 134.14.1.5"

	n, _ := parseNetworks(emptyNets)
	if len(n) > 0 {
		t.Error("Expected no network entries when passing empty network csv")
	}

	n, err := parseNetworks(goodNets)
	if err != nil {
		t.Error("Error when passing good network csv: ", err)
	}
	if len(n) < 2 {
		t.Error("Expected 2 network entries from good network csv, got ", len(n))
	}

	_, err = parseNetworks(badNets)
	if err == nil {
		t.Error("Expected an error when passing bad network csv, but didn't get it")
	}
}

func TestGetLocalNet(t *testing.T) {
	goodIP := net.ParseIP("127.0.0.1")
	badIP := net.ParseIP("8.8.8.8")

	_, err := getLocalNet(goodIP)
	if err != nil {
		t.Error("Couldn't find loopback IP using getLocalNet: ", err)
	}

	_, err = getLocalNet(badIP)
	if err == nil {
		t.Error("used getLocalNet to look for nonexistent IP, but no error")
	}
}

func TestPruneLocalAddrs(t *testing.T) {
	addrs := []string{"1.2.2.7", "127.0.0.1", "127.0.0.1", "127.0.0.1", "1.2.3.4", "127.0.0.1"}
	newAddrs := pruneLocalAddrs(addrs)
	if len(newAddrs) > 2 {
		t.Error("Expected that localhost IP would be pruned, but it wasn't")
	}
}

func TestMakeHostAddrs(t *testing.T) {
	_, n1, _ := net.ParseCIDR("10.0.2.16/30")
	_, n2, _ := net.ParseCIDR("10.1.1.1/30")
	addrs, _ := makeHostAddrs([]*net.IPNet{n1, n2})
	if len(addrs) != 4 {
		t.Error("Expected 4 addrs, got ", len(addrs))
	}

	_, n, _ := net.ParseCIDR("10.0.2.16/31")
	_, err := makeHostAddrs([]*net.IPNet{n})
	if err == nil {
		t.Error("No error even though empty addrs")
	}
}

func TestIncrementIP(t *testing.T) {
	ip := net.ParseIP("1.1.1.255")
	incrementIP(ip)
	if !ip.Equal(net.ParseIP("1.1.2.0")) {
		t.Error("incrementIP didn't roll over correctly, got result: ", ip.String())
	}
}
