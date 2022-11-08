package log

import (
	"log"
	"os"
	"fmt"
	"io"
)

// Private constants
const (
	logFlagsAlways	=	log.Lmsgprefix
	defaultPermMode	=	0o644
)

// Private types
type logMsg struct {
	format string
	args []any
	fatal bool
}

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
	errEventStat statFunc
	wrnEventStat statFunc
}

//nolint:gochecknoglobals // Auxiliary variable to avoid tests termination on Fatal() function
var fatalDoExit = true

func NewLogger() *Logger {
	// By default print log messages to default logger target
	return &Logger{
		logger: log.Default(),
		closed:	true,
	}
}

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
					l.logger.Printf(msg.format, msg.args...)

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

func (l *Logger) Flags() int {
	return l.logFlags
}

func (l *Logger) SetFlags(flags int) error {
	l.setFlags(l.origPrefix, flags)
	return l.Reopen()
}

func (l *Logger) SetDebug(v bool) {
	l.debug = v
}

func (l *Logger) SetStatFuncs(sf *StatFuncs) {
	l.errEventStat = sf.Error
	l.wrnEventStat = sf.Warning
}

func (l *Logger) D(format string, v ...any) {
	if !l.debug {
		return
	}
	l.msgCh <-&logMsg{format: "<D> " + format, args: v}
}
func (l *Logger) Debug(format string, v ...any) {
	l.D(format, v...)
}

func (l *Logger) I(format string, v ...any) {
	l.msgCh <-&logMsg{format: format, args: v}
}
func (l *Logger) Info(format string, v ...any) {
	l.I(format, v...)
}

func (l *Logger) W(format string, v ...any) {
	l.msgCh <-&logMsg{format: "<WRN> " + format, args: v}

	// Call statistic function if was set
	if l.wrnEventStat != nil {
		l.wrnEventStat(format, v...)
	}
}
func (l *Logger) Warn(format string, v ...any) {
	l.W(format, v...)
}

func (l *Logger) E(format string, v ...any) {
	// If logger output is not stderr
	if l.logger.Writer() != os.Stderr {
		// Using default logger to print message to stderr
		log.Printf("<ERR> " + format, v...)
	}

	l.msgCh <-&logMsg{format: "<ERR> " + format, args: v}

	// Call statistic function if was set
	if l.errEventStat != nil {
		l.errEventStat(format, v...)
	}
}
func (l *Logger) Err(format string, v ...any) {
	l.E(format, v...)
}

func (l *Logger) F(format string, v ...any) {
	// If logger output is not stderr
	if l.logger.Writer() != os.Stderr {
		// Using default logger to print message to stderr
		log.Printf("<FATAL> " + format, v...)
	}

	l.msgCh <-&logMsg{format: "<FATAL> " + format, args: v, fatal: true}
}
func (l *Logger) Fatal(format string, v ...any) {
	l.F(format, v...)
}

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
			return fmt.Errorf("cannot close log file: %w", err)
		}
	}

	// Set closed flag
	l.closed = true

	// OK
	return nil
}

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
			return fmt.Errorf("cannot open log file: %w", err)
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
