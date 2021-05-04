// Package log provides loglevel aware output methods.
//
// Loglevel are predefined as Ltrace, Ldebug, Linfo, Lwarn, Lerror, and Lfatal.
// The default loglevel is Linfo.
// To create a logger named mylogger under the default stream:
//     var lg = log.DefaultStream.NewLogger("mylogger", log.Linfo)
// A stream is a handle that controls the output destination and output format.
// The default output destination is stdout.
//   "stdout" : output to OS.Stdout.
// Other supported output destination:
//   "file:filepath/filename" : output to file in append mode.
// More output destinations to be added, see RegOutputterFactory() and
// fileFactory for an example.
//
// Loggers can be created under a named stream if more output destinations are
// needed.
// In "main" package:
//     mystream := log.NewStream("mystream")
// In subpackages:
//     var lg = log.GetStream("mystream").NewLogger("mylogger", log.Lwarn)
//
// It is the main package's responsibility to set or change the global output
// destination.
// To change the output destination to file:
//     mystream.SetOutput("file:filepath/filename")
//
// More than one loggers can be created on demand in one package, usually along
// with the package level logger itself, i.e. each "worker" can have its own
// logger instance, thus its loglevel can be changed separately.
// It is highly recommended that dynamic loggers are named in the URL like form
// "package/module/xxxx", eg.
// You have a package level logger lgsrv for awesomesrvice package:
//     lgsrv := log.DefaultStream.NewLogger("awesomesrvice", log.Lwarn)
// Now inside awesomesrvice, you want each worker has its own logger instance so
// that later loglevel can be changed respectively for each worker:
//     lgworker := log.DefaultStream.NewLogger("awesomesrvice/worker123", log.Lwarn)
// When you want to set loglevel to one of these loggers to debug level:
//     log.DefaultStream.SetLoglevel("awesomesrvice/worker123", log.Ldebug)
// To set all awesomesrvice loggers to debug level:
//     log.DefaultStream.SetLoglevel("awesomesrvice*", log.Ldebug)
// Dynamically created logger should be closed if not use any more.
//     lgworker.Close()
//
// When a new logger instance is getting created, it will use the loglevel that
// has been set the lastime calling stream.SetLoglevel() if the namePattern
// matches, or use the default loglevel if not.
//
// Timestamp is in format "[2006/01/02 15:04:05.000000]" by default, set it to
// "" will disable timestamp to be printed. SetTimeFormat can change the
// timestamp format, refer to standard time package for the possible formats.
//     mystream := log.NewStream("mystream")
//     mystream.SetTimeFormat(time.RFC3339Nano)
//
// Some APIs support wildcard matching to match logger name, and support multiple patterns
// separated by comma.
// The pattern uses simple * as the wildcard:
// eg.
//     "*" matches all
//     "*bar*" matches bar, foobar, or foobarabc
//     "foo*abc*" matches foobarabc, foobarabc123, or fooabc
//
// The methods are concurrency safe under linux.
package log
