package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/pastiche"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containning the CA root cert file")
	localServerAddr    = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
	remoteServerAddr   = flag.String("remote_addr", "", "A second server to use a remote")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
)

func demoRemote(pClient, pClientRemote pb.PasticheSrvClient, fakeHash string, dataRdr io.Reader) {

	fmt.Printf("\nDemo >>>>>>> REMOTE:  Trying GetPath with data on remote node <%s>  =======\n", *remoteServerAddr)
	defer fmt.Printf("\n>>>>>>>>>> Demo REMOTE: DONE <<<<<<<<<<<< ")

	// Add data to remote server so "local" server can look it up.
	localClient := pastiche.NewClient(pClient)
	remoteClient := pastiche.NewClient(pClientRemote)
	path, err := remoteClient.AddData(fakeHash, dataRdr)
	pz(1)
	if err != nil {
		fmt.Printf("Demo >>> AddData for remote server Failed!  %v", err)
		return
	}
	fmt.Printf(" %s added at path %s\n", fakeHash, path)
	// Local server finds the hash on the remote server
	bPath, err := localClient.GetPath(false, fakeHash)
	pz(1)
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
	pz(1)
	if err != nil {
		fmt.Printf("Demo >>> AddDataFromRemote Failed!  %v\n", err)
		return
	}
	fmt.Printf(" %s added at path %s\n", fakeHash, path)
	fmt.Printf("Demo >> SUCCESS: Retrieved data from remote server\n")

}

func pz(n int) {
	time.Sleep(time.Duration(n) * time.Second)
}

func demoLocal(pClient pb.PasticheSrvClient, fakeHash string, dataRdr io.Reader) {

	// Add data to blob store
	localClient := pastiche.NewClient(pClient)
	fmt.Printf("\nDemo >>>>>>>>>  Trying AddData   =======\n")
	path, err := localClient.AddData(fakeHash, dataRdr)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: AddData:  %s\n", err.Error())
		return
	}
	pz(1)
	fmt.Printf(" %s added at path %s\n", fakeHash, path)
	fmt.Printf("Demo >> SUCCESS:  AddData \n")

	fmt.Printf("\nDemo >>>>>>>>>  Trying GetPath on Added Data   =======\n")
	lookupHash := fakeHash // Look for what we just added.
	// Get path back from server
	pz(1)
	bPath, err := localClient.GetPath(false, lookupHash)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: GetPath:  %s\n", err.Error())
		return
	}
	fmt.Printf("Demo >> SUCCESS: Found %s  Path is: %v \n", lookupHash, bPath)
	pz(1)
	fmt.Printf("\nDemo >>>>>>>>>  Trying GetPath on Non-existant Data, local and any remotes   =======\n")
	// Now see remote servers queried for hash
	missingHash := "<Hash-not-in-pastiche>"
	bPath, err = localClient.GetPath(false, missingHash)
	pz(2)
	if err == nil {
		fmt.Printf("Demo >> FAILURE: GetPath returned without error for non-existent path.:  %s located at path <%s>\n", missingHash, bPath)
	} else {
		fmt.Printf("Demo >> SUCCESS: Error on non-existent path:  Error with gRPC GetPath():  %s\n", err.Error())
		return
	}
}

