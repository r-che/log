package log

import (
	"log"
	"os"
	"fmt"
)

var logger *log.Logger

var logName string
var logPrefix string
var logFlags int
var debug = false

type logMsg struct {
	format string
	args []any
	fatal bool
}

var msgCh chan *logMsg
var stpStrCh chan interface{}

func Open(file, prefix string, flags int) error {
	logName = file
	logPrefix = prefix
	logFlags = flags

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
					} else {
						logger.Printf(msg.format, msg.args...)
					}

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

func Debug(format string, v ...any) {
	if !debug {
		return
	}
	msgCh <-&logMsg{format: "DEBUG: " + format, args: v}
}
func D(format string, v ...any) {
	Debug(format, v...)
}

func Info(format string, v ...any) {
	msgCh <-&logMsg{format: format, args: v}
}
func I(format string, v ...any) {
	Info(format, v...)
}

func Warn(format string, v ...any) {
	msgCh <-&logMsg{format: "WARN: " + format, args: v}
}
func W(format string, v ...any) {
	Warn(format, v...)
}

func Err(format string, v ...any) {
	msgCh <-&logMsg{format: "ERROR: " + format, args: v}
}
func E(format string, v ...any) {
	Err(format, v...)
}

func Fatal(format string, v ...any) {
	msgCh <-&logMsg{format: "FATAL: " + format, args: v, fatal: true}
}
func F(format string, v ...any) {
	Fatal(format, v...)
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

	logger.SetFlags(log.Lmsgprefix | logFlags)
	logger.SetPrefix(fmt.Sprintf("%s[%d]: ", logPrefix, os.Getpid()))

	return nil
}
