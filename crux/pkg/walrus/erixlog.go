package walrus

import (
	"container/ring"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/erixzone/crux/pkg/clog"
)

// The level to use if Log() or With() calls don't specify a severity.
var defaultStatementLevel = InfoLevel

// SetDefaultStatementLevel - Set the level of Log() and With()
// statements if no SEV specified.
func (l *Logger) SetDefaultStatementLevel(defLevel Level) {
	defaultStatementLevel = defLevel
}

//SetOutput - Logrus's Logger.Out is public, but need method for interfaces.
func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	l.Out = out
	l.mu.Unlock()
}

//  Some Level setting functions on both Logger and Entry to make a
//  common interface possible.  Versus using SetLevel(..), which
//  exposes the Level type, interfering with making an interface.

// SetDebug - happy lint
func (l *Logger) SetDebug() { l.SetLevel(DebugLevel) }

// SetInfo - happy lint
func (l *Logger) SetInfo() { l.SetLevel(InfoLevel) }

// SetWarn - happy lint
func (l *Logger) SetWarn() { l.SetLevel(WarnLevel) }

// SetDebug - happy lint
func (e *Entry) SetDebug() { e.Logger.SetLevel(DebugLevel) }

// SetInfo - happy lint
func (e *Entry) SetInfo() { e.Logger.SetLevel(InfoLevel) }

// SetWarn - happy lint
func (e *Entry) SetWarn() { e.Logger.SetLevel(WarnLevel) }

// This file contains new methods added to walrus to support
// Tagged logging, and remote changes to logging level via gRPC.

// SetLevel - Set the underlying walrus level,
func (l *Logger) SetLevel(erixLevel Level /*walrus level*/) {
	l.mu.Lock()
	//l.Level = ToLogrusLevel(erixLevel)
	l.Level = erixLevel
	l.mu.Unlock()

	// log any log level change so log reviewers know if
	// missing log statements are meaningful or not.
	msg := fmt.Sprintf("Logging level changed to (%s)", erixLevel)
	l.Info(msg)
}

// SetTagLevel - Set the logging level for a particular tag name
func (l *Logger) SetTagLevel(tag string, erixLevel Level /*protobuf level)*/) {
	l.mu.Lock()

	if l.tagLevels == nil {
		l.tagLevels = make(map[string]Level)
	}
	l.tagLevels[tag] = erixLevel //ToLogrusLevel(erixLevel)
	l.mu.Unlock()

	// log any log level change so log reviewers know if
	// missing log statements are meaningful or not.
	msg := fmt.Sprintf("Tag (%s) logging level changed to (%s)", tag, erixLevel)
	l.Info(msg)
}

// SetTagLevel - Set the logging level for a particular tag name
func (e *Entry) SetTagLevel(tag string, erixLevel Level /*protobuf level)*/) {
	e.Logger.SetTagLevel(tag, erixLevel)
}

// GetTags - Return slice of all tags set for this entry. Could be nil.
func (e *Entry) GetTags() []string {
	return e.LogTags
}

// Tag -  Sets a single tag. Any existing tags will be overwritten
func (l *Logger) Tag(tagName string) *Entry {
	entry := l.newEntry()
	defer l.releaseEntry(entry)
	e := entry.Tag(tagName)
	return e

}

// Tag - Sets a single tag. Any existing tags will be overwritten
func (e *Entry) Tag(tagName string) *Entry {
	tag := []string{tagName}
	return &Entry{Logger: e.Logger, Data: e.Data, LogTags: tag, Level: e.Level}
}

// Tags - any existing tags will not be copied to new entry.
func (l *Logger) Tags(tagNames []string) *Entry {
	entry := l.newEntry()
	defer l.releaseEntry(entry)
	// Would be better for newEntry() to take fields and tags as
	// initializers or not return a new Entry, but set tags on
	// existing one. (differs from Fields always creating a new
	// Entry)
	e := entry.Tags(tagNames)
	return e

}

// Tags - any existing tags will be overwritten
func (e *Entry) Tags(tagNames []string) *Entry {
	return &Entry{Logger: e.Logger, Data: e.Data, LogTags: tagNames, Level: e.Level}
}

