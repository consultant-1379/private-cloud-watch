package pastiche

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

// Pastiche has features of a file cache manager, and named-data flat
// namespace file storage.  It includes some basic clustering
// operations/abilities.  While this may be generally useful, the
// Begat build system is the targeted user.  This package is the core
// code, and gRPC server.  There should be a command line tool of the
// same name elsewhere.

// Pastiche is primarily a service that manages semi-persistent files
// on a node. These files live in a set of directories specified by
// configuration (of pastiche on that node). Individual files have
// expiry times and in-use indicators to prevent premature
// flushing. Generally speaking, data should be thought of as a
// key/value store; that is, the data (or value) will be the contents
// of a file, and the key will be a unique string (most often a
// cryptohash of the data). There are some exceptions to this noted
// below. Be warned: the key may well have embedded Unix-style
// pathnames or other magic characters.

// API - To avoid tight binding to the pastiche implementation, code
// requiring a Pastiche should use this interface as a parameter, vs
// using BlobStore directly..
type API interface {
	RegisterPermanentFile(key string, fullPath string) error
	AddDirToCache(dirPath string, loadFiles bool) error
	AddData(key string, input io.Reader) (string, error)
	AddDataFromFile(key string, filePath string) (string, error)

	GetPath(localOnly bool, key string) (string, error)
	Delete(key string) error
	DeleteAll() error
	AddTar(gitCommitHash string, tarPath string) (string, error)
	AddDataFromRemote(key string) (string, error)
	SetReservation(key string, reservered bool) (*time.Time, error)
}

var _ API = (*BlobStore)(nil) // Test it satisfies interface

// InProgressSubDir - sub directory to be created under base
// directory(s) for files during transfer.
const InProgressSubDir = "in-progress"

// RemotePathSeparator - Separates server IP and filename in URI paths
// returned by GetPath.
// ex:  192.72.0.1<sep>/path/to/file
var RemotePathSeparator = "#"

// AddPrefix - utility func to create server<Sep>path URI's.
func AddPrefix(filePaths []string, server string) {
	for i, f := range filePaths {
		filePaths[i] = server + RemotePathSeparator + f
	}
}

// GetServerName - Return server part of a URI. inverse of AddPrefix,
// but for single value.
func GetServerName(fileURI string) (string, error) {

	vals := strings.Split(fileURI, RemotePathSeparator)
	if len(vals) != 2 {
		return "", crux.ErrF("Not a valid URI, no separator %v found in %s ", RemotePathSeparator, fileURI)
	}

	return vals[0], nil

}

// BlobStore - Implements the Pastiche API. state of pastiche managed storage
type BlobStore struct {
	//restarts   // Generation number or restart count
	mu          *sync.Mutex
	id          string // TODO: Remove if unique id/name not needed.
	startTime   time.Time
	storageDirs []string // Absolute paths
	cl          *CacheLogic
}

var defaultCacheMB uint = 100                     // Small, but okay for development.
var defaultEvictHeadroomMB = defaultCacheMB / 100 // 1% headroom.  Threshold at which eviction will start.

// DefaultReservation - Set to 10 minutes for now. We could make it
// user configurable up to a limit. (One size might not fit all).
var DefaultReservation = time.Minute * 10

// NewBlobStore - returns new pastiche instance, after validating the storage
// dir is accessible.  Existing files in dir are loaded
func NewBlobStore(storageDirPaths []string) (*BlobStore, error) {
	return NewCustomBlobStore(storageDirPaths, defaultCacheMB, defaultEvictHeadroomMB, DefaultReservation, true)
}

