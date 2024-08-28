// Copyright 2016 Ericsson AB All Rights Reserved.
// Contributors:
//     Andrew Hume
//     Scott Devoid

/*
Package shutdown addresses the problem of konwing when it is safe to shutdown
for a complex, multithreaded application. As background, the `net/context` package provides a
mechanism for signaling blocking threads to cancel via the `WithDeadline()` and `WithCancel()`
functions but then the application still needs to know when it is *safe* to cancel.

This package solves that in a scoped fashion by attaching "wait-layers" to `net/context.Context`
objects. First the application calls `WithWaitLayer` on an existing context object. The returned
context can then be used in the other functions in this package: `WithCleaner()` and `Wait()`.

*/
package shutdown

import (
	"sync"

	"golang.org/x/net/context"
)

func WithWaitLayer(ctx context.Context) context.Context {
	ctx, ws := getWaitStack(ctx)
	ws = append(ws, &sync.WaitGroup{})
	rtx := context.WithValue(ctx, waitKey, ws)
	return rtx
}

// Wait blocks until all registered Cleaners have completed.
func Wait(ctx context.Context) {
	_, w := getWaitStack(ctx)
	w.Wait()
}

// Cleaner implements the CleanUp() function
type Cleaner interface {
	CleanUp() error

	// Prevent any other packages from creating implementations of Cleaner.
	// We do this because a Cleaner should only be created with a ctx so that
	// the step can be recorded in the waitStack.
	implementsCleaner()
}

// WithCleaner returns a Cleaner that executes the function passed in.
// The Cleaner.CleanUp() function must then be called at some point
// in order for Wait() to return. In other words, this function increments
// the hidden wait-group and wraps fn with an operation to decrement that
// same wait group.
func WithCleaner(ctx context.Context, fn func() error) Cleaner {
	_, w := getWaitStack(ctx)
	wrapped := func() error {
		err := fn()
		w.Done()
		return err
	}
	w.Add(1)
	return &cleaner{wrapped}
}

// cleaner is a struct that implements the Cleaner interface.
type cleaner struct{ fn func() error }

// CleanUp the method that invokes the passed cleanup function.
func (s *cleaner) CleanUp() error     { return s.fn() }
func (s *cleaner) implementsCleaner() {}

// ParallelCleaner gathers multiple Cleaners together.
type ParallelCleaner struct{ c []Cleaner }

// Add adds a Cleaner to the ParallelCleaner
func (p *ParallelCleaner) Add(s Cleaner) {
	p.c = append(p.c, s)
}

// CleanUp calls CleanUp on all cleaners added to ParallelCleaner
func (p *ParallelCleaner) CleanUp() {
	for _, s := range p.c {
		go s.CleanUp()
	}
}

// helper functions and types below here

type waitStack []*sync.WaitGroup

func (ws waitStack) Add(i int) {
	for _, wg := range ws {
		wg.Add(i)
	}
}

func (ws waitStack) Wait() {
	if len(ws) == 0 {
		return
	}
	ws[len(ws)-1].Wait()
}

func (ws waitStack) Done() {
	for _, wg := range ws {
		wg.Done()
	}
}

// key and waitKey are used to bind a waitStack into
// the context.Context object.
type key int

const waitKey key = 1

// getWaitStack is a helper that ensures that the waitStack has been properly
// bound to the context. If it hasn't this binds the waitStack and returns the
// newly created context.
func getWaitStack(ctx context.Context) (context.Context, waitStack) {
	w, ok := ctx.Value(waitKey).(waitStack)
	if !ok {
		w = waitStack{&sync.WaitGroup{}}
		ctx = context.WithValue(ctx, waitKey, w)
	}
	return ctx, w
}