// demoLocalExtended - RegisterPermanentFile() , SetReservation(),  AddTar()
func demoLocalExtended(pClient pb.PasticheSrvClient, fakeHash string, dataRdr io.Reader) {
	fmt.Printf("\nDemo >>>>>>> Demo Extended Local functions  =======\n")
	defer fmt.Printf("\n>>>>>>>>>> Demo Extended Local DONE <<<<<<<<<<<< ")

	localClient := pastiche.NewClient(pClient)
	refTime := time.Now() // For reservation time comparisons

	// Test existing file registration
	dummyHash := "not-a-real-hash-329ughtbn"
	dFile := "pastiche-multiserver-test-file-empty"
	file, err := ioutil.TempFile(os.TempDir(), dFile)
	//	defer os.Remove(dFile)
	err = localClient.RegisterPermanentFile(dummyHash, file.Name())
	if err != nil {
		clog.Log.Fatal(nil, "failed  Registration: %v", err)
		return
	}

	// Add data to blob store for Reservation test
	_, err = localClient.AddData(fakeHash, dataRdr)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: AddData:  %s\n", err.Error())
		return
	}
	_, err = localClient.GetPath(false, fakeHash)
	if err != nil {
		clog.Log.Fatal(nil, "failed  getpath: %v", err)
		return
	}

	// Add a reservation that would keep it from being cache evicted.
	expire, err := localClient.SetReservation(fakeHash, true)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: SetReservation %s\n", err)
	}
	expDuration := expire.Sub(refTime)
	fmt.Printf("Time from now to expiration: %s\n", expDuration)
	if expDuration < (time.Minute * 5) {
		fmt.Printf("Demo >> FAILURE: SetReservation,  Reservation expiration time appears too short.\n")
	} else {
		fmt.Printf("Demo >> SUCCESS: SetReservation.\n")
	}

	// IsReserved() call is not in the API.
	newNow := time.Now()
	expire, err = localClient.SetReservation(fakeHash, false)
	fmt.Printf("Expiration time is %s, should be close to now %s \n", expire, time.Now())
	tt := time.Minute
	if expire.Truncate(tt) != newNow.Truncate(tt) {
		fmt.Printf("Demo >> FAILURE: SetReservation,  doesn't look like reservation was unset \n")
	} else {
		fmt.Printf("Demo >> SUCCESS: SetReservation, unset the reservation")
	}

	err = localClient.Delete(fakeHash)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: Delete\n")
	}

	_, err = localClient.GetPath(false, fakeHash)
	if err == nil {
		fmt.Printf("Demo >> FAILURE: Delete,  [%s] hash still exists\n", fakeHash)
		return
	}
	fmt.Printf("Demo >> SUCCESS: Delete,  [%s] hash removed\n", fakeHash)

	// AddTar - take file and expand it, verify GetPath() finds expansion path.
	fakeHash = "dummy-git-hash-3q41321098"
	tarPath := "../testdata/myrepo-prefix.tar"
	path, err := localClient.AddTar(fakeHash, tarPath)
	if err == nil {
		fmt.Printf("Demo >> FAILURE: AddTar(%s)\n", fakeHash)
		return
	}

	// Can we find the returned path in the cache?
	path2, err := localClient.GetPath(false, fakeHash)
	if err == nil {
		fmt.Printf("Demo >> FAILURE: GetPath() for %s after AddTar()\n", fakeHash)
		return
	}

	if path2 != path {
		fmt.Printf("Demo >> FAILURE: GetPath() for %s after AddTar() returns wrong path %s, expecting %s", fakeHash, path2, path)
		return
	}
	fmt.Printf("Demo >> SUCCESS: AddTar() expanded a tar and retreived path from cache")

}

// demoTar - show both local and remote Tar handling calls.
func demoTar(pClient, pClientRemote pb.PasticheSrvClient) {

	fmt.Printf("\nDemo >>>>>>> TAR API  =======\n")
	defer fmt.Printf("\n>>>>>>>>>> Demo TAR: DONE <<<<<<<<<<<< ")

	// Add data to remote server so "local" server can look it up.
	localClient := pastiche.NewClient(pClient)
	remoteClient := pastiche.NewClient(pClientRemote)

	fmt.Printf("\nDemo >>>>>>>>>  Trying AddFromRemoteTar  =======\n")

	// First add a tar in remote server
	tarKey := "123456789-addtar-test"
	tarFile := "./testdata/myrepo-prefix.tar"
	expandedTarDirPath, err := remoteClient.AddTar(tarKey, tarFile)
	if err != nil {
		fmt.Printf("Demo >>> AddFromRemoteTar Failed!  %v\n", err)
		return
	}

	fmt.Printf("Tar [%s]  expanded to directory [%s]\n", tarFile, expandedTarDirPath)

	// Then fetch it to the local server.
	path, err := localClient.AddTarFromRemote(tarKey)
	if err != nil {
		fmt.Printf("Demo >>> AddFromRemoteTar Failed!  %v\n", err)
		return
	}
	fmt.Printf(" %s added at path %s\n", tarKey, path)

	// show GetPath finds tar on local system
	path, err = localClient.GetPath(false, tarKey)
	if err != nil {
		fmt.Printf("Demo >> FAILURE: local GetPath for tar [%s] directory :  %s\n", tarKey, err.Error())
		return
	}

	fmt.Printf("Demo >> SUCCESS: Retrieved data from remote server\n")

}