// NewCustomBlobStore - Allow setting of blobstore's parameters.
// Choose a evictHeadroom value that's large enough to trigger
// eviction before the cache runs out of space, but not so large that
// wanted entries are flushed early.
func NewCustomBlobStore(storageDirPaths []string, CacheSize uint, evictHeadroom uint, reservationDuration time.Duration, preload bool) (*BlobStore, error) {
	// TODO: check if another pastiche is already using thisserv
	// directory, else create our own marker file.  Bad things
	// could happen if multiple containers are mapping through the
	// same directories for pastiche.

	// TODO: Make hash algorithm a constructor parameter
	clog.Log.Logm("focus", "LIFECYCLE", nil, "creating pastiche instance with directories: %v", storageDirPaths)
	if !preload {
		clog.Log.Logm("focus", "LIFECYCLE", nil, "Preload not selected. On start, will NOT load any cached (on-disk) files from previous operation.")
	}

	// On removing from cache structs, delete the file or directory hierarchy.
	evictFunc := func(path string) error {
		fmt.Printf("----------- EVICT FUNC on %s\n", path)
		return os.RemoveAll(path)
	}

	cacheLogic, err := NewCacheLogic(CacheSize, evictHeadroom, evictFunc, reservationDuration)
	if err != nil {
		return nil, err
	}
	bs := &BlobStore{mu: &sync.Mutex{}, id: "pastiche core", startTime: time.Now(), storageDirs: storageDirPaths, cl: cacheLogic}

	if storageDirPaths == nil {
		clog.Log.Logm("focus", "INIT", nil, "Blobstore created without any directories. Legal usage, but maybe not intented usage.")
		return bs, nil
	}

	// loading files could be slow for large /many files or
	// busy devices.

	// For now, not speeding this up by
	// occasionally storing the map on disk. See
	// SaveState() stub for issues.

	// TODO: If the storage dirs are on different
	// physical devices, the load speed could
	// benefit from being done in parallel. Same
	// is true for one device if IO bandwidth
	// exceeds single thread compute bandwidth.
	clog.Log.Log(nil, "BLOBSTORE  NewCustom(...) has storage dirs [+%v]", storageDirPaths)
	for _, dir := range storageDirPaths {
		err := bs.AddDirToCache(dir, preload)
		if err != nil {
			return nil, err
		}
	}

	return bs, nil
}

// Configured - return an error if server not fully configured.  Some
// settings, such as storage directories, can be set via the API after
// a server using the BlobStore has been started.  Any function
// writing to the cache dir should check this before doing write
// operations, for clearer error causes.
func (bs *BlobStore) Configured() error {
	if len(bs.storageDirs) == 0 {
		// Someone didn't configure storage during New() or call AddDirToCache() after New()
		clog.Log.Log("Configured() fail:  There should be at least one storage dir before a Add*() is called")
		return crux.ErrF("the blobstores storage directories were not configured")
	}
	return nil
}

// GetStorageDirs - Return slice of paths the blob store was configured with
func (bs *BlobStore) GetStorageDirs() []string {
	return bs.storageDirs
}

// RegisterPermanentFile - For files you don't want to be
// evicted/deleted.  Bootstrap requirements, for example.
// - The file path must exist on the SERVER.
// - The file's directory is not added to the cache-dirs list,
// and so RemoveAllFiles() will not affect it.
//  - Space used counts towards cache space.
func (bs *BlobStore) RegisterPermanentFile(key string, fullPath string) error {
	info, err := os.Stat(fullPath)
	if err != nil {
		return crux.ErrE(err)
	}
	size := uint64(info.Size())
	// Add a cache entry for this file
	bs.cl.AddEntry(key, fullPath, size)

	bs.cl.register(key, true)
	return nil
}

