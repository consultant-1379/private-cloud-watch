package cidr

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// IPsubnet : IP subnet in CIDR notation
// IP may be nil, but size must be present
type IPsubnet struct {
	IP   net.IP
	Size int
}

func (net *IPsubnet) String() string {
	if net.IP == nil {
		return fmt.Sprintf("/%d", net.Size)
	}
	return fmt.Sprintf("%s/%d", net.IP, net.Size)
}

// Match : do subnets match?
func (net *IPsubnet) Match(other *IPsubnet) bool {
	return net.Size == other.Size &&
		(net.IP == nil || other.IP == nil || net.IP.Equal(other.IP))
}

// Subnet : convert string to IP subnet in CIDR notation
func Subnet(subnet string) (*IPsubnet, error) {
	bailout := func(msg string) (*IPsubnet, error) {
		return nil, fmt.Errorf("Invalid subnet spec: %s", msg)
	}
	pieces := strings.Split(subnet, "/")
	if len(pieces) != 2 {
		return bailout("error on Split")
	}
	net := net.ParseIP(pieces[0])
	if net == nil && len(pieces[0]) > 0 {
		return bailout("invalid network address")
	}
	size, err := strconv.Atoi(pieces[1])
	if err != nil {
		return bailout("invalid network size")
	}
	if size < 0 || size > 32 {
		return bailout("network size out of range")
	}
	return &IPsubnet{net, size}, nil
}
