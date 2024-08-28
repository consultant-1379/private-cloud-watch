package test

import (
	"io/ioutil"

	"github.com/erixzone/crux/pkg/walrus"
)

// Hook is a hook designed for dealing with logs in test scenarios.
type Hook struct {
	Entries []*walrus.Entry
}

// NewGlobal installs a test hook for the global logger.
func NewGlobal() *Hook {

	hook := new(Hook)
	walrus.AddHook(hook)

	return hook

}

// NewLocal installs a test hook for a given local logger.
func NewLocal(logger *walrus.Logger) *Hook {

	hook := new(Hook)
	logger.Hooks.Add(hook)

	return hook

}

// NewNullLogger creates a discarding logger and installs the test hook.
func NewNullLogger() (*walrus.Logger, *Hook) {

	logger := walrus.New()
	logger.Out = ioutil.Discard

	return logger, NewLocal(logger)

}

// Fire returns nothing.
func (t *Hook) Fire(e *walrus.Entry) error {
	t.Entries = append(t.Entries, e)
	return nil
}

// Levels returns something.
func (t *Hook) Levels() []walrus.Level {
	return walrus.AllLevels
}

// LastEntry returns the last entry that was logged or nil.
func (t *Hook) LastEntry() (l *walrus.Entry) {

	if i := len(t.Entries) - 1; i >= 0 {
		return t.Entries[i]
	}
	return nil
}

// Reset removes all Entries from this test hook.
func (t *Hook) Reset() {
	t.Entries = make([]*walrus.Entry, 0)
}
