// Copyright 2016 Ericsson AB All Rights Reserved.
// Contributors:
//     Jared Bischof
//     Scott Devoid

/*
Package log wraps github.com/Sirupsen/logrus (which itself
implements all of the package functions of the standard 'log'
package. This package adds fields to the log indicating what
function called the log message. It also indicates the package,
filename and line-number of that function call.
*/
package log

import (
	"errors"
	"io"
	stdlib "log"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
)

// For this implementation we want to use skip = 2 to correctly
// record the location of where the log function was called.
const skip = 2
const callerFunc = "Func"
const callerFile = "File"
const callerLine = "Line"

var verbosity = 1
var log *logrus.Logger

func init() {
	log = logrus.New()

	// Configure 3rd party logging ineterfaces to use our wrapper.
	stdlib.SetOutput(newStdlibWrapper(log))
}

// Entry wraps logrus.Entry and embeds common fields into the
// entry when the final 'log event' occurs (e.g. e.Errorf()).
type Entry struct {
	*logrus.Entry
}

// SetVerbosity sets the verbosity level of logging (default=1)
// verbosity = 0 will add no callerFunc field to log messages
// verbosity = 1 will add an abbreviated callerFunc field to logs
// verbosity = 2 will add the full callerFunc field, a callerFile
// field and a callerLine field to the log messages
func SetVerbosity(v int) error {
	if v == 0 || v == 1 || v == 2 {
		verbosity = v
		return nil
	}
	return errors.New("Invalid verbosity value (allowed range: 0-2)")
}

// SetOutput sets the location to which log messages will be sent.
func SetOutput(out io.Writer) {
	log.Out = out
}

// SetLevel sets the level at which messages should be logged.
func SetLevel(level logrus.Level) {
	log.Level = level
}

// SetFormatter will assign a formatter to use when printing logs.
func SetFormatter(formatter logrus.Formatter) {
	log.Formatter = formatter
}

// WithField returns a pointer to a logrus.Entry struct with the
// provided key/value pair and the caller fields added to it.
func WithField(key string, value interface{}) *Entry {
	e := &Entry{log.WithField(key, value)}
	return e.a()
}

// WithFields returns a pointer to a logrus.Entry struct with the
// provided logrus.Fields and the caller fields added to it.
func WithFields(fields logrus.Fields) *Entry {
	lfields := logrus.Fields{}
	for k, v := range fields {
		lfields[k] = v
	}
	return &Entry{log.WithFields(lfields)}
}

// WithError returns a pointer to a logrus.Entry struct with the
// provided error and the caller fields added to it.
func WithError(err error) *Entry {
	e := &Entry{log.WithField(logrus.ErrorKey, err)}
	return e.a()
}

// Debugf logs the message at the Debug level with caller fields.
func Debugf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Debugf(format, args...)
}

// Infof logs the message at the Info level with caller fields.
func Infof(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Infof(format, args...)
}

// Printf logs the message at the Info level with caller fields.
func Printf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Printf(format, args...)
}

// Warnf logs the message at the Warning level with caller fields.
func Warnf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Warnf(format, args...)
}

// Warningf logs the message at the Warning level with caller fields.
func Warningf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Warningf(format, args...)
}

// Errorf logs the message at the Error level with caller fields.
func Errorf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Errorf(format, args...)
}

// Fatalf logs the message at the Fatal level with caller fields.
func Fatalf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Fatalf(format, args...)
}

// Panicf logs the message at the Panic level with caller fields.
func Panicf(format string, args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Panicf(format, args...)
}

// Debug logs the message at the Debug level with caller fields.
func Debug(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Debug(args...)
}

// Info logs the message at the Info level with caller fields.
func Info(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Info(args...)
}

// Print logs the message at the Info level with caller fields.
func Print(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Print(args...)
}

// Warn logs the message at the Warning level with caller fields.
func Warn(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Warn(args...)
}

// Warning logs the message at the Warning level with caller fields.
func Warning(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Warning(args...)
}

// Error logs the message at the Error level with caller fields.
func Error(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Error(args...)
}

// Fatal logs the message at the Fatal level with caller fields.
func Fatal(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Fatal(args...)
}

// Panic logs the message at the Panic level with caller fields.
func Panic(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Panic(args...)
}

// Debugln logs the message at the Debug level with caller fields.
func Debugln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Debugln(args...)
}

// Infoln logs the message at the Info level with caller fields.
func Infoln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Infoln(args...)
}

// Println logs the message at the Info level with caller fields.
func Println(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Println(args...)
}

// Warnln logs the message at the Warning level with caller fields.
func Warnln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Warnln(args...)
}

// Warningln logs the message at the Warning level with caller fields.
func Warningln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Warningln(args...)
}

// Errorln logs the message at the Error level with caller fields.
func Errorln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Errorln(args...)
}

// Fatalln logs the message at the Fatal level with caller fields.
func Fatalln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Fatalln(args...)
}

// Panicln logs the message at the Panic level with caller fields.
func Panicln(args ...interface{}) {
	log.WithFields(addCaller(logrus.Fields{})).Panicln(args...)
}

// Debug logs the message at the Debug level with caller fields.
func (e *Entry) Debug(args ...interface{}) { e.a().Entry.Debug(args...) }

// Debugf logs the message at the Debug level with caller fields.
func (e *Entry) Debugf(format string, args ...interface{}) { e.a().Entry.Debugf(format, args...) }

