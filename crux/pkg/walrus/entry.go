package walrus

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"
)

var bufferPool *sync.Pool

func init() {
	bufferPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
}

// ErrorKey  Defines the key when adding errors using WithError.
var ErrorKey = "error"

// Entry  An entry is the final or intermediate Logrus logging entry. It contains all
// the fields passed with WithField{,s}. It's finally logged when Debug, Info,
// Warn, Error, Fatal or Panic is called on it. These objects can be reused and
// passed around as much as you wish to avoid field duplication.
type Entry struct {
	Logger *Logger

	// Contains all the fields set by the user.
	Data Fields

	// Time at which the log entry was created
	Time time.Time

	// Level the log entry was logged at: Debug, Info, Warn, Error, Fatal or Panic
	Level Level

	// Message passed to Debug, Info, Warn, Error, Fatal or Panic
	Message string

	// When formatter is called in entry.log(), an Buffer may be set to entry
	Buffer *bytes.Buffer

	// Tags set for this entry. (radix added feature)
	LogTags []string
}

// NewEntry - create a new logger Entry, capable of outputting one more log statements
func NewEntry(logger *Logger) *Entry {
	return &Entry{
		Logger: logger,
		// Default is three fields, give a little extra room
		Data: make(Fields, 5),
	}
}

// String  Returns the string representation from the reader and ultimately the
// formatter.
func (entry *Entry) String() (string, error) {
	serialized, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		return "", err
	}
	str := string(serialized)
	return str, nil
}

// WithError  Add an error as single field (using the key defined in ErrorKey) to the Entry.
func (entry *Entry) WithError(err error) *Entry {
	return entry.WithField(ErrorKey, err)
}

// WithField  Add a single field to the Entry.
func (entry *Entry) WithField(key string, value interface{}) *Entry {
	return entry.WithFields(Fields{key: value})
}

// BadFieldNameFlag  - Yes, someone might enter a non existent field anme
const BadFieldNameFlag string = "(Field Name Invalid)"

// WithFields   Add a map of fields to the Entry.
func (entry *Entry) WithFields(fields Fields) *Entry {
	data := make(Fields, len(entry.Data)+len(fields))
	for k, v := range entry.Data {
		data[k] = v
	}
	for k, v := range fields {
		//<radix>: check against validFields, alter keyname
		//for easy log scrubbing if no match found in valid list
		if entry.Logger.validFields != nil {
			if _, ok := entry.Logger.validFields[k]; !ok {
				k = k + BadFieldNameFlag
			}
		}
		data[k] = v
	}
	return &Entry{Logger: entry.Logger, Data: data, LogTags: entry.LogTags, Level: entry.Level}
}

// logAndBuffer
// If retrologging is enabled, put the entry into retro buffer.
// Optionally output the entry first, controlled by the output argument.
//
func (entry Entry) logAndBuffer(level Level, msg string, output bool) {
	//fmt.Printf("##> logAndBuffer  [%s]  output %+v\n", msg, output)
	// set entry values here instead of log() so buffered entries
	// display correct info if they're ever retro-logged.
	entry.Time = time.Now()
	entry.Level = level
	entry.Message = msg

	if level <= entry.Logger.ringTrigger {
		entry.Logger.DumpRetroLog()
	}

	if output {
		entry.log()
	}

	l := entry.Logger
	if l.IsRetroLogEnabled() {
		//fmt.Printf("###> Adding %s to ring\n", entry.Message)
		// Add entry to retrolog
		l.MsgRing.Value = entry
		l.MsgRing = l.MsgRing.Next()
	}
}

