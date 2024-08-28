package pastiche

import (
	"context"
	"fmt"
	pb "github.com/erixzone/crux/gen/cruxgen"
	rpb "github.com/erixzone/crux/gen/ruckgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"io"
	"math/rand"
	"os"
	"time"
)

const (
	// MaxBufSize is the maximum buffer size (in bytes) received in a read chunk or sent in a write chunk.
	MaxBufSize  = 2 * 1024 * 1024
	backoffBase = 10 * time.Millisecond
	backoffMax  = 1 * time.Second
	maxTries    = 5
)

// Client - This is the Client implementing the Pastiche API, for use
// by applications.  It uses the pastiche grpc internally.
type Client struct {
	grpc   pb.PasticheSrvClient
	signer *grpcsig.ClientSignerT
}

var _ API = (*Client)(nil)

var pSigner *grpcsig.ClientSignerT

// SetClientSigner - sets the internal grpcsig Signer in pastiche for streaming operations
func SetClientSigner(clisigner **grpcsig.ClientSignerT) *crux.Err {
	pSigner = *clisigner
	return nil
}

// NewClient - Create a pastiche API client given a pastiche grpc client.
func NewClient(pClient pb.PasticheSrvClient) *Client {
	return &Client{grpc: pClient, signer: pSigner}
}

// NewCruxClient - Create a pastiche API client given a pastiche grpc client.
func NewCruxClient(pClient pb.PasticheSrvClient, signer **grpcsig.ClientSignerT) (*Client, *crux.Err) {

	if signer == nil {
		return nil, crux.ErrF("NewCruxClient - no grpcsig.AgentSigner provided")
	}

	pSigner = *signer

	c := Client{grpc: pClient, signer: pSigner}
	return &c, nil
}

// DeleteAll - Remove all files from cache directories.  This may not
// affect files outside cache directories that were registered
func (c *Client) DeleteAll() error {
	pd := pb.DeleteAllRequest{}
	resp, err := c.grpc.DeleteAll(context.Background(), &pd)
	if err != nil {
		return crux.ErrE(err)
	}
	if resp.GetSuccess() == false {
		return crux.ErrF("delete failed")
	}

	return nil
}

// AddDirToCache - The server requires at least one directory on its
// local system to place files in created with AddData and potentially
// other calls.  If loadFiles is true, the directory will be scanned
// for existing files, and make them available via the cache.
func (c *Client) AddDirToCache(dirPath string, loadFiles bool) error {
	pd := pb.AddDirToCacheRequest{Path: dirPath, Scan: loadFiles}
	resp, err := c.grpc.AddDirToCache(context.Background(), &pd)
	if err != nil {
		return crux.ErrE(err)
	}
	if resp.GetSuccess() == false {
		return crux.ErrF("add failed")
	}
	return nil
}

// Delete - Remove entry from cache and blob from disk
func (c *Client) Delete(key string) error {
	//  Do we allow delete of permanent file, or just removal from
	// pastiche's lookup map?
	pd := pb.DeleteRequest{Hash: key}
	resp, err := c.grpc.Delete(context.Background(), &pd)
	if err != nil {
		return crux.ErrE(err)
	}
	if resp.GetSuccess() == false {
		return crux.ErrF("delete failed")
	}

	return nil
}

// SetReservation - Prevents the blob represented by key (a hash) from
// being evicted from the cache until the expiration reached.  The
// returned time value is the reservation expiration time.
func (c *Client) SetReservation(key string, reserve bool) (*time.Time, error) {

	pr := pb.ReserveRequest{Hash: key, Reserve: reserve}
	resp, err := c.grpc.Reserve(context.Background(), &pr)
	if err != nil {
		return nil, err
	}
	seconds := resp.GetTime()
	expireTime := time.Unix(seconds, 0)
	return &expireTime, nil
}