// AddDirToCache - Initialize required pastiche subdirs and optionally
// load existing files into the cache for future lookup.
// - Pastiche supports multiple directories to allow use of
// multiple devices. There is little to no benefit in using
// directories on the same device
// - AddData*() calls create files with data hashes as filenames.
func (bs *BlobStore) AddDirToCache(dirPath string, loadFiles bool) error {
	// Dir must pre-exist. Detects mis-keying dir names
	if _, err := os.Stat(dirPath); err != nil {
		clog.Log.Log("focus", "LIFECYCLE", nil, "AddDirToCache - passed dir doesn't exist %s", dirPath)
		return crux.ErrE(err)
	}
	// Create a sub directory for in-transit files, if needed.
	err := os.Mkdir(path.Join(dirPath, InProgressSubDir), os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			clog.Log.Log("focus", "LIFECYCLE", nil, "AddDirToCache - couldn't create a %s dir in  %s", InProgressSubDir, dirPath)
			return crux.ErrE(err)
		}
	}

	bs.storageDirs = append(bs.storageDirs, dirPath)
	if !loadFiles {
		return nil
	}

	clog.Log.Log("focus", "LIFECYCLE", nil, "Loading File hashes into cache %v", dirPath)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		clog.Log.Fatal(err)
	}

	for _, file := range files {
		// read and hash the file
		fname := file.Name()
		if fname == InProgressSubDir {
			continue
		}
		filePath := filepath.Join(dirPath, fname)

		var hash string

		if file.IsDir() {
			// Currently, only tar files are added as
			// directories.  This is meant for git
			// tarballs.
			clog.Log.Log(nil, "Adding directory [%s] from disk, using its name as it's key (Git Ref ID)")
			hash = fname
		} else {
			// FIXME:  Filename should _be_
			// hash. Don't hash, or check the hash matches
			// the file name.   #PMD
			// hash, err = hashFile(filePath)
			hash = fname
		}
		//clog.Log.Log("focus","DETAIL", "Loading file %s as %s", filePath, hash)

		_, err = bs.cl.EvictIfRequired()
		if err != nil {
			return err
		}
		// TODO: AddEntry sets lastAccess to "now" which gives
		// no valuable LRU ordering in this case.. For better
		// accuracy here we could:
		// 1. store access times in a metadata file and read
		// back
		// 2. Use file write time as last-access (easier,
		// disk-seeky, less accurate)
		clog.Log.Log(nil, "Adding key %s", hash)
		_, err = bs.cl.AddEntry(hash, filePath, uint64(file.Size()))
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadConfigFile - TODO: This should be a "muck" file containing the
// directories to use as storage, and any "seed" directories to
// initialize the cache with.  Also, will need to call this from a
// constructor, or trigger via a grpc call.  Pick your poison.
func (bs *BlobStore) ReadConfigFile(path string) error {
	//  Muck. Muck. Muck.

	// Loop over storage dirs, adding them

	// Loop over seed dirs.
	// added,err:=AddFilesFromDir(seedDir)
	return crux.ErrF("Not Implemented yet")
}

// AddFilesFromDir - Do an AddFile for every file in the input
// argument directory.  For "seeding" pastiche at startup.  The
// provided directory is not tracked, monitored, or used for cache
// storage.  It is forgoteen by the blobstore after this call.  The
// number of files added is returned.
// NOTE: Earlier files may be flushed by later files, so there is no
// guarantee that all files in the directory will be in-cache at the
// end of the call.  Watch out for small cache values and/or large
// directories.
func (bs *BlobStore) AddFilesFromDir(dirPath string) (uint64, error) {
	clog.Log.Log("focus", "LIFECYCLE", nil, "Loading Files from dir %s", dirPath)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		clog.Log.Fatal(err)
		return 0, crux.ErrE(err)
	}
	var filesAdded uint64
	for _, file := range files {
		fname := file.Name()
		filePath := filepath.Join(dirPath, fname)
		hash, err := hashFile(filePath)
		if err != nil {
			clog.Log.Log(nil, "Error hashing  %s   %s", fname, err)
			return filesAdded, crux.ErrE(err)
		}
		destPath, err := bs.AddDataFromFile(hash, filePath)
		if err != nil {
			clog.Log.Log(nil, "Error adding file  %s   %s", fname, err)
			return filesAdded, crux.ErrE(err)
		}
		// Andrew said not to make seeded files durable in the cache.
		//err = RegisterPermanentFile(hash, filePath)

		clog.Log.Log(nil, "Loading File into cache as %s", destPath)

		filesAdded++
	}
	return filesAdded, nil
}

// AddTarReader - tarRdr is a stream in tar format that will be added in
// its expanded form to the blobstore
func (bs *BlobStore) AddTarReader(key string, tarRdr io.Reader, isZipped bool) (string, error) {

	if err := bs.Configured(); err != nil {
		return "", err
	}

	_, err := bs.cl.EvictIfRequired()
	if err != nil {
		return "", err
	}

	storageDir, err := bs.GetWritableDir()
	if err != nil {
		return "", err
	}

	// Keep the expansion out of the storage dir top level until
	// it's all there.
	tempPath := path.Join(storageDir, InProgressSubDir, key)
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		os.MkdirAll(tempPath, (os.ModeDir | os.ModePerm))
	}
	fmt.Printf("+++++++++ AddTarReader - tempPath %s\n", tempPath)
	numBytes, _, err := Untar(tarRdr, tempPath, isZipped)
	if err != nil {
		bs.RemoveTempFile(tempPath)
		return "", crux.ErrE(err)
	}

	// TODO: better func name. Not always a blob.  relocateAndFinalize()
	// finalizeEntry()
	finalPath, err := bs.AddBlobWithEntry(key, tempPath, numBytes)
	if err != nil {
		return "", err
	}
	fmt.Printf("+++++++++ AddTarReader - finalPath %s\n", finalPath)
	return finalPath, nil
}

// AddTar - Unpack a git tarball and add to the blobstore. Returns the
// path of the untar location.  It makes sense to pass the commit hash
// for the key here.
// NOTE, To check the commit id:
//    cat gitRepoArchive.tar | git get-tar-commit-id
func (bs *BlobStore) AddTar(key string, tarballName string) (string, error) {
	if err := bs.Configured(); err != nil {
		return "", err
	}

	if _, err := os.Stat(tarballName); os.IsNotExist(err) {
		return "", crux.ErrF("file does not exist: %s", err)
	}

	rdr, err := os.Open(tarballName)
	if err != nil {
		return "", crux.ErrF("file does not exist: %s", err)
	}

	defer rdr.Close()

	isZipped := path.Ext(tarballName) == ".tgz"

	destPath, err := bs.AddTarReader(key, rdr, isZipped)
	if err != nil {
		return "", crux.ErrE(err) // TODO: crux func to wrap an error w/o removing stack. Augment()? crux.Foo("file does not exist: %s", err)
	}
	return destPath, nil
}