// This function is not declared with a pointer value because otherwise
// race conditions will occur when using multiple goroutines
func (entry Entry) log() {
	var buffer *bytes.Buffer
	level := entry.Level

	if err := entry.Logger.Hooks.Fire(level, &entry); err != nil {
		entry.Logger.mu.Lock()
		fmt.Fprintf(os.Stderr, "Failed to fire hook: %v\n", err)
		entry.Logger.mu.Unlock()
	}

	buffer = bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)
	entry.Buffer = buffer
	serialized, err := entry.Logger.Formatter.Format(&entry)
	entry.Buffer = nil
	if err != nil {
		entry.Logger.mu.Lock()
		fmt.Fprintf(os.Stderr, "Failed to obtain reader, %v\n", err)
		entry.Logger.mu.Unlock()
	} else {
		entry.Logger.mu.Lock()
		_, err = entry.Logger.Out.Write(serialized)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
		}
		entry.Logger.mu.Unlock()
	}

	// To avoid Entry#log() returning a value that only would make sense for
	// panic() to use in Entry#Panic(), we avoid the allocation by checking
	// directly here.
	if level <= PanicLevel {
		panic(&entry)
	}
}

// tagFormat - Matched tags are added as prefix to log lines.  This is
// the format.
func tagFormat(tag string) string {
	// TODO: what is preferred formatting
	// to use for tag header?  Don't want
	// to make log parsing more difficult
	// as could happen with log line
	// prefixes than only show up for
	// tagged statements
	//
	// Make tag prefix formatting configurable. Via
	// SetFormatter()?  Get it to comply to logfmt format?

	return "[tag=" + tag + "] "
}

// When should tagged statements output if
// no tagLevel threshold set for it?
var defaultTagThreshold = ErrorLevel

// logIfLevel - radix: Check if this level should log or not, due to a matching tag or
func (entry *Entry) logIfLevel(statementLevel Level, args ...interface{}) {
	// Passing in statementLevel is odd. Entry has a .Level memeber.
	// Should really just set this entry's  .Level and remove statementLevel Param.

	// If a tag is defined, use the tag's configured level, not the logger global level.
	if entry.LogTags != nil {
		for _, tag := range entry.LogTags {
			tagThresh, ok := entry.Logger.tagLevels[tag]
			if !ok {
				tagThresh = defaultTagThreshold
			}
			dbg2("statement level=%v tag=%s  tag-output-level=%v [%t]\n", statementLevel, tag, tagThresh, ok)
			if tagThresh >= statementLevel {
				//Could do tag name output at end, like structured logging
				// but for now it's before the message.
				tagPrefix := []interface{}{tagFormat(tag)}
				args = append(tagPrefix, args...)
				entry.logAndBuffer(statementLevel, fmt.Sprint(args...), true)
				return
			}
		}
		if entry.Logger.IsTaggedRetroLogEnabled() {
			entry.logAndBuffer(statementLevel, fmt.Sprint(args...), false)
		}
		return // Tagged entry, but didn't output
	}
	// No tags defined for entry, back to traditional non-tagged case
	output := false
	//   logger thresh       statement level
	if entry.Logger.Level >= statementLevel {
		output = true
	}
	entry.logAndBuffer(statementLevel, fmt.Sprint(args...), output)
}

// logFormattedIfLevel - radix: Like logIfLevel,but for *f() methods
func (entry *Entry) logFormattedIfLevel(statementLevel Level, format string, args ...interface{}) {
	//fmt.Printf("Tags set=  %+v\n", entry.LogTags)
	if entry.LogTags != nil {
		for _, tag := range entry.LogTags {
			tagThresh, ok := entry.Logger.tagLevels[tag]
			if !ok {
				tagThresh = defaultTagThreshold
			}

			if tagThresh >= statementLevel {
				format = tagFormat(tag) + format
				entry.logAndBuffer(statementLevel, fmt.Sprintf(format, args...), true)
				return
			}
		}
		if entry.Logger.IsTaggedRetroLogEnabled() {
			entry.logAndBuffer(statementLevel, fmt.Sprintf(format, args...), false)
		}
		return
	}
	// No tags defined for entry, back to traditional non-tagged case
	output := false
	if entry.Logger.Level >= statementLevel {
		output = true
	}
	entry.logAndBuffer(statementLevel, fmt.Sprintf(format, args...), output)
}

//Debug - Happy Lint
func (entry *Entry) Debug(args ...interface{}) {
	entry.logIfLevel(DebugLevel, args...)
}

//Print - Happy Lint
func (entry *Entry) Print(args ...interface{}) {
	entry.Info(args...)
}

//Info - Happy Lint
func (entry *Entry) Info(args ...interface{}) {
	entry.logIfLevel(InfoLevel, args...)
}

