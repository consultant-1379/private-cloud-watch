// +build !windows,!nacl,!plan9

package walrusSyslog

import (
	"fmt"

	"github.com/erixzone/crux/pkg/walrus"

	"log/syslog"
	"os"
)

// SyslogHook to send logs via syslog.
type SyslogHook struct {
	Writer        *syslog.Writer
	SyslogNetwork string
	SyslogRaddr   string
}

// NewSyslogHook creates a hook to be added to an instance of logger. This is called with
// `hook, err := NewSyslogHook("udp", "localhost:514", syslog.LOG_DEBUG, "")`
// `if err == nil { log.Hooks.Add(hook) }`
func NewSyslogHook(network, raddr string, priority syslog.Priority, tag string) (*SyslogHook, error) {
	w, err := syslog.Dial(network, raddr, priority, tag)
	return &SyslogHook{w, network, raddr}, err
}

// Fire returns something.
func (hook *SyslogHook) Fire(entry *walrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	switch entry.Level {
	case walrus.PanicLevel:
		return hook.Writer.Crit(line)
	case walrus.FatalLevel:
		return hook.Writer.Crit(line)
	case walrus.ErrorLevel:
		return hook.Writer.Err(line)
	case walrus.WarnLevel:
		return hook.Writer.Warning(line)
	case walrus.InfoLevel:
		return hook.Writer.Info(line)
	case walrus.DebugLevel:
		return hook.Writer.Debug(line)
	default:
		return nil
	}
}

// Levels returns something.
func (hook *SyslogHook) Levels() []walrus.Level {
	return walrus.AllLevels
}
