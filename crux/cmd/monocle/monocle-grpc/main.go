package main

import (
	"flag"

	"os"

	"github.com/erixzone/crux/pkg/clog"
)

var (
	portArg = flag.String("port", "9090", "The TCP port to use")
)

// Monocle - A debug and monitoring tool for crux Other servers in the
// system will implement debug and/or monitoring grpc functions that
// Monocle can access.  The monocle CLI gives a command line interface
// to monocle, which acts as a proxy for collecting information from
// other parts of the system via these bespoke grpc accessible
// functions.

// Start a monocle grpc server.
func main() {
	flag.Parse()

	// TODO:: what func to instantiate new grpc server
	//   new swagger/rest server?
	// srv, err := cruxgen.NewServer() // Not using cruxgen for monocle (yet)
	srv, err := NewServer()
	if err != nil {
		clog.Log.Fatal(nil, "failed: %s", err)
		os.Exit(1)
	}

	err = srv.Start(*portArg)
	if err != nil {
		clog.Log.Fatal(nil, "Server error:  %s", err)
		os.Exit(1)
	}
}
