package walrus

import (
	"container/ring"
	"io"
	"os"
	"sync"
)

//Logger - root logging type
type Logger struct {
	// The logs are `io.Copy`'d to this in a mutex. It's common to set this to a
	// file, or leave it default which is `os.Stderr`. You can also set this to
	// something more adventorous, such as logging to Kafka.
	Out io.Writer
	// Hooks for the logger instance. These allow firing events based on logging
	// levels and log entries. For example, to send errors to an error tracking
	// service, log to StatsD or dump the core on fatal errors.
	Hooks LevelHooks
	// All log entries pass through the formatter before logged to Out. The
	// included formatters are `TextFormatter` and `JSONFormatter` for which
	// TextFormatter is the default. In development (when a TTY is attached) it
	// logs with colors, but to a file it wouldn't. You can easily implement your
	// own that implements the `Formatter` interface, see the `README` or included
	// formatters for examples.
	Formatter Formatter
	// The logging level the logger should log at. This is typically (and defaults
	// to) `walrus.Info`, which allows Info(), Warn(), Error() and Fatal() to be
	// logged.
	Level Level
	// Used to sync writing to the log. Locking is enabled by Default
	mu MutexWrap
	// Reusable empty entry
	entryPool sync.Pool

	// <radix> Supporting per-tag logging levels
	tagLevels map[string]Level /*_not_ protobuf LogLevel*/

	// <radix> Set of valid field names if not nil. Only for
	// lookup. Keys only, no values.
	validFields map[string]string

	// <radix> Retroactive logging from ringbuffer
	MsgRing     *ring.Ring
	ringTrigger Level
	ringTagged  bool
}

//MutexWrap - Make logging multi-thread safe
type MutexWrap struct {
	lock     sync.Mutex
	disabled bool
}

//Lock - Happy lint
func (mw *MutexWrap) Lock() {
	if !mw.disabled {
		mw.lock.Lock()
	}
}

//Unlock - Happy lint
func (mw *MutexWrap) Unlock() {
	if !mw.disabled {
		mw.lock.Unlock()
	}
}

//Disable - Happy lint
func (mw *MutexWrap) Disable() {
	mw.disabled = true
}

// New - Creates a new logger. Configuration should be set by changing `Formatter`,
// `Out` and `Hooks` directly on the default logger instance. You can also just
// instantiate your own:
//
//    var log = &Logger{
//      Out: os.Stderr,
//      Formatter: new(JSONFormatter),
//      Hooks: make(LevelHooks),
//      Level: walrus.DebugLevel,
//    }
//
// It's recommended to make this a global instance called `log`.
func New() *Logger {
	return &Logger{
		Out:       os.Stderr,
		Formatter: new(TextFormatter),
		Hooks:     make(LevelHooks),
		Level:     InfoLevel,
	}
}

func (logger *Logger) newEntry() *Entry {
	entry, ok := logger.entryPool.Get().(*Entry)
	if ok {
		return entry
	}
	return NewEntry(logger)
}

func (logger *Logger) releaseEntry(entry *Entry) {
	logger.entryPool.Put(entry)
}

//WithField - Adds a field to the log entry, note that it doesn't log until you call
// Debug, Print, Info, Warn, Fatal or Panic. It only creates a log entry.
// If you want multiple fields, use `WithFields`.
func (logger *Logger) WithField(key string, value interface{}) *Entry {
	entry := logger.newEntry()
	defer logger.releaseEntry(entry)
	return entry.WithField(key, value)
}

//WithFields - Adds a struct of fields to the log entry. All it does is call `WithField` for
// each `Field`.
func (logger *Logger) WithFields(fields Fields) *Entry {
	entry := logger.newEntry()
	defer logger.releaseEntry(entry)
	return entry.WithFields(fields)
}

//WithError - Add an error as single field to the log entry.  All it does is call
// `WithError` for the given `error`.
func (logger *Logger) WithError(err error) *Entry {
	entry := logger.newEntry()
	defer logger.releaseEntry(entry)
	return entry.WithError(err)
}

//Debugf - happy lint
func (logger *Logger) Debugf(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= DebugLevel {
		entry := logger.newEntry()
		entry.Debugf(format, args...)
		logger.releaseEntry(entry)
	}
}

//Infof - happy lint
func (logger *Logger) Infof(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= InfoLevel {
		entry := logger.newEntry()
		entry.Infof(format, args...)
		logger.releaseEntry(entry)
	}
}

//Printf - happy lint
func (logger *Logger) Printf(format string, args ...interface{}) {
	entry := logger.newEntry()
	entry.Printf(format, args...)
	logger.releaseEntry(entry)
}

