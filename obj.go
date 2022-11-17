package log

import (
	"log"
	"os"
	"fmt"
	"io"
	"errors"
)

// Private constants
const (
	logFlagsAlways	=	log.Lmsgprefix
	defaultPermMode	=	0o644
)

// ErrLogClosed returned when Close is called on a closed or never opened log-file
var ErrLogClosed	=	OpError{errors.New("log already closed/not opened yet")}

// Private types
type logMsg struct {
	format string
	args []any
	fatal bool
	done chan bool
}

// A Logger represents an active logging object that generates lines of output to file
// specified by the file parameter of the [Logger.Open] function. Each logging operation
// makes a single call to the Writer's Write method. A Logger can be used simultaneously
// from multiple goroutines; it guarantees to serialize access to the log file.
type Logger struct {
	logger		*log.Logger
	logName		string
	origPrefix	string
	logPrefix	string
	logFlags	int
	debug		bool
	closed		bool

	msgCh		chan *logMsg
	stpStrCh	chan any

	// Statistic functions
	errEventStat StatFunc
	wrnEventStat StatFunc
}

//nolint:gochecknoglobals // Auxiliary variable to avoid tests termination on Fatal() function
var fatalDoExit = true
//nolint:gochecknoglobals // Auxiliary variable to enable govet printf checking, can be true only in tests
var govetPrintfStub = false

// NewLogger creates a new Logger. By default, the logger object has no writer object and must
// be initialized using [Logger.Open] function.
func NewLogger() *Logger {
	// By default print log messages to default logger target
	return &Logger{
		logger: log.Default(),
		closed:	true,
	}
}

// Open calls [Open] on the l object.
func (l *Logger) Open(file, prefix string, flags int) error {
	l.logName = file

	l.setFlags(prefix, flags)

	if err := l.openLog(); err != nil {
		return err
	}

	// Initiate channel to write logging data from a single point
	l.msgCh = make(chan *logMsg)
	// Stop/start channel
	l.stpStrCh = make(chan interface{})
	go func() {
		for {
			select {
			// Wait for messages
			case msg := <-l.msgCh:
				if msg.fatal {
					// XXX This condition is not satisfied only in tests
					if fatalDoExit {
						l.logger.Fatalf(msg.format, msg.args...)
					}
				}

				// Write message to the log
				l.logger.Printf(msg.format, msg.args...)

				// Close the done channel in the message to notify the caller that the message is written
				close(msg.done)

			case <-l.stpStrCh:
				// Send signal that stop message was received
				l.stpStrCh <- nil

				// Wait for start message
				<-l.stpStrCh
			}
		}
	}()

	// No errors
	return nil
}

// Flags calls [Flags] on the l object.
func (l *Logger) Flags() int {
	return l.logFlags
}

// SetFlags calls [SetFlags] on the l object.
//
// NOTE: SetFlags must be called after calling l.Open, otherwise it will cause a panic.
func (l *Logger) SetFlags(flags int) error {
	l.setFlags(l.origPrefix, flags)
	return l.Reopen()
}

// SetDebug calls [SetDebug] on the l object.
func (l *Logger) SetDebug(v bool) {
	l.debug = v
}

// SetStatFuncs calls [SetStatFuncs] on the l object.
func (l *Logger) SetStatFuncs(ef, wf StatFunc) {
	l.errEventStat = ef
	l.wrnEventStat = wf
}

// D is an shortcut for Debug.
func (l *Logger) D(format string, v ...any) {
	if !l.debug {
		return
	}
	l.writeEvent(&logMsg{format: "<D> " + format, args: v})

	// XXX Enable govet printf checking
	if govetPrintfStub { _ = fmt.Sprintf(format, v...) }
}
// Debug calls [Debug] on the l object.
func (l *Logger) Debug(format string, v ...any) {
	l.D(format, v...)
}

// I is an shortcut for Info.
func (l *Logger) I(format string, v ...any) {
	l.writeEvent(&logMsg{format: format, args: v})

	// XXX Enable govet printf checking
	if govetPrintfStub { _ = fmt.Sprintf(format, v...) }
}
// Info calls [Info] on the l object.
func (l *Logger) Info(format string, v ...any) {
	l.I(format, v...)
}