// GetPath - Given a the hash for a blob in pastiche's blob store,
// return the path on the local filesystem.  If localOnly is false,
// then any remote servers will also be searched.
func (c *Client) GetPath(localOnly bool, key string) (string, error) {

	pr := pb.PathRequest{Hash: key, LocalOnly: localOnly}
	pathResp, err := c.grpc.GetPath(context.Background(), &pr)
	// grpc system itself had an error
	if err != nil {
		clog.Log.Log(nil, "GetPath (grpc) returns err for key %s.  err:%+v", key, err)
		return "", err
	}

	// Did the pastiche server itself send an error?
	cerr := crux.Proto2Err(pathResp.Err)
	// FIXME: client methods should all return crux.Err
	if cerr != nil {
		errRet := fmt.Errorf(cerr.String())
		clog.Log.Log(nil, "GetPath (grpc) response has err for key %s.  err:%+v", key, errRet)
		return "", errRet
	}

	//FIXME: pathResp.Path can be either a local path, or remote, prefixed with server ip & port.
	return string(pathResp.Path), nil
}

// RegisterPermanentFile - For files you don't want to be
// evicted/deleted.  Bootstrap requirements, for example.
// - The file is effectively permanent by setting huge reservation
// expiration.
// - The file's directory is not added to the cache-dirs list,
// and so RemoveAllFiles() will not affect it.
// - The file should be looked up by its hash, but the filename is not affected
// - Space used counts towards cache space.
// - Note , the passed key is a hash, but it is not checked here against the file.
func (c *Client) RegisterPermanentFile(key string, fullPath string) error {
	pr := pb.RegisterFileRequest{Hash: key, Path: fullPath}
	regResp, err := c.grpc.RegisterFile(context.Background(), &pr)
	if err != nil {
		return crux.ErrE(err)
	}
	if regResp.GetSuccess() == false {
		return crux.ErrF("registration of %s failed", fullPath)
	}
	return nil
}

func (c *Client) getGrpcStream(key string) (io.ReadCloser, error) {
	ll := clog.Log.With("focus", "client-getGrpcStream")

	path, err := c.GetPath(false, key)
	if err != nil {
		return nil, err
	}
	// FIXME:  error if GetPath returns a local path (without server prefix)

	// 2. Get a reader for the remote server's grpc  stream
	uri := string(path)
	ll.Log(nil, "got remote node resp [%v], URI [%s]", path, uri)
	srv, err := GetServerName(uri)
	if err != nil {
		return nil, err
	}
	var err1 error

	var remoteClient *Client

	if c.signer == nil {
		var opts []grpc.DialOption
		var conn *grpc.ClientConn
		opts = grpcsig.PrometheusDialOpts(grpc.WithInsecure())
		conn, err1 = grpc.Dial(srv, opts...)
		if err1 != nil {
			ll.Log(nil, "failed on dial: %v", err1)
			return nil, err1
		}
		remoteClient = NewClient(pb.NewPasticheSrvClient(conn))

	} else {
		c, err1 := GetCruxGrpcClient(srv)
		if err1 != nil {
			ll.Log(nil, "Failed to get crux grpc client")
			return nil, err1
		}
		var client pb.PasticheSrvClient
		client = *c
		var errC *crux.Err
		remoteClient, errC = NewCruxClient(client, &pSigner)
		if errC != nil {
			ll.Log(nil, "Failed to wrap a pastiche grpc client %+v", errC)
		}
	}

	rdr, err2 := remoteClient.NewReader(context.Background(), key)
	if err2 != nil {
		ll.Log(nil, "Failed to get a reader for remote data")
		return nil, err2
	}
	return rdr, nil
}

