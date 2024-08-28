package main

import (
	"fmt"
	"os"
	"plugin"
)

func main() {

	fmt.Printf("Test Go plugin functionality.\n Note, that without a dl unload equivalent, you can't re-load a function of the same name.\n")
	args := os.Args[1:]
	if len(args) < 3 {
		fmt.Printf(" %s requires 3 arguments: pluginfile1  pluginfile2  common-function-name\n", os.Args[0])
		fmt.Printf("The provided function name will be looked up in both provided files.\n")
		fmt.Printf("Files must be go plugin binaries, compiled with -buildmode=plugin\n")
		os.Exit(1)
	}

	imageName := args[0]
	image2Name := args[1]
	funcName := args[2]
	fmt.Printf("Using function %s from file %s\n", funcName, imageName)

	if _, err := os.Stat(imageName); err == nil {
		TestPlugLoadStop(imageName, funcName)
		fmt.Printf("Done\n")

	} else {
		fmt.Printf("Plugin file %s doesn't exist, Do a \"make package\" to create.\n", imageName)
		os.Exit(1)
	}

	//  Can the plugin change, and be re-loaded into this running
	//  program?  In other words, can we load new versions as they
	//  become available, without stopping the executable or
	//  server.  It shouldn't be a problem, since functions are
	//  just references into specific binary images we load. No
	//  symbol table collisions in this executable.

	fmt.Printf("Looking for alternate plugin %s to test Open() and Lookup() on different version of plugin.\n", imageName)
	if _, err := os.Stat(image2Name); err == nil {
		TestPlugLoadStop(image2Name, funcName)
	} else {
		fmt.Printf("%s doesn't exist, not doing \"reload\" test.\n", imageName)
	}
}

// TestPlugLoadStop - Load, start, stop the plugin.  No actual
// operations or data movement performed.
func TestPlugLoadStop(imageName, funcName string) {
	fmt.Printf("Attempting to Open plugin file %s \n", imageName)
	image, err := plugin.Open(imageName)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Attempting to load plugin %s from file %s\n", funcName, imageName)
	funcRef, err2 := image.Lookup(funcName)
	if err2 != nil {
		fmt.Printf("error: %s\n", err2)
		os.Exit(1)
	}

	fmt.Printf(">>> Plugin open and lookup success, got funcref %+v.\n", funcRef)
	return
	// TODO: Drop this and use crux plugin signature when it's settled.
	//	type CruxPluginFunc func(quit chan bool, svcs chan *crux.Err, period time.Duration, ch chan []crux.Fservice)

	/*

	        // Throws a lint error but works
	        type PlugFuncSig = func(<-chan bool, chan<- []pb.HeartbeatReq,
		**crux.Confab, string, clog.Logger, idutils.NodeIDT, string,
		*reeve.StateT)


		typedFunc, ok := funcRef.(PlugFuncSig)
		if !ok {
			//	panic("Plugin didn't match")
			fmt.Print("Plugin didn't match\n")
		}
		//	var heartBeatCount int




		period := 50 * time.Millisecond
		donech := make(chan bool)
		errch := make(chan *crux.Err)
		fservch := make(chan []crux.Fservice)
		fmt.Printf(">>> Starting Plugin function %s from image %s.\n", funcName, imageName)
		go typedFunc(donech, errch, period, fservch)

		fmt.Printf(">>> Started Plugin function in goroutine.\n")
		// TODO:  Read service info from channel.  Stop plugin via channel.

			// wait for registration info and then quit.
				go func(errch chan *crux.Err, donech chan bool) {
					heartBeatCount = 0
					for {
						slice := <-fservch
						// TODO: display / test the returned Fservice info
						if slice == nil {
							donech <- true
							return
						}
						heartBeatCount++
					}
				}(errch, donech)

				// wait for a few heartbeats
				const beats = 10
				time.Sleep(beats * period)
				// tell it to quit
				donech <- true
				// wait til done
				<-errch
				//	c.Assert(heartBeatCount >= beats, Equals, true)
	*/
}
