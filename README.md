Logger - enhanced simple logger based on standard log
==========

[![Go Reference](https://pkg.go.dev/badge/github.com/r-che/log.svg)](https://pkg.go.dev/github.com/r-che/log)

The log package enhances the functionality of the standard [log] package.

[log]: https://pkg.go.dev/log

-------------------------
## Installation

Install the package:

```bash
go get github.com/r-che/log
```
-------------------------

## Features

  * Process identifier (PID) in log messages, optional
  * Log file reopening function to support logs rotation
  * Support for setting statistics functions
  * Support for configuration with the flags of the standard [log] package
  * Debug function to write messages to the log file only when debug mode is enabled
  * By default, timestamps are disabled, to avoid duplicating timestamps when working under the supervisor (systemd and so on)
  * Error and Fatal messages are duplicated in the stderr
  * Concurrency safe using goroutines + channels

-------------------------

## Example

The following code will print log messages to the log file /tmp/test-app.log:
```go
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
```

The log file will contain something like this:
```
2022/11/08 14:50:53 test-app[2482579]: [#1] INFO log message
2022/11/08 14:50:53 test-app[2482579]: <WRN> [#2] WARNING log message
2022/11/08 14:50:53 test-app[2482579]: <ERR> [#3] ERROR log message
2022/11/08 14:50:53 test-app[2482579]: <D> [#4] DEBUG log message after SetDebug
2022/11/08 14:50:53 test-app[2482579]: <FATAL> [#5] FATAL log message
```

You can find [more examples] in the package reference.

[more examples]: https://pkg.go.dev/github.com/r-che/log#pkg-examples

## Feedback

Feel free to open the [issue] if you have any suggestions, comments or bug reports.

[issue]: https://github.com/r-che/log/issues
