package pastiche

// TestQueue: Hook gocheck into "go test" runner
import (
	"fmt"
	"time"

	"container/heap"
	. "gopkg.in/check.v1"
)

type QueueTest struct {
	pq *PriorityQueue
}

func init() {
	qt := QueueTest{}
	qt.pq = &PriorityQueue{}
	Suite(&qt)
}

func dumpQueue(pq *PriorityQueue) {
	//fmt.Printf("Dumping queue\n")
	for k, v := range *pq {

		fmt.Printf("entry %d  is %v\n", k, v)
	}
	fmt.Printf("\n")

}

//TestQueue - test for priority queue behavior
func (qt *QueueTest) TestQueueOps(c *C) {
	q := qt.pq

	entryData := []string{"/zero", "/one", "/two", "/three", "/four"}
	baseTime := time.Now()
	// load "cache" with successively younger entries.
	for k, v := range entryData {
		heap.Push(q, &CacheEntry{Path: v, lastAccess: baseTime.Add(time.Duration(k) * time.Second)})
	}

	//	dumpQueue(q)

	// q has all entries?
	c.Assert(q.Len(), Equals, len(entryData))

	// check re-inserting preserves order
	e1 := heap.Pop(q).(*CacheEntry)
	heap.Push(q, e1)
	e1b := heap.Pop(q).(*CacheEntry)
	c.Assert(e1b, Equals, e1)

	// Check that ReservationExpiry overrides lastAccess when present
	newEntry := *e1b
	newEntry.ReservationExpiry = time.Now().Add(time.Duration(100) * time.Second)
	//	fmt.Printf("Before %+v\nAfter  %+v\n",e1, &newEntry)
	heap.Push(q, &newEntry)
	// This entry is now reserved with a expiry higher than any
	// other item's lastAccess time, and should be last off the
	// queue.

	var ePrev *CacheEntry
	for q.Len() > 0 {
		e := heap.Pop(q).(*CacheEntry)
		if ePrev != nil {
			if !ePrev.QTime().Before(e.QTime()) {
				c.Errorf("Popped entry (%+v) did not have QTime larger than previous entry ()%+v\n", e, ePrev)
			}
		}
		ePrev = e
	}

	// Did the reserved /zero get popped off last?
	c.Assert(ePrev.Path, Equals, entryData[0])
}
