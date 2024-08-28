package pastiche

import (
	//"context"

	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"

	context "golang.org/x/net/context" // built-in context not enough for grpc. (Until golang 1.9)
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	rpb "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/reeve"
	rl "github.com/erixzone/crux/pkg/rucklib"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Pastiche Server implements the interface in the .pb.go file.

// TODO: hook into horde for service registration.  Tests need to run
// w/o horde.
// TODO: use c.Err

// Server - implement gRPC interface for pastiche library.
type Server struct {
	mu                    *sync.Mutex // concurrent status map access
	status                map[string]*pb.AddResponse
	nextPossiblePeerCheck time.Time // when others needs to be updated
	store                 *BlobStore
	PeerClients           map[string]*pb.PasticheSrvClient
	Peers                 []string  // Other members of this Pastiche "cluster".  Access as client.
	addHandler            io.Writer // if we want a bytestream like iface w/ GetWriter(name ..) Close(name)
	BufSize               int
	// Crux Auth / Signature stuff
	useSignatures bool
	SignatureAPI  rl.ReeveAPI

	// Crux Plugin stuff
	horde  string
	ipname string
	UUID   string // Informational
	//NOD    string
	NODt   idutils.NodeIDT
	signer **grpcsig.ClientSignerT // For talking to peer servers.
	doneQ  chan bool
	quit   <-chan bool
	log    clog.Logger
}

// DefaultBufSizeBytes  - Size of buffers used for grpc streaming.
const DefaultBufSizeBytes = 64 * 1024

// MaxPeerCheckInterval - don't refresh peer list more often than this
var MaxPeerCheckInterval = time.Duration(60 * time.Second)

// NewServerCrux - Include  crux specific infrastructure & Networking variables
// TODO: pass in stuff other than quit channel once we know what's needed.
func NewServerCrux(paths []string, reeveState *reeve.StateT, nod idutils.NodeIDT, useSigs bool) (*Server, error) {
	store, err := NewBlobStore(paths)
	if err != nil {
		return nil, err
	}
	status := make(map[string]*pb.AddResponse)
	others := make(map[string]*pb.PasticheSrvClient)
	then := time.Now().UTC()

	// Go through signer contortions.
	sign, err := reeveState.ClientSigner(PasticheRev)
	castSigner := sign

	l := clog.Log.With("name", PasticheName)
	return &Server{
		mu:                    &sync.Mutex{},
		status:                status,
		nextPossiblePeerCheck: then,
		PeerClients:           others,
		store:                 store,
		BufSize:               DefaultBufSizeBytes,
		doneQ:                 make(chan bool),
		useSignatures:         useSigs,
		SignatureAPI:          reeveState,
		NODt:                  nod,
		signer:                castSigner,
		log:                   l,
	}, nil
}

// NewServer - Return a grpc server with a pastiche blobstore serving the paths argument
func NewServer(paths []string) (*Server, error) {
	store, err := NewBlobStore(paths)
	if err != nil {
		return nil, err
	}
	status := make(map[string]*pb.AddResponse)
	others := make(map[string]*pb.PasticheSrvClient)
	then := time.Now().UTC()
	return &Server{mu: &sync.Mutex{}, status: status, nextPossiblePeerCheck: then, PeerClients: others, store: store, BufSize: DefaultBufSizeBytes, SignatureAPI: reeve.ReeveState}, nil
}

// NewCustomServer - Return a grpc server with a pastiche blobstore serving the paths argument
func NewCustomServer(paths []string, CacheSize uint, evictHeadroom uint, reservationDuration time.Duration, preload bool) (*Server, error) {
	store, err := NewCustomBlobStore(paths, CacheSize, evictHeadroom, reservationDuration, preload)
	if err != nil {
		return nil, err
	}
	status := make(map[string]*pb.AddResponse)
	others := make(map[string]*pb.PasticheSrvClient)
	then := time.Now().UTC()
	l := clog.Log.With("name", PasticheName)
	return &Server{mu: &sync.Mutex{}, status: status, nextPossiblePeerCheck: then, PeerClients: others, store: store, BufSize: DefaultBufSizeBytes, log: l, SignatureAPI: reeve.ReeveState}, nil
}

// NewServerNoload - don't preload any files or dirs in the passed paths.
func NewServerNoload(paths []string) (*Server, error) {
	store, err := NewCustomBlobStore(paths, defaultCacheMB, defaultEvictHeadroomMB, DefaultReservation, false)
	if err != nil {
		return nil, err
	}
	status := make(map[string]*pb.AddResponse)
	others := make(map[string]*pb.PasticheSrvClient)
	then := time.Now().UTC()
	l := clog.Log.With("name", PasticheName)
	return &Server{mu: &sync.Mutex{}, status: status, nextPossiblePeerCheck: then, PeerClients: others, store: store, BufSize: DefaultBufSizeBytes, log: l, SignatureAPI: reeve.ReeveState}, nil
}

// AddOtherServers - Add server(s) to the list of known pastiche
// servers, other than itself. This is the "pastiche cluster" that
// will be used for remote operations.
func (ps *Server) AddOtherServers(servList []string) {
	ps.Peers = append(ps.Peers, servList...)
	ps.log.Log(nil, "Adding Servers: %v", servList)

}

// GetDataStream - Reads from file or directory for the request and sends
// it to the stream.  Directories are sent as a tar format stream.
func (ps *Server) GetDataStream(request *pb.GetRequest, stream pb.PasticheSrv_GetDataStreamServer) (funcError error) {

	err1 := crux.ErrF("GetDataStream unexpected error")
	funcError = status.Errorf(codes.Unimplemented, "grpc error: %v  Library Error %v", funcError, err1)

	if ps.store == nil {
		return status.Errorf(codes.FailedPrecondition, "no underlying BlobStore for this server")
	}

	if request == nil {
		return status.Errorf(codes.InvalidArgument, "Read(readRequest == nil)")
	}
	if request.Hash == "" {
		return status.Errorf(codes.InvalidArgument, "readRequest: empty key in request")
	}

	keyPath, err := ps.store.GetPath(true /*local*/, request.Hash)
	if err != nil {
		return status.Errorf(codes.NotFound, "failed path lookup for key %q : %v", request.Hash, err)
	}

	fi, err := os.Stat(keyPath)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "failed stat for key's returned path, key=%q  path=%s: %v", request.Hash, keyPath, err)
	}

	// Handle directories differently.  Stream the data for all entries as a tar stream.
	// Note: As of April 2018, all directories are the result of an AddTar or AddTarFromRemote.
	if fi.IsDir() {
		entries, err := ioutil.ReadDir(keyPath)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "key=%q path=%s: %v", request.Hash, keyPath, err)
		}

		if len(entries) != 1 {
			return status.Errorf(codes.FailedPrecondition, "key=%q path=%s: Pastiche directories should contain one and only-one subdirectory ", request.Hash, keyPath)
		}

		// Send all files and directories as a tar formatted stream of bytes
		strm, err := NewTarStreamer(&ProtoStreamWriter{stream: stream, isTar: true})
		dirToTar := filepath.Join(keyPath, entries[0].Name())

		err = strm.SendDir(dirToTar)
		if err != nil {
			return status.Errorf(codes.Unknown, "tar stream error for (key=%q path=%s tarDir=%s: %v", request.Hash, keyPath, dirToTar, err)
		}

		return nil
	}

	// Read in data from file for named key.
	keyFile, err := os.Open(keyPath)
	if err != nil {
		return status.Errorf(codes.Unknown, "open failed for key=%q path=%s : %v", request.Hash, keyPath, err)
	}
	defer keyFile.Close()

	var buf []byte
	buf = make([]byte, 1024*1024) // 1M buffer
	var bytesSent int

	// They're asking for a blob, not a directory
	for {
		n, err := keyFile.Read(buf)
		if n > 0 {
			if err := stream.Send(&pb.GetResponse{Data: buf[:n]}); err != nil {
				return status.Errorf(grpc.Code(err), "Send(key=%q ): %v", request.Hash, grpc.ErrorDesc(err))
			}
		} else if err == nil {
			return status.Errorf(codes.Internal, "nil error on empty read: io.Reader contract violated")
		}

		bytesSent += n
		if err == io.EOF {
			break
		}

		if err != nil {
			return status.Errorf(codes.Unknown, "Read(key=%q : %v", request.Hash, err)
		}
	}
	return nil
}

