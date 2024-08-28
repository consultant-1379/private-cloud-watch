package ruck

import (
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/register"
	"time"
)

// Creates a client for the initial registration of reeve to the centralized register system
// using the reverse-grpc-http-signatures call back system. here the  register callback timeout
// arguments are almost surfaced - can be further elevated into user space
func newRegisterClient(reevenodeid string, reevenetid string, imp **grpcsig.ImplementationT) *register.ClientT {
	pinginterval := 2 * time.Second
	contimeout := 300 * time.Second
	cbtimeout := 30 * time.Second
	return register.NewClient(reevenodeid, reevenetid, pinginterval, contimeout, cbtimeout, imp)
}