//Warnf - happy lint
func (logger *Logger) Warnf(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= WarnLevel {
		entry := logger.newEntry()
		entry.Warnf(format, args...)
		logger.releaseEntry(entry)
	}
}

//Warningf - happy lint
func (logger *Logger) Warningf(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= WarnLevel {
		entry := logger.newEntry()
		entry.Warnf(format, args...)
		logger.releaseEntry(entry)
	}
}

//Errorf - happy lint
func (logger *Logger) Errorf(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= ErrorLevel {
		entry := logger.newEntry()
		entry.Errorf(format, args...)
		logger.releaseEntry(entry)
	}
}

//Fatalf - happy lint
func (logger *Logger) Fatalf(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= FatalLevel {
		entry := logger.newEntry()
		entry.Fatalf(format, args...)
		logger.releaseEntry(entry)
	}
	Exit(1)
}

//Panicf - happy lint
func (logger *Logger) Panicf(format string, args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= PanicLevel {
		entry := logger.newEntry()
		entry.Panicf(format, args...)
		logger.releaseEntry(entry)
	}
}

//Debug - happy lint
func (logger *Logger) Debug(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= DebugLevel {
		entry := logger.newEntry()
		entry.Debug(args...)
		logger.releaseEntry(entry)
	}
}

//Info - happy lint
func (logger *Logger) Info(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= InfoLevel {
		entry := logger.newEntry()
		entry.Info(args...)
		logger.releaseEntry(entry)
	}
}

//Print - happy lint
func (logger *Logger) Print(args ...interface{}) {
	entry := logger.newEntry()
	entry.Info(args...)
	logger.releaseEntry(entry)
}

//Warn - happy lint
func (logger *Logger) Warn(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= WarnLevel {
		entry := logger.newEntry()
		entry.Warn(args...)
		logger.releaseEntry(entry)
	}
}

//Warning - happy lint
func (logger *Logger) Warning(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= WarnLevel {
		entry := logger.newEntry()
		entry.Warn(args...)
		logger.releaseEntry(entry)
	}
}

//Error - happy lint
func (logger *Logger) Error(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= ErrorLevel {
		entry := logger.newEntry()
		entry.Error(args...)
		logger.releaseEntry(entry)
	}
}

//Fatal - happy lint
func (logger *Logger) Fatal(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= FatalLevel {
		entry := logger.newEntry()
		entry.Fatal(args...)
		logger.releaseEntry(entry)
	}
	Exit(1)
}

//Panic - happy lint
func (logger *Logger) Panic(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= PanicLevel {
		entry := logger.newEntry()
		entry.Panic(args...)
		logger.releaseEntry(entry)
	}
}

//Debugln - happy lint
func (logger *Logger) Debugln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= DebugLevel {
		entry := logger.newEntry()
		entry.Debugln(args...)
		logger.releaseEntry(entry)
	}
}

//Infoln - happy lint
func (logger *Logger) Infoln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= InfoLevel {
		entry := logger.newEntry()
		entry.Infoln(args...)
		logger.releaseEntry(entry)
	}
}

//Println - happy lint
func (logger *Logger) Println(args ...interface{}) {
	entry := logger.newEntry()
	entry.Println(args...)
	logger.releaseEntry(entry)
}

//Warnln - happy lint
func (logger *Logger) Warnln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= WarnLevel {
		entry := logger.newEntry()
		entry.Warnln(args...)
		logger.releaseEntry(entry)
	}
}

//Warningln - happy lint
func (logger *Logger) Warningln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= WarnLevel {
		entry := logger.newEntry()
		entry.Warnln(args...)
		logger.releaseEntry(entry)
	}
}

//Errorln - happy lint
func (logger *Logger) Errorln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= ErrorLevel {
		entry := logger.newEntry()
		entry.Errorln(args...)
		logger.releaseEntry(entry)
	}
}

//Fatalln - happy lint
func (logger *Logger) Fatalln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= FatalLevel {
		entry := logger.newEntry()
		entry.Fatalln(args...)
		logger.releaseEntry(entry)
	}
	Exit(1)
}

//Panicln - happy lint
func (logger *Logger) Panicln(args ...interface{}) {
	if logger.IsRetroLogEnabled() || logger.Level >= PanicLevel {
		entry := logger.newEntry()
		entry.Panicln(args...)
		logger.releaseEntry(entry)
	}
}

//SetNoLock - When file is opened with appending mode, it's safe to
//write concurrently to a file (within 4k message on Linux).
//In these cases user can choose to disable the lock.
func (logger *Logger) SetNoLock() {
	logger.mu.Disable()
}