// ProtoStreamWriter - A simple wrapper to put an io.Writer interface on ProtoBuf streams
type ProtoStreamWriter struct {
	stream pb.PasticheSrv_GetDataStreamServer
	isTar  bool
}

func (psw *ProtoStreamWriter) Write(buf []byte) (int, error) {
	if err := psw.stream.Send(&pb.GetResponse{Data: buf}); err != nil {
		return 0, err
	}
	return len(buf), nil
}

// PingTest -- Returns a PING for a PONG, PONG for a PING
// and a test grpc error code and message for any other value
func (ps *Server) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error")
}

// Quit for gRPC
func (ps *Server) Quit(ctx context.Context, in *pb.QuitReq) (*pb.QuitReply, error) {
	ps.log.Log(nil, "--->Pastiche quit %v", *in)
	ps.doneQ <- true
	return &pb.QuitReply{Message: ""}, nil
}

// AddDataStream - Accept data writes from clients and give to blob store .
//  errors should be grpc.rpcError type. .
func (ps *Server) AddDataStream(stream pb.PasticheSrv_AddDataStreamServer) (funcError error) {
	var blobWriter *os.File
	recvStarted := false
	var receivedSize uint64
	var key = "none"

	err := ps.store.Configured()
	if err != nil {
		return status.Errorf(codes.Internal, "Server not configured: %s", err.Error())
	}

	for {
		addReq, err := stream.Recv()
		ps.log.Log("focus", "REQUEST", nil, "%d bytes  %v", len(addReq.Data), addReq.Hash)
		if err == io.EOF {
			// io.EOF errors are a non-error for the caller.
			ps.log.Log("focus", "STREAMING", nil, "EOF for %s", addReq.Hash)
			return nil
		} else if err != nil {
			return status.Errorf(codes.Unknown, "stream.Recv() failed: %v", err)
		}

		if addReq.Hash == "" {
			ps.log.Log("focus", "STREAMING", "Missing Hash")
			return status.Errorf(codes.InvalidArgument, "Empty or missing Hash")
		}

		// Initialize if we're the first one trying to write for this key.
		// FIXME:  Disallow write if read currently ongoing.
		if !recvStarted {
			ps.mu.Lock()
			stat, found := ps.status[addReq.Hash]
			if found {
				ps.mu.Unlock()
				ps.log.Log("focus", "DUPLICATES", "Hash", addReq.Hash)
				return status.Errorf(codes.AlreadyExists, "Already a AddData in progress for Hash %v", addReq.Hash)
			}
			// Not found. New blob/hash. Add to status map.
			stat = &pb.AddResponse{ReceivedSize: 0}
			ps.status[addReq.Hash] = stat
			ps.mu.Unlock()

			ps.log.Log("focus", "STREAMING", nil, "Starting stream for hash %v", addReq.Hash)

			// Clean up on failure so someone else can try again.
			defer func() {
				ps.mu.Lock()
				delete(ps.status, addReq.Hash)
				ps.mu.Unlock()
			}()

			_, err := ps.store.cl.EvictIfRequired()
			if err != nil {
				return status.Errorf(codes.Internal, "Cache could not evict space %s", err.Error())
			}

			// Get a temp file for the blob store
			key = addReq.Hash
			blobWriter, err = ps.store.NewBlobTempFile(key)
			// TODO: What if one of N blobstore
			// dirs already has this key?  Do we
			// overwrite it, or return a "File
			// already exists?"  If overwrite,
			// then NewBLobWRiter needs to detect
			// existing files and pick the correct
			// final path so it gets overwritten.

			if err != nil {
				ps.log.Log(nil, "server failed getting blob writer %v,  err: %s", addReq.Hash, err)
				return status.Errorf(codes.Internal, "Blob store failed to get new writer for Hash %v", addReq.Hash)
			}

			recvStarted = true
		}

		// New data available
		if len(addReq.Data) != 0 {
			ps.log.Log("focus", "STREAMING", nil, "Recv'd %d bytes", len(addReq.Data))
			n, err := blobWriter.Write(addReq.Data)
			if err != nil {
				ps.log.Log(nil, "write failed %v", err)
				err := blobWriter.Close() // FIXME: delete in-progress file. Close will put it in cache.
				return status.Errorf(codes.Internal, "Blob store write failed for Hash %v: err %v", addReq.Hash, err)
			}
			// Write() guarantees an error if not all byte written.
			receivedSize += uint64(n)
		}

		if addReq.LastData {
			// TODO: check provided hash against hash computed on received data
			ps.log.Log("focus", "STREAMING", "Finished SendAndClose")

			// No data sent. No reason to allow zero size files
			if receivedSize == 0 {
				err := blobWriter.Close()
				err2 := ps.store.RemoveTempFile(blobWriter.Name())
				if err2 != nil {
					err = errors.WithMessage(err, err2.Error())
				}

				return status.Errorf(codes.FailedPrecondition, "AddData %s: 0 bytes written", addReq.Hash)
			}

			err := blobWriter.Close()
			if err != nil {
				err2 := ps.store.RemoveTempFile(blobWriter.Name())
				if err2 != nil {
					errors.WithMessage(err, err2.Error())
				}

				return status.Errorf(codes.Internal, "Blob store write failed for Hash %v: err %s", addReq.Hash, err)
			}

			finalPath, err := ps.store.AddBlobWithEntry(key, blobWriter.Name(), receivedSize)
			if err != nil {
				ps.log.Log(nil, "Couldn't add to cache structs after blob persisted %v", err)
				return status.Errorf(codes.Internal, "Blobstore couldn't add to cache structs after blob persisted %v: err %v", key, err)
			}

			r := &pb.AddResponse{Path: finalPath}
			if err := stream.SendAndClose(r); err != nil {
				return status.Errorf(codes.Internal, "stream.SendAndClose(%q, AddResponse{ %d }): %v", key, receivedSize, err)
			}
			// TODO: Confirm timeout done by grpc itself if stream stalls / sender dies?
			// Else, we have to timeout the recv loop ourselves

			return nil
		} //
	} //
}