// hashFile - Return, a string of the computed hash suitable for use a
// map key
func hashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", crux.ErrE(err)
	}
	defer f.Close()
	// TODO: make the hashing algorithm configurable via the constructor
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", crux.ErrE(err)
	}

	hash := hex.EncodeToString(h.Sum(nil))
	return hash, nil
}

// SaveState -persist cache map, or anything to help with starting a pastiche
// from this cache.
func (bs *BlobStore) SaveState() error {
	// TODO: We could occasionally store the cache map to a file,
	// but since both adds and deletes are done synchronously,
	// disk state should always be same as in-mem representation.

	// Not doing for now, since we can restart without any extra
	// metadata.  Would need a use case worth the added
	// complexity.

	// For data integrity checks, we can compute and check hashes
	// against the one used for the filename (except expanded
	// tars).

	return crux.ErrF("Not Implemented")
}

// FinalPath - Transform "in progress" path to final path
func FinalPath(inProgressPath string) string {
	// For now, in-progress dir will be a child of main storage dir, so
	// just move it up to parent directory.

	// Eventually we may want to use a hashed directory structure
	// to reduce directory sizes if underlying filesystem has perf
	// issues

	// NOTE: The in-progress path should be on same filesystem as final
	// path to avoid copy overhead on move and to get atomic
	// rename behavior.
	path := strings.Replace(inProgressPath, InProgressSubDir+"/", "", 1)
	clog.Log.Log("focus", "PATH", nil, " path is %s ", path)
	return path
}

// GetWritableDir only says it has space left, and is the
// least full of the storage devices. No guarantee there's
// enough room for the (potentially uncompressed) tar.
func (bs *BlobStore) GetWritableDir() (string, error) {

	if len(bs.storageDirs) == 0 {
		clog.Log.Logm("InProgressPath() : No storage dirs configured, can't construct a path for key.")
		return "", crux.ErrF("no storage dirs configured")
	}

	dirNum := 0 // TODO: pick the least-full of the N possible storage directories.
	return bs.storageDirs[dirNum], nil

}

// InProgressPath - Path for files to be written to the "inprogress" directory.
func (bs *BlobStore) InProgressPath(key string) (string, error) {
	storDir, err := bs.GetWritableDir()
	if err != nil {
		return "", err
	}
	path := path.Join(storDir, InProgressSubDir, key)
	clog.Log.Log("focus", "PATH", nil, " path is %s for key %s", path, key)
	return path, nil
}

// GetPath -Return error if no key found, and empty string
func (bs *BlobStore) GetPath(localOnly bool, key string) (string, error) {
	entry := bs.cl.GetEntry(key, true)
	if entry != nil {
		//TODO: multi-path returns for get.
		return entry.Path, nil
	}
	unavail := ""
	// localOnly here for interface completeness.  Implementation
	// requires a rpc or protocol.
	if !localOnly {
		unavail = "Remote lookup not available in base library."
	}
	clog.Log.Log("focus", "CACHE", nil, "key not found [%s] localonly %t", key, localOnly)
	return "", crux.ErrF("Key not in cache.  No path found.  %s", unavail)
}

const writeFault = false //force any AddData calls to fail.

// NewBlobReader - Given a key (usually a hash), find the file for it
// and return a reader for it.
func (bs *BlobStore) NewBlobReader(key string) (io.ReadCloser, error) {
	if key == "" {
		err := crux.ErrF("key argument empty")
		return nil, err
	}

	path, err := bs.GetPath(false, key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, crux.ErrE(err)
	}
	return f, nil
}

// NewBlobTempFile - Return a file in the "in progress" area for
// writing.  Caller is expected to close file.
func (bs *BlobStore) NewBlobTempFile(key string) (*os.File, error) {
	// TODO: Validate that key isn't too big for a filename.
	// TODO: Do we want a warning on a file over-write.
	tempPath, err := bs.InProgressPath(key)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(tempPath)
	// FIXME: We shouldn't throw an error for a pre-existing file. Just overwrite.
	if err != nil {
		return nil, crux.ErrE(err)
	}
	return f, nil
}

