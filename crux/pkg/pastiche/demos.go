package pastiche

import (
	"fmt"
	"io"

	pb "github.com/erixzone/crux/gen/cruxgen"
)

// DemoLocal - ripstop test function
func DemoLocal(pClient pb.PasticheSrvClient, fakeHash string, dataRdr io.Reader) {

	// Add data to blob store
	localClient := NewClient(pClient)
	fmt.Printf("\nDemo >>>>>>>>>  Trying AddData   =======\n")
	path, err := localClient.AddData(fakeHash, dataRdr)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: AddData:  %s\n", err.Error())
		return
	}
	fmt.Printf(" %s added at path %s\n", fakeHash, path)
	fmt.Printf("Demo >> SUCCESS:  AddData \n")

	fmt.Printf("\nDemo >>>>>>>>>  Trying GetPath on Added Data   =======\n")
	lookupHash := fakeHash // Look for what we just added.
	// Get path back from server

	bPath, err := localClient.GetPath(false, lookupHash)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: GetPath:  %s\n", err.Error())
		return
	}
	fmt.Printf("Demo >> SUCCESS: Found %s  Path is: %v \n", lookupHash, bPath)
	fmt.Printf("\nDemo >>>>>>>>>  Trying GetPath on Non-existant Data, local and any remotes   =======\n")
	// Now see remote servers queried for hash
	missingHash := "<Hash-not-in-pastiche>"
	bPath, err = localClient.GetPath(false, missingHash)
	if err != nil {
		fmt.Printf("Demo >> SUCCESS: Error on non-existent path:  Error with gRPC GetPath():  %s\n", err.Error())
	} else {
		fmt.Printf("Demo >> FAILURE: GetPath: didn not throw an error on non-existent path\n")
	}
	if bPath == "" {
		fmt.Printf("Demo >>  GetPath path is empty string\n")
	}
	return
}

// DemoRemote - ripstop demo code
func DemoRemote(pClient, pClientRemote pb.PasticheSrvClient, fakeHash string, dataRdr io.Reader) {

	fmt.Printf("\nDemo >>>>>>> REMOTE:  Trying GetPath with data on remote node  =======\n")
	defer fmt.Printf("\n>>>>>>>>>> Demo REMOTE: DONE <<<<<<<<<<<< ")

	// Add data to remote server so "local" server can look it up.
	localClient := NewClient(pClient)
	remoteClient := NewClient(pClientRemote)
	path, err := remoteClient.AddData(fakeHash, dataRdr)
	if err != nil {
		fmt.Printf("Demo >>> AddData for remote server Failed!  %v", err)
		return
	}
	fmt.Printf(" %s added at path %s\n", fakeHash, path)
	// Local server finds the hash on the remote server
	bPath, err := localClient.GetPath(false, fakeHash)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: Error with gRPC GetPath():  %s\n", err.Error())
		return
	}
	fmt.Printf("Demo >> SUCCESS: Found data on remote server.  Path is: %v \n", bPath)

	// Get data from the remote server
	// Find data on a remote blob store server and add to the local blob store
	fmt.Printf("\nDemo >>>>>>>>>  Trying AddDataFromRemote   =======\n")

	path, err = localClient.AddDataFromRemote(fakeHash)

	// TODO:  remote url provided, or just use GetPath to lookup right path and use it.
	if err != nil {
		fmt.Printf("Demo >>> AddDataFromRemote Failed!  %v\n", err)
		return
	}
	fmt.Printf(" %s added at path %s\n", fakeHash, path)
	fmt.Printf("Demo >> SUCCESS: Retrieved data from remote server\n")

}