// RegisterFile - Place the designated file (already on the server)
// into the pastiche lookup table by the provided hash.  File is not
// transferred or moved.
func (ps *Server) RegisterFile(ctx context.Context, dReq *pb.RegisterFileRequest) (*pb.RegisterFileResponse, error) {
	success := true
	err := ps.store.RegisterPermanentFile(dReq.GetHash(), dReq.GetPath())
	if err != nil {
		success = false
	}
	return &pb.RegisterFileResponse{Success: success}, status.Errorf(codes.Unknown, "RegisterFile Failed: %s", err)
}

// Reserve - Prevent a file from being cache evicted.  Return the
// expiration time of the reservation
func (ps *Server) Reserve(ctx context.Context, dReq *pb.ReserveRequest) (*pb.ReserveResponse, error) {
	expTime, err := ps.store.SetReservation(dReq.GetHash(), dReq.GetReserve())
	if err != nil {
		return &pb.ReserveResponse{Success: false, Time: 0}, err
	}
	return &pb.ReserveResponse{Success: true, Time: expTime.Unix()}, status.Errorf(codes.Unknown, "Reservel Failed: %s", err)
}

// Delete - Delete a single entry and it's data file.
func (ps *Server) Delete(ctx context.Context, dReq *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := ps.store.Delete(dReq.GetHash())
	if err != nil {
		return &pb.DeleteResponse{Success: false}, err
	}
	return &pb.DeleteResponse{Success: true}, status.Errorf(codes.Unknown, "Delete Failed: %s", err)
}