// Debugln logs the message at the Debug level with caller fields.
func (e *Entry) Debugln(args ...interface{}) { e.a().Entry.Debugln(args...) }

// Error logs the message at the Error level with caller fields.
func (e *Entry) Error(args ...interface{}) { e.a().Entry.Error(args...) }

// Errorf logs the message at the Error level with caller fields.
func (e *Entry) Errorf(format string, args ...interface{}) { e.a().Entry.Errorf(format, args...) }

// Errorln logs the message at the Error level with caller fields.
func (e *Entry) Errorln(args ...interface{}) { e.a().Entry.Errorln(args...) }

// Fatal logs the message at the Fatal level with caller fields.
func (e *Entry) Fatal(args ...interface{}) { e.a().Entry.Fatal(args...) }

// Fatalf logs the message at the Fatal level with caller fields.
func (e *Entry) Fatalf(format string, args ...interface{}) { e.a().Entry.Fatalf(format, args...) }

// Fatalln logs the message at the Fatal level with caller fields.
func (e *Entry) Fatalln(args ...interface{}) { e.a().Entry.Fatalln(args...) }

// Info logs the message at the Info level with caller fields.
func (e *Entry) Info(args ...interface{}) { e.a().Entry.Info(args...) }

// Infof logs the message at the Info level with caller fields.
func (e *Entry) Infof(format string, args ...interface{}) { e.a().Entry.Infof(format, args...) }

// Infoln logs the message at the Info level with caller fields.
func (e *Entry) Infoln(args ...interface{}) { e.a().Entry.Infoln(args...) }

// Panic logs the message at the Panic level with caller fields.
func (e *Entry) Panic(args ...interface{}) { e.a().Entry.Panic(args...) }

// Panicf logs the message at the Panic level with caller fields.
func (e *Entry) Panicf(format string, args ...interface{}) { e.a().Entry.Panicf(format, args...) }

// Panicln logs the message at the Panic level with caller fields.
func (e *Entry) Panicln(args ...interface{}) { e.a().Entry.Panicln(args...) }

// Warn logs the message at the Warn level with caller fields.
func (e *Entry) Warn(args ...interface{}) { e.a().Entry.Warn(args...) }

// Warnf logs the message at the Warn level with caller fields.
func (e *Entry) Warnf(format string, args ...interface{}) { e.a().Entry.Warnf(format, args...) }

// Warnln logs the message at the Warn level with caller fields.
func (e *Entry) Warnln(args ...interface{}) { e.a().Entry.Warnln(args...) }

// WithField returns an Entry with the field added.
func (e *Entry) WithField(key string, val interface{}) *Entry {
	return &Entry{e.Entry.WithField(key, val)}
}

// WithFields returns an Entry with the fields added.
func (e *Entry) WithFields(fields logrus.Fields) *Entry {
	return &Entry{e.Entry.WithFields(fields)}
}

// WithError returns an Entry with the error added
func (e *Entry) WithError(err error) *Entry {
	f := &Entry{e.Entry.WithError(err)}
	return f.a()
}

func addCaller(fields logrus.Fields) logrus.Fields {
	if verbosity == 1 {
		pc := make([]uintptr, skip)
		runtime.Callers(skip, pc)
		f := runtime.FuncForPC(pc[skip-1] - 1)
		fields[callerFunc] = getAbbrevPath(f.Name())
	} else if verbosity == 2 {
		pc := make([]uintptr, skip)
		runtime.Callers(skip, pc)
		f := runtime.FuncForPC(pc[skip-1] - 1)
		file, line := f.FileLine(pc[skip-1] - 1)
		fields[callerFunc] = f.Name()
		fields[callerFile] = file
		fields[callerLine] = line
	}
	return fields
}

// This is a helper function like addCaller except that it works on entries rather than fields.
func (e *Entry) a() *Entry {
	fields := make(logrus.Fields)
	if verbosity == 1 {
		pc := make([]uintptr, skip)
		runtime.Callers(skip, pc)
		f := runtime.FuncForPC(pc[skip-1] - 1)
		fields[callerFunc] = getAbbrevPath(f.Name())
	} else if verbosity == 2 {
		pc := make([]uintptr, skip)
		runtime.Callers(skip, pc)
		f := runtime.FuncForPC(pc[skip-1] - 1)
		file, line := f.FileLine(pc[skip-1] - 1)
		fields[callerFunc] = f.Name()
		fields[callerFile] = file
		fields[callerLine] = line
	}
	return e.WithFields(fields)
}

func getAbbrevPath(path string) string {
	res := ""
	strSinceSlash := ""
	for _, c := range path {
		s := string(c)
		strSinceSlash += s
		if s == "/" {
			res += s
			strSinceSlash = ""
		} else if len(strSinceSlash) == 1 {
			res += s
		}
	}
	res += strSinceSlash[1:]
	return res
}

// stdlibWrapper wraps a Logger and allows it to be passed to the
// stdlib logger's SetOutput as an io.Writer.
type stdlibWrapper struct{ *logrus.Logger }

// newStdlibWrapper returns a new stdlibWrapper around the passed
// logrus.Logger. It's designed to be passed to log.SetOutput.
func newStdlibWrapper(logger *logrus.Logger) io.Writer {
	return stdlibWrapper{Logger: logger}
}

func (a stdlibWrapper) Write(p []byte) (int, error) {
	a.Logger.Print(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}
