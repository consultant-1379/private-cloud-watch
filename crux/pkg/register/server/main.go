// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// test server for registry

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
	"github.com/erixzone/crux/pkg/register"
)

func main() {
	logserver := clog.Log.With("focus", "register_server")
	srvdocker := os.Getenv("DOCKER_REGISTRY_SERVER")
	if srvdocker == "" {
		srvdocker = "localhost"
	}

	dbname, ierr := reeve.AgentInit("", logserver)
	if ierr != nil {
		fmt.Fprintf(os.Stderr, "%s\nStack: %s\n", ierr.String(), ierr.Stack)
		os.Exit(1)
	}

	registryfid, err := idutils.NewNodeID("flock", "horde", srvdocker, register.RegistryName, register.RegistryAPI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	registryaddress := srvdocker + ":50060"
	stewardaddress := srvdocker + ":50061"
	testenckey := "27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf"
	reevetimeout := 10 * time.Second

	logreg := clog.Log.With("focus", register.RegistryRev)
	imp, cerr := grpcsig.InitDefaultService(dbname, reeve.ReeveRev, nil, logreg, 300, false)
	if cerr != nil {
		fmt.Fprintf(os.Stderr, "%s\nStack: %s\n", cerr.String(), cerr.Stack)
		os.Exit(1)
	}
	ptrimp := &imp

	rerr := register.RegistryInit(registryfid, registryaddress, stewardaddress, testenckey, reevetimeout, &ptrimp)
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", rerr)
		os.Exit(1)
	}
	// wait it out
	_, ts := grpcsig.GetPidTS()
	logserver.Log("SEV", "INFO", "MSG", "registry running 15 more seconds ", "TS", ts)
	time.Sleep(15 * time.Second)
}