// W is an shortcut for Warn.
func (l *Logger) W(format string, v ...any) {
	l.writeEvent(&logMsg{format: "<WRN> " + format, args: v})

	// Call statistic function if was set
	if l.wrnEventStat != nil {
		l.wrnEventStat(format, v...)
	}

	// XXX Enable govet printf checking
	if govetPrintfStub { _ = fmt.Sprintf(format, v...) }
}
// Warn calls [Warn] on the l object.
func (l *Logger) Warn(format string, v ...any) {
	l.W(format, v...)
}

// E is an shortcut for Err.
func (l *Logger) E(format string, v ...any) {
	// If logger output is not stderr
	if l.logger.Writer() != os.Stderr {
		// Using default logger to print message to stderr
		log.Printf("<ERR> " + format, v...)
	}

	l.writeEvent(&logMsg{format: "<ERR> " + format, args: v})

	// Call statistic function if was set
	if l.errEventStat != nil {
		l.errEventStat(format, v...)
	}

	// XXX Enable govet printf checking
	if govetPrintfStub { _ = fmt.Sprintf(format, v...) }
}
// Err calls [Err] on the l object.
func (l *Logger) Err(format string, v ...any) {
	l.E(format, v...)
}

// F is an shortcut for Fatal.
func (l *Logger) F(format string, v ...any) {
	// If logger output is not stderr
	if l.logger.Writer() != os.Stderr {
		// Using default logger to print message to stderr
		log.Printf("<FATAL> " + format, v...)
	}

	l.writeEvent(&logMsg{format: "<FATAL> " + format, args: v, fatal: true})

	// XXX Enable govet printf checking
	if govetPrintfStub { _ = fmt.Sprintf(format, v...) }
}
// Fatal calls [Fatal] on the l object.
func (l *Logger) Fatal(format string, v ...any) {
	l.F(format, v...)
}

// Close calls [Close] on the l object.
func (l *Logger) Close() error {
	// Check for log already closed
	if l.closed {
		return &ErrLogClosed
	}

	// Stop receiving messages
	l.stpStrCh<-nil
	// Wait acknowledge message from writer-goroutine
	<-l.stpStrCh

	// Check for empty name of the log file
	if l.logName == "" {
		// Standard logger was used, nothing to close
		return nil
	}

	// Close opened file
	if closer, ok := l.logger.Writer().(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return NewFileError("cannot close log file: %w", err)
		}
	}

	// Set closed flag
	l.closed = true

	// OK
	return nil
}

// Reopen calls [Reopen] on the l object.
func (l *Logger) Reopen() error {
	// Close opened log file
	if err := l.Close(); err != nil {
		return err
	}

	// Open log file again
	if err := l.openLog(); err != nil {
		return err
	}

	// Start mesages processing
	l.stpStrCh<-nil

	// Log reopened successfully
	return nil
}

func (l *Logger) openLog() error {
	if l.logName == DefaultLog {
		l.logger = log.Default()
	} else {
		logFd, err := os.OpenFile(l.logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, defaultPermMode)
		if err != nil {
			return NewFileError("cannot open log file: %w", err)
		}

		l.logger = log.New(logFd, "", log.LstdFlags)
	}

	l.logger.SetFlags(l.logFlags)
	l.logger.SetPrefix(l.logPrefix)

	// Configure default logger to print error/fatal messages to stderr
	log.SetPrefix(l.logPrefix)
	log.SetFlags(l.logFlags)

	// Reset closed flag
	l.closed = false

	return nil
}

func (l *Logger) setFlags(prefix string, flags int) {
	// Keep an original prefix value
	l.origPrefix = prefix

	if flags & NoPID == 0 {
		// Print PID in each log line
		l.logPrefix = fmt.Sprintf("%s[%d]: ", prefix, os.Getpid())
	} else
	// PID should not be printed
	if prefix != "" {
		l.logPrefix = fmt.Sprintf("%s: ", prefix)
	} // else - do not print any prefix

	// Apply mandatory flags
	l.logFlags = flags | logFlagsAlways
}

func (l *Logger) writeEvent(event *logMsg) {
	// Initiate a channel to block call until the message is written
	event.done = make(chan bool)

	// Send event to writer goroutine
	l.msgCh<-event

	// Wait for done signal
	<-event.done
}
