package pastiche

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	pb "github.com/erixzone/crux/gen/cruxgen"
	. "gopkg.in/check.v1"

	context "golang.org/x/net/context" // built-in context not enough for grpc. (Until golang 1.9)
)

/*
TODO: Test
- multiple root dirs
- Space conflict if all reserved and space limit would be exceeded on new add  - NoSpace error
*/

// TestPastiche: Hook gocheck into "go test" runner
func TestPastiche(t *testing.T) { TestingT(t) }

type PasticheTest struct {
	whoKnows string
	bs       *BlobStore
}

func init() {
	fmt.Printf("=== Init\n")
	pt := PasticheTest{}

	var pasticheDir string
	var err error
	if pasticheDir, err = ioutil.TempDir("/tmp", "pstch-test"); err != nil {
		fmt.Printf("Error creating temp dir:%v\n", err)
		os.Exit(0)
	}
	fmt.Printf("pastiche test using temp dir: %s\n", pasticheDir)
	// hard to have small test cache with headroom algo and checker.
	pt.bs, err = NewCustomBlobStore([]string{pasticheDir}, 1, defaultEvictHeadroomMB, DefaultReservation, false) // Small blobstore, preloading off.
	if err != nil {
		fmt.Printf("pasiche.New(), Error creating pastiche instance:%v\n", err)
		os.Exit(0)
	}
	Suite(&pt)

}

func (p *PasticheTest) SetUpSuite(c *C) {

}

func (p *PasticheTest) TearDownSuite(c *C) {

}

// TestCacheFull - Verify that things get flushed (LRU) when cache fills up.
func (p *PasticheTest) TestCacheFull(c *C) {
	// Add file 1  1/3 cache size
	// Add file 2  1/3 cache size
	// Add File 3  1/2 cache size,  File2 should be flushed.after file3 has landed.

}

// TestRegistration - See that
// A. Don't get flushed,
// B. Eventually have their lease timeout and then get flushed if
// pressure still exists.
func (p *PasticheTest) TestRegistration(c *C) {

	dummyFile := "pastiche-dummy"
	// Need a file to register
	file, err := ioutil.TempFile(os.TempDir(), dummyFile)
	defer os.Remove(dummyFile)

	key := "not-a-real-hash-xasjeh"
	err = p.bs.RegisterPermanentFile(key, file.Name())
	c.Assert(err, IsNil)

	// Check we can look it up : GetPath()
	_, err = p.bs.GetPath(true, key)
	c.Assert(err, IsNil)

	// Check that it's in the LRU w/  a huge date
	cacheEntry := p.bs.cl.GetEntry(key, false)
	c.Assert(cacheEntry.IsRegistered(), Equals, true)
	c.Logf(">>>>>>>>>>  Reservation expiration time: %v\n", cacheEntry.ReservationExpiry)

	// Verify it fails if file doesn't exist
	err = p.bs.RegisterPermanentFile(key, "Non-existant-pv0tpang")
	c.Assert(err, NotNil)
}

func (p *PasticheTest) TestAddTar(c *C) {
	// git archive created with command like:
	//    git archive --format tar.gz -o ../branch2.tar.gz --prefix branch2 master

	// uncompressed first
	dummyKey1 := "123456789-addtar-test"
	repoPrefix := "testdata/myrepo-prefix.tar"
	finalPath, err := p.bs.AddTar(dummyKey1, repoPrefix)
	c.Assert(err, IsNil)
	c.Logf("Success: Final path for %s is %s\n", repoPrefix, finalPath)
	os.RemoveAll(finalPath)
	// Will still be in map, even though we just removed the dir.
	returnedPath, err2 := p.bs.GetPath(true, dummyKey1)
	c.Assert(err2, IsNil)
	c.Assert(returnedPath, Equals, finalPath)
}

// TestAddFilesFromDir
// Testing grpc server wrapper for blobstore AddFilesFromDir func.
func (p *PasticheTest) TestAddFilesFromDirGrpc(c *C) {
	// Instantiate a pastiche server
	// s:= server{}
	tmpDir, err := ioutil.TempDir("/tmp", "pstch-test")
	c.Assert(err, IsNil)

	seedDir, err := ioutil.TempDir("/tmp", "pstch-seeddir")
	c.Assert(err, IsNil)

	fileName := "seed-file"
	var data = []byte("DATA1 TEST12341345234523452345235423455 END-DATA")
	fPath := path.Join(seedDir, fileName)
	err = ioutil.WriteFile(fPath, data, 0644)
	c.Assert(err, IsNil)

	paths := []string{tmpDir}
	s, err := NewServer(paths)
	c.Assert(err, IsNil)

	// create a request structure
	req := &pb.AddFilesFromDirRequest{Dirpath: seedDir}

	// Call it
	resp, err := s.AddFilesFromDir(context.Background(), req)
	c.Assert(err, IsNil)
	c.Assert(resp.Numfiles, Equals, int64(1))
}

