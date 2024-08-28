package pastiche

import (
	"fmt"
	"strconv"
	"time"

	. "gopkg.in/check.v1"
)

type CacheTest struct {
	cache *CacheLogic
}

func init() {
	ct := CacheTest{}

	var CacheSizeMB uint = 10
	var EvictHeadroomMB = CacheSizeMB / 10
	var Reservation = time.Second * 100

	evictFunc := func(path string) error {
		fmt.Printf("-- Dummy evict function, doing nothing for path %s\n", path)
		return nil
	}

	cacheLogic, err := NewCacheLogic(CacheSizeMB, EvictHeadroomMB, evictFunc, Reservation)
	if err != nil {
		fmt.Printf("Error creating CacheLogic %s\n", err)
	}
	ct.cache = cacheLogic
	Suite(&ct)
}

// TestCacheOps - evictions and reservation tests.
func (ct *CacheTest) TestCacheOps(c *C) {
	cacheSizeBytes := uint64(ct.cache.CacheSizeMB * 1024 * 1024)
	entriesFitInCache := uint64(5)
	entrySize := uint64(cacheSizeBytes / entriesFitInCache)

	fmt.Printf("Cache size MB: %d   Cache entry size Bytes: %d\n", ct.cache.CacheSizeMB, entrySize)
	var reserveList []string

	// Choose number of entries to fill beyond cache max size,
	// reserving every other. Eviction should hit two items to
	// reach headroom goal
	numEntries := uint64(entriesFitInCache + 2)
	var totSize uint64
	var i uint64
	for i = 0; i < numEntries; i++ {
		// Path and keys are meaningless here, and are only to
		// provide valid parameters for unique cache entries.
		key := strconv.FormatUint(i, 2)
		path := strconv.FormatUint(i, 10)
		size := entrySize // Fixed size entries
		totSize += size
		used, err := ct.cache.AddEntry(key, path, size)
		c.Assert(nil, Equals, err)
		c.Assert(totSize, Equals, used)
		fmt.Printf("Used Bytes %d\n", used)

		// Reserve every other
		if i%2 == 0 {
			_, err := ct.cache.Reserve(key, true)
			c.Assert(nil, Equals, err)
			reserveList = append(reserveList, key)
		}
	}

	// Evict, should not touch reserved items
	freed, err := ct.cache.EvictIfRequired()
	c.Assert(nil, Equals, err)
	fmt.Printf("Cache Headroom MB:%d   Freed bytes:%d    Cache Used bytes:%d\n", ct.cache.CacheEvictHeadroomMB, freed, ct.cache.CacheUsedBytes)
	freeSpace := uint64(ct.cache.CacheSizeMB*1024*1024) - ct.cache.CacheUsedBytes
	headRoomSatisfied := freeSpace >= uint64(ct.cache.CacheEvictHeadroomMB*1024*1024)
	c.Assert(headRoomSatisfied, Equals, true)

	setLastAccess := false
	setReserved := false
	// Verify all reserved still there (and clear reservation for next phase)
	for _, rKey := range reserveList {
		// Get entry _without_ modifying last-access so
		// they'll still be "oldest" for next stage of test.
		entry := ct.cache.GetEntry(rKey, setLastAccess)
		c.Assert(nil, Not(Equals), entry)
		// Clear reservation.
		_, err := ct.cache.Reserve(rKey, setReserved)
		c.Assert(nil, Equals, err)
	}

	// Enough new unique entries to fill cache again so oldest
	// (previously reserved ones) will get flushed.
	for i = numEntries; i < (numEntries * 2); i++ {
		key := strconv.FormatUint(i, 16)
		path := strconv.FormatUint(i, 10)
		used, err := ct.cache.AddEntry(key, path, entrySize)
		c.Assert(nil, Equals, err)
		fmt.Printf("Used Bytes %d\n", used)
	}

	freed, err = ct.cache.EvictIfRequired()
	c.Assert(nil, Equals, err)
	fmt.Printf("Cache Headroom MB:%d   Freed bytes:%d    Cache Used bytes:%d\n", ct.cache.CacheEvictHeadroomMB, freed, ct.cache.CacheUsedBytes)

	freeSpace = uint64(ct.cache.CacheSizeMB*1024*1024) - ct.cache.CacheUsedBytes
	headRoomSatisfied = freeSpace >= uint64(ct.cache.CacheEvictHeadroomMB*1024*1024)
	c.Assert(headRoomSatisfied, Equals, true)

	// Check that last eviction got it's space from "old" unreserved entries.
	for _, rKey := range reserveList {
		entry := ct.cache.GetEntry(rKey, setLastAccess)
		c.Assert(entry, IsNil)
	}
}