// DeleteAll - Clear all cache directories of all files.  Remove
// entries from cache lookup table
func (ps *Server) DeleteAll(ctx context.Context, dReq *pb.DeleteAllRequest) (*pb.DeleteAllResponse, error) {
	err := ps.store.DeleteAll()
	if err != nil {
		return &pb.DeleteAllResponse{Success: false}, err
	}
	return &pb.DeleteAllResponse{Success: true}, status.Errorf(codes.Unknown, "Delete All Failed: %s", err)
}

// AddFilesFromDir - seed pastiche with all files found in the directory argument
func (ps *Server) AddFilesFromDir(ctx context.Context, dReq *pb.AddFilesFromDirRequest) (*pb.AddFilesFromDirResponse, error) {
	numFiles, err := ps.store.AddFilesFromDir(dReq.GetDirpath())
	if err != nil {
		ps.log.Log(nil, "Server AddFilesFromDir error ", err)
		return &pb.AddFilesFromDirResponse{Success: false}, status.Errorf(codes.Unknown, "AddFilesFromDir Failed: %s", err)
	}
	return &pb.AddFilesFromDirResponse{Success: true, Numfiles: int64(numFiles)}, nil

}

// AddTar -Take the path of a local tar file, expand it, register it
// with the provided key  and return the path it was expanded
// to.
func (ps *Server) AddTar(ctx context.Context, dReq *pb.AddTarRequest) (*pb.AddTarResponse, error) {
	// Tar is already a local file. Just tell the blobstore code to handle it.
	path, err := ps.store.AddTar(dReq.GetHash(), dReq.GetFilename())
	if err != nil {
		ps.log.Log(nil, "Server AddTar ERROR", err)
		return &pb.AddTarResponse{}, status.Errorf(codes.Unknown, "AddTarFromRemote Failed: %s", err)
	}

	return &pb.AddTarResponse{Path: path}, nil
}