// WithAttribsLevel - Create a entry with level, tags, and fields with one method call.
// Versus calling .Tags(), .Fields() and direct setting of Level separately.
func (l *Logger) WithAttribsLevel(level Level, tagNames []string, fields Fields) *Entry {
	e := l.Tags(tagNames)
	// Normally set on output in Entry.log(), but we're pre-determining here.
	e.Level = level
	e.Data = fields
	return e
}

// WithAttribs - Create an entry with tags and fields.
func (l *Logger) WithAttribs(tagNames []string, fields Fields) *Entry {
	e := l.Tags(tagNames)
	e.Data = fields
	return e
}

// With () -
// - this pkg can know about crux.logger
// - can
func (l *Logger) With(keyvals ...interface{}) clog.Logger {
	entry := l.newEntry()
	entry.Level = defaultStatementLevel
	return entry.With(keyvals...)
}

// With - Entry version.
//  TODO: extract common logic from Entry.Log, Entry.With
func (e *Entry) With(keyvals ...interface{}) clog.Logger {

	// Read varargs
	// For each one, check if TagKey or SeverityKey
	dbg(1, ">>><Entry.With> len keyvals %d\n", len(keyvals))
	fields := make(Fields, len(keyvals)/2)
	lastParam := len(keyvals) - 1
	for i := 0; i < len(keyvals); i += 2 {
		a := keyvals[i]
		// Special case of lass odd parameter is the log message.
		// The key is implied here.
		if i == lastParam {
			dbg(1, "  >>> Odd number of parameters. Last parameter [%v] ignored\n", a)
			// TODO: We should let the user know they
			// misused With(), but it's fragile to have
			// an error dependent on the number of varargs.
			goto ParamDone
		}

		b := keyvals[i+1]

		// Magic key name handling
		if a == SeverityKey {
			sev := b
			dbg(2, "  >>>Found SEV, with value %s \n", sev)

			newLevel, err := ParseLevel(sev.(string))
			if err != nil {
				fmt.Printf("Error: Invalid severity level %s : %s\n", sev, err)
				// TODO: Could return as error, but who
				// checks return value of logging
				// statements?
			} else {
				// level to output this log message at
				e.Level = newLevel
			}
			continue
			// Not printing the Level/Severity as a field
			// since underlying logger will display it.
		}

		// More magic key name handling
		if a == TagKey {
			tags := b
			dbg2("  >>>Found TAG, with value %+v \n", tags)
			// Tags should be  slice of string, need
			if tagsSlice, ok := tags.([]string); ok {
				//FIXME: not slice of string.  Then array?  convert to slice for .Tags()?
				dbg2(" [Tag value(s) good:: %v ] \n ", tags)
				e = e.Tags(tagsSlice)
			} else {
				dbg2(" Error: [Tag value(s) bad:: %v ] \n ", tags)
				// TODO: okay to return nil?
				return nil

			}

			continue
		}

		// Handle normal key/value pairs (vs special keys)
		if strval, ok := a.(string); ok {
			fields[strval] = b
			dbg(3, " [log key good: %v] \n ", a)
		} else {
			dbg(3, " [log key failure: %v is not a string] \n ", a)
			return nil
			// TODO: return error, or insert a log message
			// with this error (with ErrorLevel?) and
			// exit.  So someone sees it.  This varags
			// system could lead to silent failures.
		}
	}
ParamDone:
	// Tack on the fields we collected earlier
	if len(fields) != 0 {
		dbg(2, "  Collected fields: %v\n", fields)
		e = e.WithFields(fields)
		dbg(2, "  Fields after append: %v\n", e.Data)
	}

	return e
}

// Fields - Optionally configure the set of valid fields that can be
// used in WithFields(), etc.  Only marginally useful since misspelled
// field names won't be caught until run time, and we don't want to
// error.  At best we can flag unexpected names to make them easy to
// find in logs.  TODO: much more useful if done in a pre-compile-time
// checker
// NOTE: this erases any fields set by previous calls to Fields()
func (l *Logger) Fields(fieldNames []string) {
	// Include the built-in fieldname used by WithError
	names := append(fieldNames, ErrorKey)

	if l.validFields == nil {
		l.validFields = make(map[string]string, len(names))
	}

	for _, nm := range names {
		// key doesn't matter. Only for lookup
		l.validFields[nm] = ""
	}

}