// TestAddFilesFromDir - load files from a directory.
func (p *PasticheTest) TestAddFilesFromDir(c *C) {
	seedDir, err := ioutil.TempDir("/tmp", "pstch-seeddir")
	c.Assert(err, IsNil)

	pasticheDir, err := ioutil.TempDir("/tmp", "pstch-storage-dir")
	c.Assert(err, IsNil)

	fileName := "seed-file"
	var data = []byte("DATA1 TEST12341345234523452345235423455 END-DATA")
	fPath := path.Join(seedDir, fileName)
	err = ioutil.WriteFile(fPath, data, 0644)
	c.Assert(err, IsNil)

	//Use same hash algo as blobstore
	hash, err := hashFile(fPath)
	c.Assert(err, IsNil)

	dirs := []string{pasticheDir}
	blobStore, err := NewCustomBlobStore(dirs, 1, defaultEvictHeadroomMB, DefaultReservation, true)
	c.Assert(err, IsNil)

	// Load the seed dir's files into the storage dir.
	filesAdded, err := blobStore.AddFilesFromDir(seedDir)
	c.Assert(err, IsNil)
	c.Assert(int(filesAdded), Equals, 1)

	// check the file was brought in correctly.
	localOnly := true
	returnedPath, err2 := blobStore.GetPath(localOnly, hash)
	c.Assert(err2, IsNil)
	c.Assert(len(returnedPath), Not(Equals), 0)
}

// TestReload - start a one-off blobstore just to see if it picks up the file that's already there.
func (p *PasticheTest) TestReload(c *C) {
	var pasticheDir string
	var err error
	// Will start a new blobstore with this new dir
	pasticheDir, err = ioutil.TempDir("/tmp", "pstch-reload-test")
	c.Assert(err, IsNil)
	fileName := "this-would-be-a-hash-if-created-via-AddData"
	var data = []byte("DATA1 TEST12341345234523452345235423455 END-DATA")

	fPath := path.Join(pasticheDir, fileName)
	err = ioutil.WriteFile(fPath, data, 0644)
	c.Assert(err, IsNil)

	dirs := []string{pasticheDir}
	blobStore, err := NewCustomBlobStore(dirs, 1, defaultEvictHeadroomMB, DefaultReservation, true)
	c.Assert(err, IsNil)

	// check the file was brought in during initialization.
	localOnly := true
	returnedPath, err2 := blobStore.GetPath(localOnly, fileName)
	c.Assert(err2, IsNil)

	c.Assert(len(returnedPath), Not(Equals), 0)
	fmt.Printf("Found file [%s], existing in pastiche dir from a faked previous run", returnedPath)

}

// addFile - helper for other tests
func (p *PasticheTest) addFile(c *C, fileKey string) {
	c.Logf("=== Testing Add, Get, Clear, & cache size tracking")
	cacheSize := p.bs.CacheSizeMB()
	cacheUsedStart := p.bs.CacheUsedBytes()
	c.Assert(cacheUsedStart, Equals, uint64(0))

	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())

	c.Assert(true, Equals, true)

	//file := "some data"
	var data = []byte("DATA1 TEST12341345234523452345235423455 END-DATA")
	buf := bytes.NewBuffer(data)
	localOnly := true
	// data NOT in cache yet
	_, err := p.bs.GetPath(localOnly, fileKey)
	c.Assert(err, NotNil)

	bufSize := buf.Len()
	added := bufSize
	// Add some data
	_, err = p.bs.AddData(fileKey, buf)
	c.Assert(err, IsNil)
	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())
	c.Assert(p.bs.CacheUsedBytes(), Equals, uint64(added))

	// check it's there
	returnedPath, err2 := p.bs.GetPath(localOnly, fileKey)
	c.Assert(err2, IsNil)

	c.Assert(len(returnedPath), Not(Equals), 0)
}

func (p *PasticheTest) TestAddGet(c *C) {
	c.Logf("=== Testing Add, Get, Clear, & cache size tracking")
	cacheSize := p.bs.CacheSizeMB()
	cacheUsedStart := p.bs.CacheUsedBytes()
	c.Assert(cacheUsedStart, Equals, uint64(0))

	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())

	c.Assert(true, Equals, true)
	key := "testkey"
	//file := "some data"
	var data = []byte("DATA1 TEST12341345234523452345235423455 END-DATA")
	buf := bytes.NewBuffer(data)
	localOnly := true
	// data NOT in cache yet
	_, err := p.bs.GetPath(localOnly, key)
	c.Assert(err, NotNil)

	bufSize := buf.Len()
	added := bufSize
	// Add some data
	_, err = p.bs.AddData(key, buf)
	c.Assert(err, IsNil)
	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())
	c.Assert(p.bs.CacheUsedBytes(), Equals, uint64(added))

	// check it's there
	returnedPath, err2 := p.bs.GetPath(localOnly, key)
	c.Assert(err2, IsNil)

	c.Assert(len(returnedPath), Not(Equals), 0)

	// Add more data, to check cache size tracking
	key2 := "junk-key"
	buf = bytes.NewBuffer(data)
	_, err = p.bs.AddData(key2, buf)
	added += bufSize
	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())
	c.Assert(p.bs.CacheUsedBytes(), Equals, uint64(added))

	// delete & check it's gone
	p.bs.Delete(key)
	added -= bufSize
	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())
	c.Assert(p.bs.CacheUsedBytes(), Equals, uint64(added))
	_, err3 := p.bs.GetPath(localOnly, key)
	c.Assert(err3, NotNil)

	// Should be empty now
	p.bs.Delete(key2)
	added -= bufSize
	c.Logf("Cache size MB: %d    Cache used bytes: %d\n", cacheSize, p.bs.CacheUsedBytes())
	c.Assert(p.bs.CacheUsedBytes(), Equals, uint64(added))

	cacheUsedEnd := p.bs.CacheUsedMB()
	c.Logf("Cache size: %d    Cache used: %d\n", cacheSize, cacheUsedEnd)
}