// GetCruxGrpcClient - Non caching version.
func GetCruxGrpcClient(pstchServer string) (*pb.PasticheSrvClient, error) {

	rLog := clog.Log.With("focus", "client-getRemoteCruxClient")
	// For crux s, we load PstchServers with NetIDT stringifications
	netID, err := idutils.NetIDParse(pstchServer)
	if err != nil {
		rLog.Log(nil, "fail to get NetID from server string: %v", err)
		return nil, err
	}

	if pSigner == nil {
		msg := "pSigner not set in client."
		rLog.Log(nil, msg)
		return nil, fmt.Errorf(msg)
	}

	// FIXME: pSigner - "client signer interface sent is not a signer type"
	// Getting pSigner init wrong? Fails inside ConnectPasticheSrv, but same func works in direct call from bootstra[
	newClient, err := rpb.ConnectPasticheSrv(netID, &pSigner, rLog)
	if err != nil {
		rLog.Log("fail to Connect as pastiche client err: %+v", err)
		return nil, err
	}
	return &newClient, nil
}

// AddDataFromRemote - Transfer a blob from a remote blobstore to the
// local one. Return the destination local path.
func (c *Client) AddDataFromRemote(key string) (string, error) {
	//Note: The client is acting as an intermediary between the
	//remote and local server.  This should probably be made to
	//work like AddTarFromRemote() where the local server gets the
	//datastream itself

	rdr, err := c.getGrpcStream(key)
	if err != nil {
		return "", err
	}

	// AddData() puts on _local_ server
	path, err3 := c.AddData(key, rdr)
	if err3 != nil {
		return "", err
	}

	return path, nil

}

// AddFilesFromDir - client wrapper for grpc call of same name
func (c *Client) AddFilesFromDir(dirPath string) (uint64, error) {

	afr := pb.AddFilesFromDirRequest{Dirpath: dirPath}
	resp, err := c.grpc.AddFilesFromDir(context.Background(), &afr)
	if err != nil {
		return 0, err
	}
	return uint64(resp.Numfiles), nil
}

// AddTar - Explode git tarball and return path to the created sub
// directory.  tarPath is path to the tar file to be added
func (c *Client) AddTar(commitHash string, tarPath string) (string, error) {
	pa := pb.AddTarRequest{Hash: commitHash, Filename: tarPath}
	tarResp, err := c.grpc.AddTar(context.Background(), &pa)
	if err != nil {
		return "", err
	}
	return tarResp.GetPath(), nil
}

// AddTarFromRemote - find the key on a remote server, and transfer
// it's entire directory hierarchy to the local server of this client.
func (c *Client) AddTarFromRemote(key string) (string, error) {
	at := pb.AddTarFromRemoteRequest{Hash: key}
	tarResp, err := c.grpc.AddTarFromRemote(context.Background(), &at)
	if err != nil {
		return "", err
	}

	return tarResp.Path, nil
}

// AddDataFromFile - Copy the file's content to a new file, who's name is the
//  key.  Sugar to relieve user of creating/closing a file reader.
//  the filename.
func (c *Client) AddDataFromFile(key string, fileName string) (string, error) {

	rdr, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer rdr.Close()
	// Note: From this point on, this data is only accessible via
	// its hash.  Filename has been discarded
	return c.AddData(key, rdr)
}

// AddData - send the contents of rdr to gRPC AddData stream.
// Satisfies the Pastiche API. Shields the user from gRPC data stream
// mechanics.
func (c *Client) AddData(key string, dataRdr io.Reader) (string, error) {

	streamClient, err := c.grpc.AddDataStream(context.Background())
	if err != nil {
		return "", crux.ErrF("error with gRPC AddData():  %s", err.Error())
	}

	BufBytes := 64 * 1024
	buf := make([]byte, BufBytes)

	for {
		n, err := dataRdr.Read(buf)
		ar := &pb.AddRequest{Hash: key}
		ar.Data = buf[:n]
		if err == io.EOF {
			//fmt.Printf("EOF reached,  %d final bytes\n", n)
			ar.LastData = true
			streamClient.Send(ar)
			break
		}

		if err != nil {
			return "", crux.ErrF("error reading buffer, n %d bytes,  %s", n, err)
		}

		//fmt.Printf("Sending, n %d bytes\n", n)

		streamClient.Send(ar)
	}

	err = streamClient.CloseSend() // Does stream.CloseSend internally
	if err != nil {
		return "", crux.ErrF(">>> Close Send Failed:  %v", err)
	}

	adResp := &pb.AddResponse{}
	err = streamClient.RecvMsg(adResp) // Does stream.CloseSend internally
	if err != nil {
		return "", crux.ErrF(">>> RecvMesg Failed:  %v", err)
	}
	return adResp.Path, nil
}

