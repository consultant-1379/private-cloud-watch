package main

import (
	"flag"
	"fmt"

	"os"
	"strings"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/pastiche"
)

var (
	portArg      = flag.String("port", "10000", "The TCP port to use")
	pasticheDirs = flag.String("blob_dirs", "", "Comma separated Directories for blob storage")
	pServers     = flag.String("servers", "", "Comma separated list of other pastiche servers addr:port. For testing only.")
)

// Start a pastich server with args from command line for basic testing.
func main() {
	flag.Parse()
	fmt.Printf("pastiche-testserver\n")
	fmt.Printf("Directories for this server at startup.: %v\n", *pasticheDirs)
	var dirs []string
	if *pasticheDirs != "" {
		dirs = strings.Split(*pasticheDirs, ",")
	}
	store, err := pastiche.NewServer(dirs)
	if err != nil {
		clog.Log.Fatal(nil, "failed to create blob store : %s", err)
		os.Exit(1)
	}

	otherServers := strings.Split(*pServers, ",")
	clog.Log.Log(nil, "remote servers: %v", otherServers)
	store.AddOtherServers(otherServers) // TODO: find out about other servers via strew

	fmt.Printf("Server starting on port %s\n", *portArg)
	err = store.Start(*portArg)
	if err != nil {
		clog.Log.Fatal(nil, "Server error:  %s", err)
		os.Exit(1)

	}
}
