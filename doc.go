/*
Package log is an enhanced simple logger based on the standard package [log].

Package key features are:

 * Process identifier (PID) in log messages, optional
 * Log file reopening function to support logs rotation
 * Support for setting statistics functions
 * Support for configuration with the flags of the standard [log] package
 * Debug function to write messages to the log file only when debug mode is enabled
 * By default, timestamps are disabled, to avoid duplicating timestamps when working under the supervisor (systemd and so on)
 * Error and Fatal messages are duplicated in the stderr
 * Concurrency safe using goroutines + channels

# Basic usage

 import (
     "github.com/r-che/log"
     stdLog "log"
 )

 func main() {
     log.Open(
         "/tmp/test-app.log",       // log file name
         "test-app",                // application name, used as messages prefix
         stdLog.Ldate|stdLog.Ltime) // use flags of standard log to enable timestamps

     // log.Close MUST be called to avoid loss of messages on exit
     defer log.Close()

     log.D("[#%d] DEBUG log message BEFORE SetDebug", 0) // not logged, because debug is not enabled
     log.I("[#%d] INFO log message", 1)
     log.W("[#%d] WARNING log message", 2)
     log.E("[#%d] ERROR log message", 3)

     log.SetDebug(true)  // enable debug
     log.D("[#%d] DEBUG log message after SetDebug", 4)  // the debug message is now logged

     log.F("[#%d] FATAL log message", 5)
 }

# Important notes

 * Writing messages into the log before calling [Open] will cause a panic
 * [SetFlags] must be called after calling [Open], otherwise it will cause a panic
 * [Close] must be called before exiting the progam to avoid loss of the last log messages.

[log]: https://pkg.go.dev/log

# Feedback

Feel free to open the [issue] if you have any suggestions, comments or bug reports.
*/
package log