// Reader code is google bytestream code with minor changes to adapt
// to our different client layout:
// google/google-api-go-client/blob/master/transport/bytestream/client.go
type Reader struct {
	ctx        context.Context
	wc         *Client
	readClient pb.PasticheSrv_GetDataStreamClient
	hash       string
	err        error
	buf        []byte
}

// NewReader - Creates creates a incoming grpc bytestream and wraps it
// in a returned reader object.
func (c *Client) NewReader(ctx context.Context, key string) (*Reader, error) {
	getReq := &pb.GetRequest{Hash: key}
	streamClient, err := c.grpc.GetDataStream(context.Background(), getReq)
	if err != nil {
		return nil, crux.ErrF("error with gRPC GetData():  %s", err.Error())
	}

	return &Reader{ctx: ctx, wc: c, readClient: streamClient, hash: key}, nil
}

// Read implements io.Reader.
// Read buffers received bytes that do not fit in p.
// lifted from bytestream/client.go
func (r *Reader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	var backoffDelay time.Duration
	for tries := 0; len(r.buf) == 0 && tries < maxTries; tries++ {
		// No data in buffer.
		resp, err := r.readClient.Recv()
		if err != nil {
			r.err = err
			return 0, err
		}
		r.buf = resp.Data
		if len(r.buf) != 0 {
			break
		}

		// back off
		if backoffDelay < backoffBase {
			backoffDelay = backoffBase
		} else {
			backoffDelay = time.Duration(float64(backoffDelay) * 1.3 * (1 - 0.4*rand.Float64()))
		}
		if backoffDelay > backoffMax {
			backoffDelay = backoffMax
		}
		select {
		case <-time.After(backoffDelay):
		case <-r.ctx.Done():
			if err := r.ctx.Err(); err != nil {
				r.err = err

				return 0, r.err
			}
		}
	}

	// Copy from buffer.
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil

}

// Close implements io.Closer.
func (r *Reader) Close() error {
	if r.readClient == nil {
		return nil
	}
	err := r.readClient.CloseSend()
	r.readClient = nil
	return err
}

// ProbeServer - use PingTest so we can see/debug grpcsig errors
func ProbeServer(client pb.PasticheSrvClient) *crux.Err {
	localClient := NewClient(client)
	// Make a ping
	ping := &pb.Ping{Value: pb.Pingu_PING}
	_, err := localClient.grpc.PingTest(context.Background(), ping)
	if err != nil {
		return crux.ErrF("pastiche not repsonding: %v", err)
	}
	return nil
}

// ProbeServer0 - See if the Pastiche server a client has dialed for is
// really there.
func ProbeServer0(client pb.PasticheSrvClient) error {
	// We'll do a GetPath lookup to check server liveness
	localClient := NewClient(client)
	fmt.Printf("\nTesting server connection \n")
	lookupHash := "" // Get error by asking for a empty path
	_, err := localClient.GetPath(true, lookupHash)
	if err == nil {
		fmt.Printf("How did GetPath not error with path of (%s)?\n", lookupHash)
		return nil
	}
	code := grpc.Code(err)
	if code == codes.Unavailable {
		// There may be other reasons the server could
		// fail, but at this point we only care if
		// it's there or not
		return crux.ErrF("server not available. Not running, wrong port, or port already bound ")
	}
	return nil
}
