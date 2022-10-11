package log

import (
	"log"
	"os"
	"fmt"
)

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

type Logger struct {
	logger		*log.Logger
	logName		string
	origPrefix	string
	logPrefix	string
	logFlags	int
	debug		bool

	msgCh		chan *logMsg
	stpStrCh	chan any

	// Statistic functions
	errEventStat statFunc
	wrnEventStat statFunc
}

func NewLogger() *Logger {
	// By default print log messages to default logger target
	return &Logger{
		logger: log.Default(),
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
						l.logger.Fatalf(msg.format, msg.args...)
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

func (l *Logger) SetFlags(flags int) {
	l.setFlags(l.origPrefix, flags)
	l.Reopen()
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
	// Stop receiving messages
	l.stpStrCh<-nil
	// Wait acknowledge message from writer-goroutine
	<-l.stpStrCh

	if l.logName == "" {
		// Standard logger was used, nothing to close
		return nil
	}

	// Close opened file
	return l.logger.Writer().(*os.File).Close()
}

func (l *Logger) Reopen() error {
	// Close opened log file
	l.Close()

	// Open log file again
	if err := l.openLog(); err != nil {
		return err
	}

	// Start mesages processing
	l.stpStrCh<-nil

	// Log reopened successfuly
	return nil
}

func (l *Logger) openLog() error {
	if l.logName == DefaultLog {
		l.logger = log.Default()
	} else {
		logFd, err := os.OpenFile(l.logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		l.logger = log.New(logFd, "", log.LstdFlags)
	}

	l.logger.SetFlags(l.logFlags)
	l.logger.SetPrefix(l.logPrefix)

	// Configure default logger to print error/fatal messages to stderr
	log.SetPrefix(l.logPrefix)
	log.SetFlags(l.logFlags)

	return nil
}

func (l *Logger) setFlags(prefix string, flags int) {
	// Keep an original prefix value
	l.origPrefix = prefix

	if flags & NoPID == 0 {
		// Print PID in each log line
		l.logPrefix = fmt.Sprintf("%s[%d]: ", prefix, os.Getpid())
	} else {
		// PID should not be printed
		if prefix != "" {
			l.logPrefix = fmt.Sprintf("%s: ", prefix)
		} // else - do not print any prefix
	}

	// Apply mandatory flags
	l.logFlags = flags | logFlagsAlways
}
