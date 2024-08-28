package shutdown

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ret(fn func(), t ...time.Duration) error {
	var d time.Duration
	if len(t) == 0 {
		d = time.Millisecond
	}
	for _, tt := range t {
		d += tt
	}
	done := make(chan struct{})
	tick := time.After(d)

	go func() {
		fn()
		done <- struct{}{}
	}()

	select {
	case <-done:
		return nil
	case <-tick:
		return fmt.Errorf("Timeout expired (%s)", d.String())
	}
}

func TestWaitStack(t *testing.T) {
	// empty stack will return
	ws := waitStack{&sync.WaitGroup{}}
	assert.Nil(t, ret(func() { ws.Wait() }))

	// stack with one item will block
	ws.Add(1)
	assert.NotNil(t, ret(func() { ws.Wait() }))

	// Removing the item causes the stack to unblock
	ws.Done()
	assert.Nil(t, ret(func() { ws.Wait() }))

	// Reset to get around issue with ret() where blocking will
	// cause the error 'WaitGroup is reused before previous Wait has returned'
	// to be returned if you call ret() again on the previously blocked
	// group.

	ws = waitStack{&sync.WaitGroup{}}
	ws2 := append(ws, &sync.WaitGroup{})
	// Second layer will return as it is also empty.
	assert.Nil(t, ret(func() { ws2.Wait() }))

	ws2.Add(1)
	ws.Add(1)
	// Adding to both will block for good reason.
	assert.NotNil(t, ret(func() { ws.Wait() }))
	assert.NotNil(t, ret(func() { ws2.Wait() }))

	// Removing from layer two causes layer two to
	// return but layer one will still block.
	ws2.Done()
	assert.NotNil(t, ret(func() { ws.Wait() }))
	assert.Nil(t, ret(func() { ws2.Wait() }))

	// Removing from layer one causes it to unblock
	ws.Done()
	assert.Nil(t, ret(func() { ws.Wait() }))
}