// AddTarFromRemote - Transfer a directory hierarchy from another
// server to this one.  Directories are transferred via tar protocol
// (and most likely were imported into Pastiche via AddTar)
func (ps *Server) AddTarFromRemote(ctx context.Context, tReq *pb.AddTarFromRemoteRequest) (*pb.AddTarFromRemoteResponse, error) {

	errResp := &pb.AddTarFromRemoteResponse{Success: false}

	ll := ps.log.With("focus", "TAR-server")
	ll.Log(nil, "hash %s ", tReq.Hash)
	// 1. GetPath to find a remote server
	paths, err := ps.GetPathRemote(ctx, tReq.Hash)
	if err != nil {
		return errResp, status.Errorf(codes.Unknown, "AddTarFromRemote Failed: %s", err)
	}

	if len(paths) > 1 {
		// TODO: We could change GetPathRemote to return on first
		// found valid path.  Depends on if we want to build
		// load balancing logic into the client or server.
		ll.Log(nil, "found multiple paths %v, for %s  but using first one", paths, tReq.Hash)
	}
	remoteURL := paths[0]
	srv, err := GetServerName(remoteURL)
	if err != nil {
		return errResp, status.Errorf(codes.Unknown, "AddTarFromRemote Failed: %s", err)
	}

	ll.Log(nil, "remoteURL %s  on server %s", remoteURL, srv)

	// Create connection & client
	var remoteClient *Client

	if !ps.useSignatures {
		// Pre-crux code
		opts := grpcsig.PrometheusDialOpts(grpc.WithInsecure())
		conn, err1 := grpc.Dial(srv, opts...)
		if err1 != nil {
			ll.Log(nil, "failed on dial: %v", err1)
			return errResp, status.Errorf(codes.Unknown, "AddTarFromRemote Failed: %s", err1)
		}
		remoteClient = NewClient(pb.NewPasticheSrvClient(conn))
	} else {
		c, err1 := ps.GetCruxGrpcClient(srv)
		if err1 != nil {
			return errResp, err1
		}
		var client pb.PasticheSrvClient
		client = *c
		remoteClient = NewClient(client)
	}

	// 3. Get a reader for remote tar to pass to this server's blobstore
	tarRdr, err := remoteClient.NewReader(context.Background(), tReq.Hash)
	if err != nil {
		return errResp, status.Errorf(codes.Unknown, "AddTarFromRemote Failed: %s", err)
	}

	isZipped := true // Always true for our tar streams.
	localPath, err := ps.store.AddTarReader(tReq.Hash, tarRdr, isZipped)
	if err != nil {
		return errResp, status.Errorf(codes.Unknown, "AddTarFromRemote Failed: %s", err)
	}

	return &pb.AddTarFromRemoteResponse{Success: true, Path: localPath}, nil
}

// AddDirToCache - AddData calls will place files in directories added via this call.
func (ps *Server) AddDirToCache(ctx context.Context, dReq *pb.AddDirToCacheRequest) (*pb.AddDirToCacheResponse, error) {
	err := ps.store.AddDirToCache(dReq.GetPath(), dReq.GetScan())
	if err != nil {
		ps.log.Log(nil, "AddDirToCache()  SERVER, ERROR adding path %s in blobstore call", dReq.GetPath())
		return &pb.AddDirToCacheResponse{Success: false}, status.Errorf(codes.Unknown, "AddDirToCache Failed: %s", err)
	}
	return &pb.AddDirToCacheResponse{Success: true}, err
}

