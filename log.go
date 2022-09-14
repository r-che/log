package log

import (
	"log"
	"os"
	"fmt"
)

// Public constants
const (
	// Create flags constants from left part of 32-bit number
	// to avoid collision with flags from standard log package
	NoPID	= (1 << 31) >> iota
)

// Public types
type statFunc func(string, ...any)
type StatFuncs struct {
	Error	statFunc
	Warning	statFunc
}

// Private constants
const (
	logFlagsAlways = log.Lmsgprefix
)

// Private types
type logMsg struct {
	format string
	args []any
	fatal bool
}

// Private global variables
var logger *log.Logger
var logName string
var logPrefix string
var logFlags int
var debug = false
// Statistic functions
var errEventStat statFunc
var wrnEventStat statFunc

var msgCh chan *logMsg
var stpStrCh chan interface{}

func Open(file, prefix string, flags int) error {
	logName = file
	if flags & NoPID == 0 {
		// Print PID in each log line
		logPrefix = fmt.Sprintf("%s[%d]: ", prefix, os.Getpid())
	} else {
		// PID should not be printed
		if prefix != "" {
			logPrefix = fmt.Sprintf("%s: ", prefix)
		} // else - do not print any prefix
	}

	// Apply mandatory flags
	logFlags = flags | logFlagsAlways

	if err := openLog(); err != nil {
		return err
	}

	// Initiate channel to write logging data from a single point
	msgCh = make(chan *logMsg)
	// Stop/start channel
	stpStrCh = make(chan interface{})
	go func() {
		for {
			select {
				case msg := <-msgCh:
					if msg.fatal {
						logger.Fatalf(msg.format, msg.args...)
					}
					logger.Printf(msg.format, msg.args...)

				case <-stpStrCh:
					// Send signal that stop message was received
					stpStrCh <- nil

					// Wait for start message
					<-stpStrCh
			}
		}
	}()

	// No errors
	return nil
}

func SetDebug(v bool) {
	debug = v
}

func SetStatFuncs(sf *StatFuncs) {
	errEventStat = sf.Error
	wrnEventStat = sf.Warning
}

func D(format string, v ...any) {
	if !debug {
		return
	}
	msgCh <-&logMsg{format: "<D> " + format, args: v}
}
func Debug(format string, v ...any) {
	D(format, v...)
}

func I(format string, v ...any) {
	msgCh <-&logMsg{format: format, args: v}
}
func Info(format string, v ...any) {
	I(format, v...)
}

func W(format string, v ...any) {
	msgCh <-&logMsg{format: "<WRN> " + format, args: v}

	// Call statistic function if was set
	if wrnEventStat != nil {
		wrnEventStat(format, v...)
	}
}
func Warn(format string, v ...any) {
	W(format, v...)
}

func E(format string, v ...any) {
	// If logger output is not stderr
	if logger.Writer() != os.Stderr {
		// Using default logger to print message to stderr
		log.Printf("<ERR> " + format, v...)
	}

	msgCh <-&logMsg{format: "<ERR> " + format, args: v}

	// Call statistic function if was set
	if errEventStat != nil {
		errEventStat(format, v...)
	}
}
func Err(format string, v ...any) {
	E(format, v...)
}

func F(format string, v ...any) {
	// If logger output is not stderr
	if logger.Writer() != os.Stderr {
		// Using default logger to print message to stderr
		log.Printf("<FATAL> " + format, v...)
	}

	msgCh <-&logMsg{format: "<FATAL> " + format, args: v, fatal: true}
}
func Fatal(format string, v ...any) {
	F(format, v...)
}

func Close() error {
	// Stop receiving messages
	stpStrCh<-nil
	// Wait acknowledge message from writer-goroutine
	<-stpStrCh

	if logName == "" {
		// Standard logger was used, nothing to close
		return nil
	}

	// Close opened file
	return logger.Writer().(*os.File).Close()
}

func Reopen() error {
	// Close opened log file
	Close()

	// Open log file again
	if err := openLog(); err != nil {
		return err
	}

	// Start mesages processing
	stpStrCh<-nil

	// Log reopened successfuly
	return nil
}

func openLog() error {
	if logName == "" {
		logger = log.Default()
	} else {
		logFd, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		logger = log.New(logFd, "", log.LstdFlags)
	}

	logger.SetFlags(logFlags)
	logger.SetPrefix(logPrefix)

	// Configure default logger to print error/fatal messages to stderr
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	return nil
}