//Warn - Happy Lint
func (entry *Entry) Warn(args ...interface{}) {
	entry.logIfLevel(WarnLevel, args...)
}

//Warning - Happy Lint
func (entry *Entry) Warning(args ...interface{}) {
	entry.Warn(args...)
}

//Error- Happy Lint
func (entry *Entry) Error(args ...interface{}) {
	entry.logIfLevel(ErrorLevel, args...)
}

//Fatal - Happy Lint
func (entry *Entry) Fatal(args ...interface{}) {
	entry.logIfLevel(FatalLevel, args...)
	Exit(1)
}

//Panic - Happy Lint
func (entry *Entry) Panic(args ...interface{}) {
	entry.logIfLevel(PanicLevel, args...)
	panic(fmt.Sprint(args...))
}

// Entry Printf family functions

//Debugf - Happy Lint
func (entry *Entry) Debugf(format string, args ...interface{}) {
	entry.logFormattedIfLevel(DebugLevel, format, args...)
}

//Infof - Happy Lint
func (entry *Entry) Infof(format string, args ...interface{}) {
	entry.logFormattedIfLevel(InfoLevel, format, args...)
}

//Printf - Happy Lint
func (entry *Entry) Printf(format string, args ...interface{}) {
	entry.Infof(format, args...)
}

//Warnf - Happy Lint
func (entry *Entry) Warnf(format string, args ...interface{}) {
	entry.logFormattedIfLevel(WarnLevel, format, args...)
}

//Warningf - Happy Lint
func (entry *Entry) Warningf(format string, args ...interface{}) {
	entry.Warnf(format, args...)
}

//Errorf - Happy Lint
func (entry *Entry) Errorf(format string, args ...interface{}) {
	entry.logFormattedIfLevel(ErrorLevel, format, args...)
}

//Fatalf - Happy Lint
func (entry *Entry) Fatalf(format string, args ...interface{}) {
	entry.logFormattedIfLevel(FatalLevel, format, args...)
	Exit(1)
}

//Panicf - Happy Lint
func (entry *Entry) Panicf(format string, args ...interface{}) {
	entry.logFormattedIfLevel(PanicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Entry Println family functions
// FIXME: DMB:  add tag checking logic to *ln() methods

//Debugln - Happy Lint
func (entry *Entry) Debugln(args ...interface{}) {
	if entry.Logger.Level >= DebugLevel {
		entry.Debug(entry.sprintlnn(args...))
	}
}

//Infoln - Happy Lint
func (entry *Entry) Infoln(args ...interface{}) {
	if entry.Logger.Level >= InfoLevel {
		entry.Info(entry.sprintlnn(args...))
	}
}

//Println - Happy Lint
func (entry *Entry) Println(args ...interface{}) {
	entry.Infoln(args...)
}

//Warnln - Happy Lint
func (entry *Entry) Warnln(args ...interface{}) {
	if entry.Logger.Level >= WarnLevel {
		entry.Warn(entry.sprintlnn(args...))
	}
}

//Warningln - Happy Lint
func (entry *Entry) Warningln(args ...interface{}) {
	entry.Warnln(args...)
}

//Errorln - Happy Lint
func (entry *Entry) Errorln(args ...interface{}) {
	if entry.Logger.Level >= ErrorLevel {
		entry.Error(entry.sprintlnn(args...))
	}
}

//Fatalln - Happy Lint
func (entry *Entry) Fatalln(args ...interface{}) {
	if entry.Logger.Level >= FatalLevel {
		entry.Fatal(entry.sprintlnn(args...))
	}
	Exit(1)
}

//Panicln - Happy Lint
func (entry *Entry) Panicln(args ...interface{}) {
	if entry.Logger.Level >= PanicLevel {
		entry.Panic(entry.sprintlnn(args...))
	}
}

// Sprintlnn => Sprint no newline. This is to get the behavior of how
// fmt.Sprintln where spaces are always added between operands, regardless of
// their type. Instead of vendoring the Sprintln implementation to spare a
// string allocation, we do the simplest thing.
func (entry *Entry) sprintlnn(args ...interface{}) string {
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}
