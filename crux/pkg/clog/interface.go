/*
Package clog defines the Logger interface -- the primary logging interface for the crux project.
The implementation draws heavily from the logrus and go-kit packages, and from ideas from Cheney.

The scheme embodied here is quite different from the traditional System 5 syslog stuff of yore.
It differs both in WHAT to log (the content) and HOW that logging is processed.

Lets talk about the HOW first.
There is a reference model for emitting and collecting these log messages. It consists of several parts:
message construction:
	this is structured logging (from logrus), which is fielded information, including a set of labels or tags.
message generation:
	this is a filter governing whether or not an message is actually generated by a constructed message.
	this reflects concern over the cost and volume of the message stream.
message buffering:
	generated messages are fed to a sliding window in-memory buffer of messages.
message filtering:
	this is a filter governing whether or not an message leaving the message buffer is transported.
message transport:
	this consists of the encoding and transmission of messages from the emitting process to wherever
	the messages end up. the end-point might be a disk file, a daemon on the current system, or a remote server (e.g. rsyslog).

The filters are set by SetGenerate*; we are not yet sure of the syntax. The default generate filter is only generate
Error messages. The default transport is transmit everything; that is, it has a nil filter.

Message buffering is not yet done; details to come. For now, treat it as a buffer of size 1.

Transport is done as part of Logger creation. Currently, there are two choices available:
crux.GetLogger(): messages are dumped into the native syslog. note that in syslog terminology,
	the messages will have a facility=LOG_LOCAL4, and a level of either LOG_ERR, LOG_INFO or LOG_DEBUG.
crux.GetLoggerW(io.Writer): messages are written in JSON format to the Writer. as above, they will
	have a facility and level set for convenience, but it would be unwise to depend on that, especially
	if the Writer goes to a remote server.

Keep in mind that we plan a scheme to dynamically change several aspects, including the two filters. Thus, a production
system might run with the generate filter set to Error, but if it misbehaves, an operator might change that filter
to a coarse level of debugging and information messages.

The WHAT aspect is more philosophical in nature. Older schemes effectively give you the option of
almost no output (just warnings and errors), a modest amount of output (informational), or huge amounts (debug).
Clog allows you to improve upon this in two independent ways:
	detail: four levels: info, coarse, medium, fine (from less to more volume)
	tags: arbitrary string labels that let you specify areas of interest

This encourages the user to think about each message; what functional part it plays in the software,
and when we'd like to see it. For example, clustering software might naturally fall into distinct activities:
encryption, membership, leader elections, and the rest (lets call this "general"). So mark each of those with
the activity as a tag. Within an activity, think about how often you need to see that message. For example,
within the membership activity, we might set up these messages (and detail level):
	heartbeat from leader: Medium
	probe from a stranger: Coarse
	we want to join a different cluster: Info
	we reject an offer to join another cluster: Coarse
	start our own new cluster: Info

This may seem tedious to do, but when things blow up, it lets you select an area to focus on,
and then turn up or down the level of detail of activity within that focus.
*/
package clog

// shut up lint!
const (
	LogfDetail   = "LOGF_DETAIL"
	LogdInfo     = "DETAIL_INFO"
	LogdCoarse   = "DETAIL_COARSE"
	LogdMedium   = "DETAIL_MEDIUM"
	LogdFine     = "DETAIL_FINE"
	LogfTags     = "LOGF_TAGS"
	LogfPriority = "LOGF_PRIORITY"
	LogpNormal   = "PRIORITY_NORMAL"
	LogpError    = "PRIORITY_ERROR"
	LogpFatal    = "PRIORITY_FATAL"
)

// Logger is the type exported by the logging stuff.
type Logger interface {
	With(fields ...interface{}) Logger
	Log(fields ...interface{}) Logger

	SetGenerateFilter(fields ...interface{})
	SetTransportFilter(fields ...interface{})

	// various helper functions
	Logi(keyvals ...interface{})  // Log(LogfDetail, LogdInfo, keyvals)
	Logc(keyvals ...interface{})  // Log(LogfDetail, LogdCoarse, keyvals)
	Logm(keyvals ...interface{})  // Log(LogfDetail, LogdMedium, keyvals)
	Logf(keyvals ...interface{})  // Log(LogfDetail, LogdFine, keyvals)
	Error(keyvals ...interface{}) // Log(LogfPriority, LogpError, keyvals)
	Fatal(keyvals ...interface{}) // Log(LogfPriority, LogpFatal, keyvals); os.Exit(1)
	Panic(keyvals ...interface{}) // Log(LogfPriority, LogpFatal, keyvals); coredump

	SetDebug() // filter(LogfDetail, {LogdInfo,LogdCoarse,LogdMedium,LogdFine}, OR, LogfPriority, {LogpError,LogpFatal})
	SetInfo()  // filter(LogfDetail, LogdInfo, OR, LogfPriority, {LogpError,LogpFatal})
	SetWarn()  // filter(LogfPriority, {LogpError,LogpFatal})
}
