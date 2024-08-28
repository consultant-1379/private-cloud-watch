package crux

import (
	"net"
	"time"
)

// DialTimeout is for all requests of some kind.
var DialTimeout = 4 * time.Second

// GetHostIPstring returns GetHostIP as a string
func GetHostIPstring() string {
	ip := GetHostIP()
	if ip == nil {
		return ""
	}
	return ip.String()
}

// GetHostIP returns our best guess at this system's IP address.
func GetHostIP() net.IP {
	// Attempt a UDP connection to a dummy IP, which will cause the local end
	// of the connection to be the interface with the default route
	addr := &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP
}