// Special keys for Log() function that signal specific actions

// TagKey - Associated Value for this key  will be a slice of strings holding Tag names to apply
const TagKey = "TAG"

// SeverityKey - Associated value for this key will be a String specifying
// what log severity this is. Currently, severity strings are same as logrus's
// level strings.  ParseLevel(string) can be used to check validity.
const SeverityKey = "SEV"

//Log - gokit style TODO: To return an error or not?  What are the odds anyone would check
// it

// Who logs the logger?
func dbg(detail uint8, format string, args ...interface{}) {
	var detailLevel uint8 = 3
	// 1=low, 2=mid , 3=high detail
	if !true && (detail <= detailLevel) { /* TURN OFF HERE */
		fmt.Printf(format, args...)
	}
}

func dbg2(format string, args ...interface{}) {

	pc, _, line, ok := runtime.Caller(1)
	//pc, file, line, ok := runtime.Caller(1)
	if !ok {
		fmt.Println("dev-dbg Caller failed to get all info")
	}
	funcName := runtime.FuncForPC(pc).Name()
	sl := strings.Split(funcName, ".")
	funcName = sl[len(sl)-1]

	// TODO: Allow white-listing. Only show stuff from functions
	// in a list.

	//  Name returned by Caller() is similar to:
	//        github.com/erixzone/crux/pkg/walrus.(*Entry).Log
	// white-listing by a string, say "Log" means you
	// would get both Entry.Log and Logger.Log To limit to one or
	// the other, use a regex mech, or..... just multiple string
	// checks (slice)

	info := fmt.Sprintf(format, args...)

	if !true { /*TURN OFF HERE*/

		//log.Printf("dev-dbg  %s:%d %v", file, line, info)
		fmt.Printf("dev-dbg:%s() %d>  %v\n", funcName, line, info)
	}

}

// Logf - sugar
func (l *Logger) Logf(keyvals ...interface{}) {
	l.Log(append([]interface{}{clog.LogfDetail, clog.LogdFine}, keyvals...)...)
}

// Logm - sugar
func (l *Logger) Logm(keyvals ...interface{}) {
	l.Log(append([]interface{}{clog.LogfDetail, clog.LogdMedium}, keyvals...)...)
}

// Logc - sugar
func (l *Logger) Logc(keyvals ...interface{}) {
	l.Log(append([]interface{}{clog.LogfDetail, clog.LogdCoarse}, keyvals...)...)
}

// Logi - sugar
func (l *Logger) Logi(keyvals ...interface{}) {
	l.Log(append([]interface{}{clog.LogfDetail, clog.LogdInfo}, keyvals...)...)
}

// Logf - sugar
func (e *Entry) Logf(keyvals ...interface{}) {
	e.Log(append([]interface{}{clog.LogfDetail, clog.LogdFine}, keyvals...)...)
}

// Logm - sugar
func (e *Entry) Logm(keyvals ...interface{}) {
	e.Log(append([]interface{}{clog.LogfDetail, clog.LogdMedium}, keyvals...)...)
}

// Logc - sugar
func (e *Entry) Logc(keyvals ...interface{}) {
	e.Log(append([]interface{}{clog.LogfDetail, clog.LogdCoarse}, keyvals...)...)
}

// Logi - sugar
func (e *Entry) Logi(keyvals ...interface{}) {
	e.Log(append([]interface{}{clog.LogfDetail, clog.LogdInfo}, keyvals...)...)
}

// Log - See Entry.Log below for description.
func (l *Logger) Log(keyvals ...interface{}) clog.Logger {

	entry := l.newEntry()

	defer l.releaseEntry(entry)
	dbg(2, ">>><Logger.Log> len keyvals %d  level:%s with: %v\n", len(keyvals), entry.Level, entry.Data)
	entry.Level = defaultStatementLevel
	entry.Log(keyvals...)
	return entry
}

