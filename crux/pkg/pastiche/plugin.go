package pastiche

import (
	"fmt"
	"strings"
	"time"

	"github.com/erixzone/crux/pkg/crux"
)

// PluginFunc - Function to be called by pastiche plugin code. Really
// just a server main loop with some stuff sent to the caller via the
// channel arguments.  You could use this as a normal non-plugin
// function.  To use as a plugin, create a simple wrapper program
// that's compiled as a go plugin.
func PluginFunc(quit chan bool, errch chan *crux.Err, period time.Duration, svcch chan []crux.Fservice) {
	xx := period * 3 / 4
	heart := time.Tick(xx)
	//dummy := crux.ErrF("Pastiche service")

	fmt.Printf("## ====== PluginFunc called ====\n")

	// TODO: Make configurable after manifest / config code works.
	// config works.
	pasticheDirs := "/tmp"
	portArg := "10000"

	fmt.Printf("## Directories for this server at startup.: %v\n", pasticheDirs)
	var dirs []string
	if pasticheDirs != "" {
		dirs = strings.Split(pasticheDirs, ",")
	}
	store, err := NewServerNoload(dirs)
	if err != nil {
		crux.Fatalf("failed to create blob store : %s", err)
	}

	// TODO: After Reeve/Steward are working, need to lookup other
	// Pastiche's in the cluster if constructor doesn't.
	// Periodically re-check for peers.

	//		otherServers := strings.Split(*pServers, ",")
	//		clog.Log.Log(nil, "remote servers: %v", otherServers)
	//		store.AddOtherServers(otherServers) // TODO: find out about other servers via strew
	//

	fmt.Printf("## Server starting on port %s\n", portArg)

	go store.Start(portArg)

	/*
		               err = store.Start(portArg)
		if err != nil {
				//clog.Log.Fatal(nil, "Server error:  %s", err)
				//TODO: send err up on channel to the picket proc that launched us.
				fmt.Printf("## ERROR starting store.")
				return
			}
	*/
	// Report up to picket what we are. Follow convention of
	// putting connection info in Image field.
	fs := crux.Fservice{UUID: "Pastiche-UUID", FuncName: "PastichePlugin", FileName: portArg}
	fsvec := []crux.Fservice{fs}
	fmt.Printf("## Issuing report: plugin func reporting service info on errch channel\n")
	svcch <- fsvec
	fmt.Printf("## Report accepted.\n")
	// Now handle channels for crux plugin stuff
loop:
	for {
		select {
		// This periodic  heartbeat is no guarantee that grpc functionality is working.
		case <-heart:
			svcch <- fsvec // FIXME: AFAIK, we're sending
			// the service registration
			// periodically in case
			// anything changes, and as a
			// heartbeat
		case <-quit:
			fmt.Printf("## Got QUIT on channel\n")
			qerr := crux.ErrF("Plugin exiting due to value on quit channel")
			errch <- qerr
			break loop
		}
	}
	svcch <- nil
}
