package pastiche

import (
	"container/heap"
	"sync"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

// CacheLogic - Everything but the file-system operations
type CacheLogic struct {
	mu    *sync.Mutex
	cache map[string]*CacheEntry // Blob hash keyed

	CacheSizeMB          uint   // Size threshold to trigger eviction. Will go over.
	CacheUsedBytes       uint64 // Includes reserved blob bytes.
	CacheEvictHeadroomMB uint   // Evict at least this much free below CacheSizeMB

	EvictAction         func(string) error
	ReservationDuration time.Duration
	// reservedBytes  int64 // Hard to track with lazy expiry checking.
	cacheLRU *PriorityQueue
	cstats   cacheStats
}

// NewCacheLogic - returns a CacheLogic
func NewCacheLogic(size uint, evictHdRm uint, evictAction func(string) error, reserveDur time.Duration) (*CacheLogic, error) {

	cacheMap := make(map[string]*CacheEntry)
	lru := &PriorityQueue{}
	cl := &CacheLogic{mu: &sync.Mutex{}, cache: cacheMap, CacheSizeMB: size, CacheEvictHeadroomMB: evictHdRm, EvictAction: evictAction, ReservationDuration: reserveDur, cacheLRU: lru}

	err := cl.CheckCacheParams()
	if err != nil {
		clog.Log.Log(nil, "Warning: %s", err)
	}

	return cl, nil
}

// CheckCacheParams - Returns an error if it looks like cache settings
// would lead to unstable behavior.
func (cl *CacheLogic) CheckCacheParams() error {
	// Check for headroom more than 20% or less than 1%, or cache
	// small enough to to have significant thrashing from headroom
	// mechanism.

	if (cl.CacheEvictHeadroomMB > (uint(.2 * float64(cl.CacheSizeMB)))) ||
		(cl.CacheEvictHeadroomMB < uint(.1*float64(cl.CacheSizeMB))) ||
		(cl.CacheSizeMB < 10) {
		return crux.ErrF("cache settings seem likely to cause problems. sizeMB %d   HeadroomMB %d", cl.CacheSizeMB, cl.CacheEvictHeadroomMB)

	}
	return nil
}

// AddEntry - Place a new item in cache or refresh last-added time.
func (cl *CacheLogic) AddEntry(key string, rsrcPath string, sizeBytes uint64) (uint64, error) {

	// TODO: make multi-path capable, or dis-allow multiple paths
	// for given hash.
	// Store multiple paths in a single map value, in case there's
	// multiple base dirs. Or...  multiple maps, searching each in
	// turn

	cl.mu.Lock()
	oldEntry, found := cl.cache[key]
	if found {
		// Updates last access time and reorders LRU queue
		cl.cacheLRU.Update(oldEntry, time.Now())
		cl.mu.Unlock() // unlock before possibly high latency log operation.

		if oldEntry.Path != rsrcPath {
			// TODO: Better define behavior if pastiche
			// has multiple directories.  Preferred soln
			// is to only allow a hash to exist in one
			// dir, but user could have pre-loaded two
			// dirs with same blob, maybe even with
			// different file names.
			// For now, overwrite old path with new path.  Shoul d
			clog.Log.Logi(nil, "%s re-added entry %v. New Path %s", oldEntry, rsrcPath)
			oldEntry.Path = rsrcPath
		}
		return cl.CacheUsedBytes, nil
	}

	// TODO: Performance: Callers could reduce memory usage by
	// replacing path's directory prefix with an integer index
	// field into the blobstore.StorageDirs slice.
	entry := &CacheEntry{Hash: key, Path: rsrcPath, SizeBytes: sizeBytes}
	entry.lastAccess = time.Now()
	cl.CacheUsedBytes += entry.SizeBytes
	cl.cache[entry.Hash] = entry
	heap.Push(cl.cacheLRU, entry)
	cl.mu.Unlock()
	clog.Log.Log(nil, "New entry, Total cache used: %d", cl.CacheUsedBytes)
	return cl.CacheUsedBytes, nil
}

// DeleteEntry - remove from all cache structures and return entry for any upstream handlers
func (cl *CacheLogic) DeleteEntry(key string) (*CacheEntry, error) {
	entry, ok := cl.cache[key]
	if !ok {
		return nil, crux.ErrF("key %s not in cache. Can't delete", key)
	}

	cl.CacheUsedBytes -= entry.SizeBytes
	// Clear the two cache entry locations.
	delete(cl.cache, key)
	heap.Remove(cl.cacheLRU, entry.index)

	return entry, nil
}

// GetEntry - return entry for key, or nil if key doesn't exist.
// Optionally refresh Last access time for cache ordering.
func (cl *CacheLogic) GetEntry(key string, refreshLastAccess bool) *CacheEntry {
	entry, found := cl.cache[key]
	if found {
		// GetPath counts as a cache hit for the blob.
		cl.cacheLRU.Update(entry, time.Now())
		cl.cstats.Hits++
		return entry
	}
	return nil
}

// FarFuture - Registration uses this as its reservation expiration,
// so registered files are effectively never evicted
var FarFuture = time.Date(2200, 1, 1, 1, 1, 1, 1, time.UTC)

// register - Add an entry to the cache with a reservation the will
// effectively never expire.  This is for files that should never be
// evicted, but should still be included in the cache usage total.
// Returns the time of expiration, which is a constant.
func (cl *CacheLogic) register(key string, registerMe bool) (*time.Time, error) {

	cl.mu.Lock()
	defer cl.mu.Unlock()

	entry, ok := cl.cache[key]
	if !ok {
		return nil, crux.ErrF("Key %s not in cache map, can't reserve", key)
	}

	if registerMe {
		entry.ReservationExpiry = FarFuture
		entry.lastAccess = time.Now() // Reservation counts as access
		cl.cstats.Hits++              // Reservation counts as an access
	} else {
		// un-reserve.  Set to now as breadcrumb that entry
		// was reserved at least once.
		entry.ReservationExpiry = time.Now()
	}
	// Reorder the heap.
	heap.Fix(cl.cacheLRU, entry.index)
	return &entry.ReservationExpiry, nil
}

// reserveDuration - Allow any duration.  Returns time of expiration
func (cl *CacheLogic) reserveDuration(key string, reserveMe bool, duration time.Duration) (*time.Time, error) {

	cl.mu.Lock()
	defer cl.mu.Unlock()

	entry, ok := cl.cache[key]
	if !ok {
		return nil, crux.ErrF("Key %s not in cache map, can't reserve", key)
	}

	if reserveMe {
		entry.ReservationExpiry = time.Now().Add(duration)
		entry.lastAccess = time.Now() // Reservation counts as access
		cl.cstats.Hits++              // Reservation counts as an access
	} else {
		// un-reserve.  Set to now as breadcrumb that entry
		// was reserved at least once.
		entry.ReservationExpiry = time.Now()
	}
	// Reorder the heap.
	heap.Fix(cl.cacheLRU, entry.index)
	return &entry.ReservationExpiry, nil
}

// Reserve - Make or Clear a reservation using the default reservation
// duration.  Returns time of reservation expiration.
func (cl *CacheLogic) Reserve(key string, reserveMe bool) (*time.Time, error) {
	return cl.reserveDuration(key, reserveMe, cl.ReservationDuration)
}

// CacheUsedMB - converts  bytes to MB
func (cl *CacheLogic) CacheUsedMB() uint {
	return uint(cl.CacheUsedBytes / (1024 * 1024))
}

// CacheSizeExceeded - Return true if cache exceeds max setting. For
// large cache objects, it is likely that cache size will be exceeded
// regularly, and brought back into compliance with the next eviction.
// When caching potentially large objects, it's a good idea to
// configure cache size lower than free space on storage, to allow for
// occasional overflows.
func (cl *CacheLogic) CacheSizeExceeded() bool {
	if cl.CacheUsedMB() > cl.CacheSizeMB {
		return true
	}
	return false
}

// EvictIfRequired - check and do evictions. Return space freed by
// evictions, if any.  Could be called before and/or after adding
// entries to the cache
func (cl *CacheLogic) EvictIfRequired() (uint64, error) {
	var evicted uint64
	var err error
	ll := clog.Log.With("focus", "EVICTION")
	if cl.CacheSizeExceeded() {
		evicted, err = cl.DoEvictions()
		ll.Log(nil, "cache size exceeded resulting in %d bytes evicted", evicted)
		if err != nil {
			ll.Log(nil, " eviction error: %s", err)
			return evicted, err
		}
	}
	return evicted, nil
}

// DoEvictions - remove enough unreserved entries to meet headroom
// requirements, using the cache's Evict Action function for any
// eviction side effects. Return bytes freed.
func (cl *CacheLogic) DoEvictions() (uint64, error) {
	// TODO: If begat shows thrashing tendencies, or low cache hit
	// rates, we can try better ones like ARC or CAR.

	// NOTE: There's a potential race between eviction of a
	//  resource and its use by its path/handle.  Unless the
	//  EvictAction function can check for conflict. (Detect readers)

	// FIXME: If pastiche is configured with multiple subdirs on
	// different devices, there's a edge case where eviction frees
	// space about evenly between the devices, but no one device
	// has enough space to take a file that technically fits in
	// the newly "available" space.

	// We could spend a long time under this lock if there are
	// many small files to evict or file-system(s) are busy.  Not
	// much benefit in a RW lock for Pastiche's use cases though.
	cl.mu.Lock()
	defer cl.mu.Unlock()
	ll := clog.Log.With("focus", "EVICTION")
	ll.Log("===== START EVICTION ======")
	var freedBytes uint64
	targetSizeMB := (cl.CacheSizeMB - cl.CacheEvictHeadroomMB)
	tsBytes := targetSizeMB * 1024 * 1024

	ll.Log(nil, ">> cache bytes:  Used:%d   Target:%d  ", cl.CacheUsedBytes, tsBytes)
	ll.Logf(nil, ">> cache MB:  Used:%d   Target:%d  ", cl.CacheUsedMB(), targetSizeMB)
	for cl.CacheUsedMB() > targetSizeMB {
		ll.Logf(nil, "cache bytes:  Used:%d   Target:%d  ", cl.CacheUsedBytes, tsBytes)
		entry := heap.Pop(cl.cacheLRU).(*CacheEntry)
		if entry == nil {
			return 0, crux.ErrF("no entries remain. Tiny cache or bad headroom parameter. usedMB %d  sizeMB %d  headroom %d", cl.CacheUsedMB(), cl.CacheSizeMB, cl.CacheEvictHeadroomMB)
		}

		//  No more un-reserved files. There's no use in going
		//  further.
		if entry.ReservationExpiry.After(time.Now()) {
			heap.Push(cl.cacheLRU, entry)
			return freedBytes, crux.ErrF("no un-reserved entries in cache. can't free enough space to reach headroom.  Make cache larger or reduce/remove reservations")
		}

		// Remove from map
		delete(cl.cache, entry.Hash)

		// Process action, like removing resource from filesystem,etc.
		err := cl.EvictAction(entry.Path)
		if err != nil {
			return freedBytes, crux.ErrF("couldn't evict file. %s", err)
		}

		// Adjust cache bookeeping and stats
		freedBytes += uint64(entry.SizeBytes)
		cl.CacheUsedBytes -= uint64(entry.SizeBytes)
		cl.cstats.Evictions++
		cl.cstats.EvictionBytes += uint64(entry.SizeBytes)
		ll.Logf(nil, "Evicted entry %+v", entry)
	}

	ll.Logf(nil, "Exiting Eviction")
	return freedBytes, nil
}

type cacheStats struct {
	Hits uint64
	// Misses        uint64 // Can't do misses, but could track new Adds.
	Evictions     uint64
	EvictionBytes uint64
}

// CacheEntry - Info needed by cache algorithms.
type CacheEntry struct {
	// Cache is LRU, but since clients bypass Pastiche for actual
	// reads, we track read-hits indirectly via calls to get path,
	// or set reservations.

	lastAccess        time.Time // Cache is LRU on this.
	ReservationExpiry time.Time // Or LRU on this.  See QTime()
	SizeBytes         uint64
	index             int
	// FIXME: Depending on how we handle multiple pastiche dirs
	// having the same file, we may need a slice for Path name
	Path string // Local file system path for the object
	Hash string // Hash of file/blob.
}

// IsRegistered - Returns true if the entry is registered and not expired.
func (ce *CacheEntry) IsRegistered() bool {
	if ce.ReservationExpiry.Equal(FarFuture) {
		return true
	}
	return false
}

// IsReserved - Returns true if the entry is reserved and not expired.
func (ce *CacheEntry) IsReserved() bool {
	if ce.ReservationExpiry.Before(time.Now()) {
		return false
	}
	return true
}

//QTime - Return lastAccess time or ReservationExpiry, whatever is
// larger/newer.  This allows the queue to be reordered correctly when
// entries are reserved (with expiry times in the future)
func (ce *CacheEntry) QTime() time.Time {
	if ce.ReservationExpiry.After(ce.lastAccess) {
		return ce.ReservationExpiry
	}
	return ce.lastAccess
}

// PriorityQueue - LRU list via priority queue using go's Heap
// interface.  Larger time values have higher priority.  Operations
// MUST be done via heap functions, heap.Init(), heap.Pop(q), etc.
// NOT THREAD SAFE
type PriorityQueue []*CacheEntry

// Len - Length of heap
func (pq PriorityQueue) Len() int { return len(pq) }

// Less - is i less worthy than j.  We want to pop entrys with the
// smallest (oldest) time value.
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].QTime().Before(pq[j].QTime())

}

// Swap - Exchange places of two entries
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push - Heap interface's Push impl
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	entry := x.(*CacheEntry)
	entry.index = n
	*pq = append(*pq, entry)
}

// Pop - Heap interface's Pop impl
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	l := len(old)
	entry := old[l-1]
	entry.index = -1 // for safety
	*pq = old[0 : l-1]
	return entry
}

// Update -  modifies the lastaccess time (priority) and reorders the
// queue.  Not part of heap interface.  Cheaper than heap.Remove + heap.Push
func (pq *PriorityQueue) Update(entry *CacheEntry, lastAccess time.Time) {
	entry.lastAccess = lastAccess
	heap.Fix(pq, entry.index)
}