// Log - A multi-purpose logging method.
// Arguments should be pairs of field,value for structured logging, except....
// - "SEV" in a field is used to set logging level., and values must
// be ERROR,DEBUG, etc.
// - A "TAGS" field will expect a slice of tags in the value.
// - A "nil" as field name will interpret the rest of the values to
// arguments of type (string, ...args) for a printf
// - If the last field is also the last argument, it will be used as log message.
func (e *Entry) Log(keyvals ...interface{}) clog.Logger {

	var printFunc func(args ...interface{})

	// TODO: Decide what to do in code below when logic for field
	// value or position fails.  This is price we pay for
	// - Taking strings instead of typed values
	// - Assigning special meanings
	// - Positional significance.

	var message = ""
	dbg(2, ">>><Entry.Log> len keyvals %d  level:%s with: %v\n", len(keyvals), e.Level, e.Data)
	fields := make(Fields, len(keyvals)/2)
	lastParam := len(keyvals) - 1

	// Read varargs
	// For each one, check if TagKey or SeverityKey
	// If one left over, use as standalone message.

	for i := 0; i < len(keyvals); i += 2 {
		fld := keyvals[i] // Field

		// Special case - Last odd parameter is the log message.
		// The key is implied here
		if i == lastParam {
			if strval, ok := fld.(string); ok {
				message = strval
				dbg(3, " [Message good: %v] \n ", fld)
			} else {
				dbg(3, " [Message failure: %v is not a string] \n ", fld)
				return e
				// TODO: return error, or insert a log message
				// with this error (with ErrorLevel?) and
				// exit.  So someone sees it.  This varags
				// system could lead to silent failures.
			}
			goto ParamDone
		}

		// We allow odd number of arguments, so
		// can't set until after lastparam check.
		val := keyvals[i+1]

		// Special case - A nil in a "field" position
		// indicates any remaining paramters should be taken
		// as a Printf arguments (format, ...args).
		if fld == nil {
			subargs := keyvals[i+2:]
			//fmt.Printf("subargs are: %v\n", subargs)
			fmtstring, ok := val.(string)
			if !ok {
				fmt.Printf("value after a field <nil> should be a format string.")
				return e
			}
			message = fmt.Sprintf(fmtstring, subargs...)
			//fmt.Printf("result string is : %s\n", str)
			goto ParamDone
		}

		// Magic key name handling
		if fld == SeverityKey {
			sev := val
			dbg(3, "  >>>Found SEV, with value %s \n", sev)

			newLevel, err := ParseLevel(sev.(string))
			if err != nil {
				fmt.Printf("Error: Invalid severity level %s : %s\n", sev, err)
				// TODO: return error, insert a log message, or ....
			} else {
				// level to output this log message at
				e.Level = newLevel
				dbg(3, ">level: %s, set by parameter\n", e.Level)
			}
			continue
		}

		// More magic key name handling
		if fld == TagKey {
			tags := val
			if tagsSlice, ok := tags.([]string); ok {
				dbg2(" [Tag value(s) good:: %v ] \n ", tags)
				e = e.Tags(tagsSlice)
			} else {
				dbg2(" Error: [Tag value(s) bad:: %v ] \n ", tags)
				return e
				// TODO: return error, insert a log message, or ....
			}
			continue
		}

		// Handle normal key/value pairs destined for structured logging
		if strval, ok := fld.(string); ok {
			fields[strval] = val
			dbg(3, " [log key good: %v] \n ", fld)
		} else {
			dbg(3, " [log key failure: %v is not a string] \n ", fld)
			return e
			// TODO: return error, insert a log message, or ....
		}
	}
ParamDone:
	// Tack on the fields we collected earlier
	if len(fields) != 0 {
		dbg(2, "  Collected fields: %v   Level:%s\n", fields, e.Level)
		e = e.WithFields(fields)
		dbg(2, "  Fields after append: %v  Level:%s\n", e.Data, e.Level)
	}

	// What level of log statement is this?
	dbg(2, "  >>>level: %s\n", e.Level)
	switch e.Level {
	case DebugLevel:
		printFunc = e.Debug
	case InfoLevel:
		printFunc = e.Info
	case WarnLevel:
		printFunc = e.Warn
	case ErrorLevel:
		printFunc = e.Error
	case FatalLevel:
		printFunc = e.Fatal
	case PanicLevel:
		printFunc = e.Panic
	default:
		printFunc = e.Print
	}

	printFunc(message)
	return e
}

//TODO: remove these or rename them if anyone still wants a Log() call
//with types vs string

//LogOld - lint
func (e Entry) LogOld(msg string) {
	e.logIfLevel(e.Level, msg)
}