// GetPath - Implements GetPath defined in .proto.  Return a path on
// local storage for the corresponding key or name in the request.
func (ps *Server) GetPath(ctx context.Context, pReq *pb.PathRequest) (*pb.PathResponse, error) {
	var err error
	var localPath string
	var paths []string
	localPath, err = ps.store.GetPath(pReq.LocalOnly, pReq.Hash)
	l := ps.log.With("focus", "GETPATH")
	l.Log(nil, "GetPath for hash %s", pReq.Hash)
	if err != nil {
		l.Log(nil, " ERROR for Hash/Key:%s,  err:%s", pReq.Hash, err)
		if pReq.LocalOnly {
			l.Log(nil, " local getpath error for Hash/Key:%s,  err:%s\n", pReq.Hash, err)
			return &pb.PathResponse{Path: "", Err: crux.Err2Proto(crux.ErrE(err))}, err
		}

		// Check remote(s) if no local choice found.
		paths, err = ps.GetPathRemote(ctx, pReq.Hash)
		if err != nil {
			l.Log(nil, "  GetPathRemote error for Hash/Key:%s,  err:%s\n", pReq.Hash, err)
			return &pb.PathResponse{Path: "", Err: crux.Err2Proto(crux.ErrE(err))}, err
		}
		if len(paths) > 1 {
			// TODO: This remote server could also return multiple paths. Update .proto
			l.Log(nil, "  GetPathRemote returns multiple paths. Using first one.  Hash/Key:%s\n", pReq.Hash)
		}
		if len(paths) > 0 {
			return &pb.PathResponse{Path: paths[0]}, nil
		}

		l.Log(nil, " Potential Error:  GetPathRemote returns no error and no path.    Hash/Key:%s", pReq.Hash)
		return &pb.PathResponse{Path: "", Err: crux.Err2Proto(crux.ErrE(err))}, err
	}

	pResp := &pb.PathResponse{Path: localPath}
	return pResp, nil
}

// Heartbeat -
// TODO:  Implement or remove. Superseded by PingTest
func (ps *Server) Heartbeat(ctx context.Context, pReq *pb.HeartbeatReq) (*pb.HeartbeatReply, error) {
	return nil, nil
}

// RefreshPeerlist - update the Peers peer list, no more than every X seconds.
func (ps *Server) RefreshPeerlist() error {
	log := ps.log.With("focus", "pastiche-refresh")

	if time.Now().UTC().After(ps.nextPossiblePeerCheck) {
		// Set future time after which we should refresh the
		// peer server list.  This is to catch nodes we hadn't
		// connected to before, and nodes with intentional
		// pastiche shutdowns (no new server started)

		//Failures of a existing server will be handled by the
		//generated grpc client code, which will automatically
		//connect to a new server after a restart, so this
		//code doesn't have to create new clients

		// TODO: populate NOD in Server at creation
		flockID := ps.NODt

		log.Log("Looking for pastiche servers AKA endpoints")
		epl, err := rl.AllEndpoints(flockID, PasticheRev, ps.SignatureAPI)
		if err != nil {
			return err
		}

		if epl == nil {
			return crux.ErrF("no endpoints found")
		}

		log.Log("AllEndpoints returned")
		var pasticheServers []string

		for _, q := range epl {
			log.Log("endpoint:  %+v", q)
			f, err := idutils.NodeIDParse(q.Nodeid)
			if err != nil {
				log.Fatal("nodeid(s) parse fail: %s", q.Nodeid, err.String())
				return err
			}
			if f.ServiceName == PasticheName {
				log.Log("Pastiche @ %s", q.Netid)
				// We only need the net id string to connect to a server
				pasticheServers = append(pasticheServers, q.Netid)
			}
		}

		// Replace existing peer list & reset next check time.
		ps.Peers = pasticheServers
		ps.nextPossiblePeerCheck = time.Now().UTC().Add(MaxPeerCheckInterval)
		return nil
	}

	log.Log("Not time to refresh yet")

	return nil
}

// GetCruxGrpcClient - Returns a grpc client with signature keys setup for
// a peer.  Either from the stored list or by creating a new one and
// storing it.  This could have scaling issues if the number of
// pastiche servers were large (100's) and many remote operations
// occur
func (ps *Server) GetCruxGrpcClient(peer string) (*pb.PasticheSrvClient, error) {

	rLog := ps.log.With("focus", "pastiche-get-grpc-client")
	// For crux s, we load Peers with NetIDT stringifications
	netID, err := idutils.NetIDParse(peer)
	if err != nil {
		rLog.Log(nil, "fail to get NetID from server string: %v", err)
		return nil, err
	}

	if fndClient, ok := ps.PeerClients[peer]; ok {
		return fndClient, nil
	}

	newClient, err := rpb.ConnectPasticheSrv(netID, ps.signer, rLog)
	if err != nil {
		rLog.Log("fail to Connect as pastiche client: %v", err)
		return nil, err
	}

	ps.PeerClients[peer] = &newClient
	return &newClient, nil
}

