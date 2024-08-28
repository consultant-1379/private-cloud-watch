// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package main

// Client to test the registry service

import (
	"fmt"
	"os"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/register"
	"github.com/erixzone/crux/pkg/rucklib"
)

func main() {
	// Ports we will use
	reevePort := 50059
	registryport := ":50060"

	srvdocker := "localhost"
	clidocker := os.Getenv("DOCKER_REG_CLIENT")
	if clidocker == "" {
		clidocker = "localhost"
	} else { // we are in myriad
		srvdocker = "registry"
	}
	testenckey := "27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf"
	regaddress := srvdocker + registryport

	// Start Reeve service, return an interface to its non-grpc services
	// for client grpcsig signing, and for server public key database lookups
	// which are local, pointer based structs that are passed via interface{}
	logreeve := clog.Log.With("focus", "reeveinit", "node", clidocker)
	logreg := clog.Log.With("focus", "register_client", "node", clidocker)
	logreeve.Log("node", clidocker, "Starting Reeve")

	// Get the reeveapi via interface

	var reeveapi rucklib.ReeveAPI
	rv, err := reeve.StartReeve("", "flock", "horde", clidocker, reevePort, "", nil, logreeve)
	if err != nil {
		fmt.Printf("Fatal -  reeve cannot start: %v", err)
		os.Exit(1)
	}
	reeveapiif := rv
	reeveapi = reeveapiif
	reevenodeid, reevenetid, _, reevepubkeyjson, reeveimp := reeveapi.ReeveCallBackInfo()
	if reeveimp == nil {
		logreg.Log("node", clidocker, "fatal", "Failed reeveapi.ReeveCallBackInfo")
		os.Exit(1)
	}

	// Make a client to talk to the Registry Server, holding our reeve
	// callback information
	logreg.Log("SEV", "INFO", "Making RegisterClient")

	var reg crux.RegisterClient
	pinginterval := 2 * time.Second
	contimeout := 300 * time.Second
	cbtimeout := 30 * time.Second
	registercli := register.NewClient(reevenodeid, reevenetid, pinginterval, contimeout, cbtimeout, reeveimp)
	if registercli == nil {
		logreg.Log("node", clidocker, "fatal", "Failed to get newRegisterClient")
		os.Exit(1)
	}
	reg = registercli // handle the interface

	// Now we can call the Register server and
	// invoke the Registration Method to register our
	// reeve with the flock, executing
	// the two-way exchange of public keys
	logreg.Log("SEV", "INFO", fmt.Sprintf("Registering our Reeve with: %s", regaddress))
	gerr := reg.AddAReeve(regaddress, testenckey, reevepubkeyjson)
	if gerr != nil {
		// Fatal if we cannot register
		logreg.Log("node", clidocker, "fatal", fmt.Sprintf("Could not AddAReeve() %v", gerr))
		os.Exit(1)
	}
}
