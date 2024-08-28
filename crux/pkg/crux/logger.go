package crux

import (
	"io"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/walrus"
)

// Usage: Either call GetLogger to fetch the default logger, or
// declare a variable with the crux interface (var myLog crux.Logger)
// and assign it something matching the interface.  StandardLogger and
// ExtendedLogger interfaces are available if you wish to avoid using
// extra features.

func init() {
	clog.Log = walrus.New()
}

// GetLogger returns a logger
func GetLogger() clog.Logger {
	return clog.Log
}

// GetLoggerW returns a logger operating on a given Writer
func GetLoggerW(w io.Writer) clog.Logger {
	xx := walrus.New()
	xx.SetOutput(w)
	return xx
}

// The buffered/retroactive logging isn't a common feature. But then
// only the code that creates the logger will be likely to set
// retroactive logging options.  Let the upper level code see the real
// logger, or at least a larger interface, while called functions are
// passed the simple Logger interface.

// ConfigurableLogger - Note, the "Logger" iface is in crux/log
// package to avoid cycles due to walrus references in this iface
// NOT FOR FUNCTION PARAMETER TYPES, use crux.Logger iface instead
type ConfigurableLogger interface {
	// TODO: Could drop this interface, or only use to validate the logger
	// returned by crux.GetLogger() meets this iface.

	clog.Logger // With() and other funcs.

	// Walrus specific
	SetOutput(w io.Writer)
	SetLevel(erixLevel walrus.Level)
	SetTagLevel(tag string, erixLevel walrus.Level)
	Tag(tagName string) *walrus.Entry
	Tags(tagNames []string) *walrus.Entry
	WithAttribsLevel(level walrus.Level, tagNames []string, fields walrus.Fields) *walrus.Entry
	WithAttribs(tagNames []string, fields walrus.Fields) *walrus.Entry
	Fields(fieldNames []string)
	//Log(level walrus.Level, tagName string, fields walrus.Fields, message string)
	EnableRetroLog(bufferEntries int, triggerLevel walrus.Level, bufferTagged bool) error
	IsRetroLogEnabled() bool
	IsTaggedRetroLogEnabled() bool
	DisableRetroLog()
	DumpRetroLog()

	// StartConfigWatcher()  // not implemented yet

	// Logrus & Walrus
	WithFields(fields walrus.Fields) *walrus.Entry
	WithError(err error) *walrus.Entry
	SetNoLock()
	Writer() *io.PipeWriter
	WriterLevel(level walrus.Level) *io.PipeWriter
}

// LoggerOld exposes the walrus.StdLogger interface to only expose golang standard
// logger functions.
type LoggerOld interface {

	// Functions extracted via:  cd pkg/walrus; grep func * | grep Logger | grep -v Assert |cut -d" " -f 4-

	// Walrus specific
	SetLevel(erixLevel walrus.Level)
	SetTagLevel(tag string, erixLevel walrus.Level)
	SetDefaultTagLevels(tag string, erixLevel walrus.LogLevel)
	Tag(tagName string) *walrus.Entry
	Tags(tagNames []string) *walrus.Entry
	WithAttribsLevel(level walrus.Level, tagNames []string, fields walrus.Fields) *walrus.Entry
	WithAttribs(tagNames []string, fields walrus.Fields) *walrus.Entry
	Fields(fieldNames []string)
	Log(level walrus.Level, tagName string, fields walrus.Fields, message string)
	EnableRetroLog(bufferEntries int, triggerLevel walrus.Level, bufferTagged bool) error
	IsRetroLogEnabled() bool
	IsTaggedRetroLogEnabled() bool
	DisableRetroLog()
	DumpRetroLog()

	// StartConfigWatcher()  // not implemented yet

	// Logrus & Walrus
	WithFields(fields walrus.Fields) *walrus.Entry
	WithError(err error) *walrus.Entry
	SetNoLock()
	Writer() *io.PipeWriter
	WriterLevel(level walrus.Level) *io.PipeWriter
}

// TODO: Remove StandardLogger Iface from this file.
// Should crux's logging interface include StandardLogger?
// Skaar asks if our logger can replace the standard logger, but for
// our own code, we don't intend to to use the standard functions.
// Nor is there a standard interface that is in common use, so that
// replacing loggers would be easy.

// Skaar's question seems more targeted as other people's packages,
// and doing some source editing to swap out their logger package for
// ours.

// StandardLogger - What the go standard logger gives you
type StandardLogger interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalln(args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Panicln(args ...interface{})

	Print(args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})
}