// LogOld - sugared call combining walrus Tag() and legacy logrus WithFields() to make the
// common logging case simpler. Only implementing for Logger, not Entry, so
// you can't use this to further embellish an existing Entry with more
// Tags or Fields
func (l *Logger) LogOld(level Level, tagName string, fields Fields, message string) {
	// Since Entry already has Tagging and Fields methods, we'll
	// implement Log as a short-lived entry.  The Entry pool
	// should reduce the overhead.
	entry := l.newEntry()
	defer l.releaseEntry(entry)

	if tagName != "" {
		//entry.LogTags = []string{tagName}
		entry = entry.Tag(tagName)
	}

	if fields != nil {
		entry = entry.WithFields(fields)
	}

	var printFunc func(args ...interface{})

	// What level of log statement is this?
	switch level {
	case DebugLevel:
		printFunc = entry.Debug
	case InfoLevel:
		printFunc = entry.Info
	case WarnLevel:
		printFunc = entry.Warn
	case ErrorLevel:
		printFunc = entry.Error
	case FatalLevel:
		printFunc = entry.Fatal
	case PanicLevel:
		printFunc = entry.Panic
	default:
		printFunc = entry.Print
	}

	printFunc(message)
}

// StartConfigWatcher - monitor a network connection (gRPC or other) and
// use any incoming log change messages to update this logger's
// configuration
// NOTE:  we could do this automatically with a package level init() function.
// TODO: package level vs logger level watching
//   ?? decouple
func (l *Logger) StartConfigWatcher() {

	//TODO:

	// hook directly into gRPC, or use a channel to make it more easily tested?
	// What interface could this consume that gRPC thingy would satisfy.

	// TODO: How would you pass in gRPC thingy in real world?
	// Get gRPC demo working and/or look at code to see how two programs are  set up.
	fmt.Printf("==> StartConfigWatcher not implemented yet\n")
}

// ======= Some aspects of the retrologging design may change :

// - buffered entries are not removed when output.  A second trigger
// can output previously-output entries

// - Tagged entries are buffered only if storeTaggedEntries is true.
// It may be simpler to always include them, but the whole point of
// tagged logging is to be able to have more verbose logging that can
// be selectively triggered, since the output volume of all tags may
// be high.

// EnableRetroLog -  sets up retrograde logging that occurs when a trigger
// level message is output.
// For instance, it may be beneficial to see the last 10 log messages
// of any type once a Error level message is output.
func (l *Logger) EnableRetroLog(bufferEntries int, triggerLevel Level, bufferTagged bool) error {
	if bufferEntries < 1 {
		return fmt.Errorf("%d is too small a ring buffer to do retroactive logging", bufferEntries)
	}

	l.MsgRing = ring.New(bufferEntries)
	l.ringTrigger = triggerLevel
	l.ringTagged = bufferTagged
	return nil
}

// IsRetroLogEnabled - Happy lint
func (l *Logger) IsRetroLogEnabled() bool {
	return l.MsgRing != nil
}

// IsTaggedRetroLogEnabled - Happy lint
func (l *Logger) IsTaggedRetroLogEnabled() bool {
	return (l.MsgRing != nil) && l.ringTagged
}

// DisableRetroLog - Happy lint
func (l *Logger) DisableRetroLog() {
	l.MsgRing = nil
}

// DumpRetroLog - assumes buffer size and number of entries to dump is
// the same.
func (l *Logger) DumpRetroLog() {
	if l.MsgRing == nil {
		return
	}
	backup := -l.MsgRing.Len()
	l.MsgRing.Move(backup)

	l.MsgRing.Do(func(x interface{}) {
		if x != nil {
			e := x.(Entry)
			//fmt.Printf(" retroDump:::")
			e.log()
		}

	})

}

// SetGenerateFilter does nothing for now.
func (e *Entry) SetGenerateFilter(fields ...interface{}) {
}

// SetTransportFilter does nothing for now.
func (e *Entry) SetTransportFilter(fields ...interface{}) {
}

// SetGenerateFilter does nothing for now.
func (l *Logger) SetGenerateFilter(fields ...interface{}) {
}

// SetTransportFilter does nothing for now.
func (l *Logger) SetTransportFilter(fields ...interface{}) {
}