// NewInsecureGrpcClient  - Sugar for creating a simple grpc client and verify server is there
func NewInsecureGrpcClient(addr string, opts []grpc.DialOption) (pb.PasticheSrvClient, error) {
	fmt.Printf("Demo - Creating connection for addr: %s\n", addr)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		clog.Log.Fatal(nil, "fail to dial: %v", err)
		return nil, err
	}

	client := pb.NewPasticheSrvClient(conn)

	// Getting a connection does not imply there's a server on the other end
	// Do a quick liveness test.
	if err := pastiche.ProbeServer(client); err != nil {
		return nil, errors.Wrap(err, addr)
	}

	return client, nil
}

func fail() {
	fmt.Printf("!!! This demo needs two pastiche servers running and the addresses passed in via the command line.  -server_addr,  -remote_addr\n")
	os.Exit(1)
}

// demo: Create a client and do some operations on pastiche blobstore server(s).
func main() {

	fmt.Printf("\n=== Starting Example client ===\n")
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*localServerAddr, opts...)
	if err != nil {
		clog.Log.Fatal(nil, "fail to dial: %v", err)
		fail()
	}
	defer conn.Close()

	localGrpcClient, err := NewInsecureGrpcClient(*localServerAddr, opts)
	if err != nil {
		clog.Log.Fatal(nil, fmt.Sprintf("failed to make grpc client for local server (err=%v)", err))
		fail()
	}

	//"remote" server is really just on another port for this demo.
	remoteGrpcClient, err := NewInsecureGrpcClient(*remoteServerAddr, opts)
	if err != nil {
		clog.Log.Fatal(nil, fmt.Sprintf("failed to make grpc client for remote server (err=%v)", err))
		fail()
	}

	//  Setup both servers with a storage directory, configured via environment variables.
	localDir := os.Getenv("LocalStore")
	remoteDir := os.Getenv("RemoteStore")
	fmt.Printf("local %s,   remote %s\n", localDir, remoteDir)
	if localDir == "" || remoteDir == "" {
		clog.Log.Fatal(nil, "Missing environment variable(s) LocalStore and/or RemoteStore.  These are the directories that the two servers will use for storing files.  They must be different directories.")
		os.Exit(1)
	}
	loadFiles := false
	localClient := pastiche.NewClient(localGrpcClient)
	err = localClient.AddDirToCache(localDir, loadFiles)
	if err != nil {
		clog.Log.Fatal(nil, "Couldn't add storage dir to server: ", localDir)
		os.Exit(1)
	}
	remoteClient := pastiche.NewClient(remoteGrpcClient)
	err = remoteClient.AddDirToCache(remoteDir, loadFiles)
	if err != nil {
		os.Exit(1)
		clog.Log.Fatal(nil, "Couldn't add storage dir to server: ", remoteDir)
	}

	demoTar(localGrpcClient, remoteGrpcClient)

	TestHash := "fake-crypto-hash-1"
	bulkdata := "once upon a time there was a streaming protocol"
	dataRdr := bytes.NewBufferString(bulkdata)
	demoLocal(localGrpcClient, TestHash, dataRdr)

	TestHash2 := "Test-crypto-hash-2-remote"
	bulkdata = "This file is to be placed first on the remote server, before being AddDataFromRemote()'d to local server"
	dataRdr = bytes.NewBufferString(bulkdata)
	demoRemote(localGrpcClient, remoteGrpcClient, TestHash2, dataRdr)
	/*
		TestHash3 := "Test-crypto-hash-3"
		bulkdata = "This file is placed by the extended local server test"
		dataRdr = bytes.NewBufferString(bulkdata)
		demoLocalExtended(localGrpcClient, TestHash3, dataRdr)
	*/
}