// AddBlobWithEntry - Called after a blob is successfully written, to relocate
// and add to cache structures.
// Pre-existing data stored at the same hash/key will be overwritten without warning.
func (bs *BlobStore) AddBlobWithEntry(key string, origPath string, size uint64) (string, error) {
	clog.Log.Logm("CacheAdd")
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Put blob in final storage path
	finalPath := FinalPath(origPath)
	// remove any pre-existing data for this hash's path
	err := os.RemoveAll(finalPath)
	if err != nil {
		return "", crux.ErrE(err)
	}

	err = os.Rename(origPath, finalPath)
	if err != nil {
		return "", crux.ErrE(err)
	}

	// Cache bookkeeping
	_, err = bs.cl.AddEntry(key, finalPath, size)
	if err != nil {
		return "", err
	}
	clog.Log.Log(nil, "blobstore AddBlobWithEntry :  finalpath %s", finalPath)
	return finalPath, nil
}

// RemoveTempFile - For api symmetry with AddBlobWithEntry, and
// to prevent callers from making assumptions about removing the file
// being enough to clear everything, which may not always be the case.
func (bs *BlobStore) RemoveTempFile(fileName string) error {
	return os.RemoveAll(fileName)
}

// AddData - this will create a file named <key> and add it to the
// in-memory map for fast lookup and cache management.
func (bs *BlobStore) AddData(key string, input io.Reader) (string, error) {
	if err := bs.Configured(); err != nil {
		return "", err
	}

	_, err := bs.cl.EvictIfRequired()
	if err != nil {
		return "", err
	}

	f, err := bs.NewBlobTempFile(key)
	tempPath := f.Name()
	size, err := io.Copy(f, input)
	if err != nil {
		f.Close()
		bs.RemoveTempFile(tempPath)
		return "", crux.ErrE(err)
	}
	clog.Log.Log("focus", "DATA", nil, " wrote %d bytes for key %s", size, key)

	err = f.Close()
	if err != nil {
		bs.RemoveTempFile(tempPath)
		return "", crux.ErrE(err)
	}

	finalPath, err := bs.AddBlobWithEntry(key, tempPath, uint64(size))
	if err != nil {
		// Cache may be in an inconsistent state.
		// TODO:  "inconsistent"  err msg.
		return "", err
	}
	return finalPath, nil
}

// AddDataFromFile - Copy the file's content to a new file, who's name is the
//  key.  Sugar to relieve user of creating/closing a file reader.
//  Note: There's potential confusion here since this doesn't preserve
//  the filename. See AddDirToCache(..) for filename preserving behavior.
func (bs *BlobStore) AddDataFromFile(key string, fileName string) (string, error) {

	rdr, err := os.Open(fileName)
	if err != nil {
		return "", crux.ErrE(err)
	}
	defer rdr.Close()
	return bs.AddData(key, rdr)
}

// AddDataFromRemote - Not implemented on local server
func (bs *BlobStore) AddDataFromRemote(key string) (string, error) {
	return "", crux.ErrF("not implemented on a bare blobstore. See grpc server")
}

// SetReservation - Prevents the blob represented by key (a hash) from
// being evicted from the cache until the expiration reached.
func (bs *BlobStore) SetReservation(key string, reserve bool) (*time.Time, error) {
	return bs.cl.Reserve(key, reserve)
}

// Delete - Remove entry from cache and blob or dir from disk.
func (bs *BlobStore) Delete(key string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	entry, err := bs.cl.DeleteEntry(key)
	if err != nil {
		return err
	}
	err = os.RemoveAll(entry.Path)
	if err != nil {
		return crux.ErrF("Inconsistent State: Blobstore delete failed. Key and any data still remain. %s", err)
	}

	return nil
}

// DeleteAll - Remove all data from disk and internal map.
// WARNING: This will delete content from _all_ directories under
// patiche's control, even if it did not originate them.
func (bs *BlobStore) DeleteAll() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for _, dir := range bs.storageDirs {
		err := os.RemoveAll(dir)
		if err != nil {
			// cache used size will not match what's really there if we fail in middle of removing.
			return crux.ErrF("Inconsistent State: Failed to remove some or all  files during a Clear() operation. : %s", err)
		}
	}
	return nil
}

// CacheUsedBytes - How much is used on disk
func (bs *BlobStore) CacheUsedBytes() uint64 {
	return bs.cl.CacheUsedBytes
}

// CacheUsedMB - How much is used on disk.
func (bs *BlobStore) CacheUsedMB() uint {
	return bs.cl.CacheUsedMB()
}

// CacheSizeMB - Max disk space the blobstore is allowed to use
func (bs *BlobStore) CacheSizeMB() uint {
	return bs.cl.CacheSizeMB
}