func (ps *Server) readyForPeerOperations() (int, error) {
	rLog := ps.log.With("focus", "remote-pastiche")
	if ps.useSignatures {
		if ps.signer == nil {
			msg := "No clientsigner configured for pastiche server"
			rLog.Error(msg)
			return 0, fmt.Errorf(msg)
		}
	}

	if ps.Peers == nil {
		// FIXME: remove once we have timer based lookup
		rLog.Log("Peer server list is nil. attempting Refresh")
		err := ps.RefreshPeerlist()
		if err != nil {
			return 0, fmt.Errorf("failed peer refresh %+v", err)
		}
		//return nil,fmt.Errorf("Peer server list is nil.")
		rLog.Log("Got Peer server list from reeve")
	}

	numPeers := len(ps.Peers)
	if numPeers == 0 {
		return 0, fmt.Errorf("no peers in peer list")
	}
	return numPeers, nil
}

// GetPathRemote - Search other servers for paths, prepending server into to paths returned.
func (ps *Server) GetPathRemote(ctx context.Context, hash string) ([]string, error) {

	var err error

	rLog := ps.log.With("focus", "remote-pastiche")
	rLog.Log(nil, "GetPathRemote for hash %s", hash)

	numPeers, err := ps.readyForPeerOperations()
	if err != nil {
		return nil, err
	}
	// Search each server sequentially for now.  Could do a
	// few-at-a-time to reduce latency, or use a local remote-path
	// cache, etc.
	var client pb.PasticheSrvClient
	if ps.useSignatures {
		for _, srv := range ps.Peers {
			rLog.Log(nil, "Trying server: %s", srv)

			c, err := ps.GetCruxGrpcClient(srv)
			if err != nil {
				return nil, err
			}
			client = *c

			rLog.Log(nil, "Trying client for server %s for hash %s", srv, hash)

			// We're looking on a remote server, but we don't
			// want the remote server to also look elsewhere
			pReq := pb.PathRequest{Hash: hash, LocalOnly: true}

			var pResp *pb.PathResponse
			pResp, err = client.GetPath(ctx, &pReq)
			if err == nil {
				// Build slice until GetPath returns slice
				paths := []string{pResp.Path}
				AddPrefix(paths, srv)
				rLog.Log(nil, "Prefixing found path(s), resulting in: %v", paths)
				return paths, nil
			}

		}
		rLog.Log(nil, "Hash %s not found, after trying %d servers", hash, numPeers)
		return nil, err
	}

	// pre-crux code.  No signatures needed.
	// TODO: remove or move to a new non-crux server struct.
	for _, srv := range ps.Peers {
		rLog.Log(nil, "Trying server: %s", srv)

		// Un-signed grpc.
		opts := grpcsig.PrometheusDialOpts(grpc.WithInsecure())
		conn, err := grpc.Dial(srv, opts...)
		client := pb.NewPasticheSrvClient(conn)

		rLog.Log(nil, "Trying server %s for hash %s", srv, hash)
		pReq := pb.PathRequest{Hash: hash, LocalOnly: true}
		var pResp *pb.PathResponse
		pResp, err = client.GetPath(ctx, &pReq)
		conn.Close()
		if err == nil {
			// Build slice until GetPath returns slice
			paths := []string{pResp.Path}
			AddPrefix(paths, srv)
			rLog.Log(nil, "Prefixing found path(s) to %v", paths)
			return paths, nil
		}
	}
	rLog.Log(nil, "Hash %s not found, after trying %d servers", hash, numPeers)
	return nil, err

}

// Start - The grpc server will start listening for requests.
func (ps *Server) Start(port string) error {
	port = ":" + port
	ps.log.Log(nil, "starting pastiche blob server on localhost, addr: 127.0.0.1%s\n", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		ps.log.Fatal(nil, "failed to listen: %v", err)
		return err
	}
	_ = grpcsig.CheckCertificate(ps.SignatureAPI.GetCertificate(), "pastiche server Start")
	s := grpcsig.NewTLSServer(ps.SignatureAPI.GetCertificate())
	pb.RegisterPasticheSrvServer(s, ps)
	grpc_prometheus.Register(s)
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {

		ps.log.Fatal(nil, "failed to serve: %v", err)
		return err
	}

	return nil
}
