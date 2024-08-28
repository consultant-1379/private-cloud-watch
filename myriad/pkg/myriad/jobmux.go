// Copyright 2016 Ericsson AB All Rights Reserved.

package myriad

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/erixzone/myriad/pkg/util/log"
)

// JobMux collects error responses from added jobs and will block with a
// Wait() call until all registered jobs have completed. Note
// that you should not mix AddJob calls with Wait() calls, nor
// should you call Wait() multiple times.
type JobMux struct {
	errors  map[string]error
	waiting map[string]bool
	ch      chan muxMsg
	guard   *sync.Mutex
}

// NewJobMux returns a JobMux, which muxes the output of several blocking JobFns.
func NewJobMux() *JobMux {
	j := &JobMux{
		errors:  make(map[string]error),
		waiting: make(map[string]bool),
		ch:      make(chan muxMsg, 1),
		guard:   &sync.Mutex{},
	}
	return j
}

type muxMsg struct {
	id  string
	err error
}

func (m *JobMux) msg(id string, err error) {
	m.ch <- muxMsg{id: id, err: err}
}

// JobFn runs a job and returns an error or nil.
type JobFn func() error

// AddJob add's the JobFn to the wait-group and starts it.
func (m *JobMux) AddJob(id string, fn JobFn) error {
	m.guard.Lock()
	defer m.guard.Unlock()
	if _, ok := m.waiting[id]; ok {
		return fmt.Errorf("Job '%s' already registered!", id)
	}
	m.waiting[id] = true
	go func() {
		log.Debugf("Starting job: %s", id)
		jerr := fn()
		log.Debugf("Got return from job: %s, %v", id, jerr)

		m.msg(id, jerr)
		log.Debugf("Sent msg for job: %s", id)
	}()
	return nil
}

// JobErrors captures all errors for a job.
type JobErrors struct {
	Errors map[string]error
}

// Error returns an error string
func (e JobErrors) Error() string {
	s := []string{"Job Errors:"}
	for id, err := range e.Errors {
		s = append(s, fmt.Sprintf("Job %s: %s", id, err))
	}
	return strings.Join(s, "\n")
}

// Wait waits until all of the jobs have returned
// TODO: This can only be called once since it consumes the mssages directly.
func (m *JobMux) Wait(ctx context.Context) error {
	wait := viper.GetBool("wait") // myriad running in wait-mode
	jerr := &JobErrors{Errors: make(map[string]error)}
Loop:
	for {
		m.guard.Lock()
		if !wait && len(m.waiting) == 0 {
			break Loop
		}
		if len(m.waiting) > 0 {
			log.Debugf("Waiting on %d things", len(m.waiting))
		}
		m.guard.Unlock()

		select {
		case msg := <-m.ch:
			m.guard.Lock()
			m.errors[msg.id] = msg.err
			delete(m.waiting, msg.id)
			log.Debugf("Got response for %s: %v", msg.id, msg.err)
			m.guard.Unlock()
		case <-ctx.Done():
			m.guard.Lock()
			for id := range m.waiting {
				jerr.Errors[id] = ctx.Err()
			}
			break Loop
		}
	}
	defer m.guard.Unlock()
	for id, err := range m.errors {
		if err != nil {
			log.WithError(err).WithField("job", id).Warn("Job failed")
			jerr.Errors[id] = err
		}
	}
	if len(jerr.Errors) > 0 {
		return jerr
	}
	return nil
}
