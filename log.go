package log

// Exported constants:
const (
	// Default log target - empty line means that the default
	// logger from the standart log package will be used
	DefaultLog	=	""

	//
	// Supported flags
	//
	NoFlags = 0

	// Create flags constants from left part of 32-bit number
	// to avoid collision with flags from standard log package
	// XXX Do not forget to update TestFlags function after adding or removing flags
	NoPID	= (1 << 31) >> iota	//nolint:gomnd // described above
)

//
// Public types
//

// StatFunc defines the interface for error and warning statistics functions.
// The statistics function gets the same arguments, the logging function to
// which it is associated with.
type StatFunc func(format string, args ...any)

//
// Default logger object
//

//nolint:gochecknoglobals // Pointer to the default logger
var logger *Logger

// Open opens the log file to write messages with the application prefix.
// If DefaultLog (empty string) is used as the file, the output is written
// to the standard log module's Writer (usual - stderr). The value of the flags field
// can be a bit combination of NoFlags, NoPID and flags of standard log package.
//
// NOTE: writing messages into the log before calling Open will cause a panic.
func Open(file, prefix string, flags int) error {
	logger = NewLogger()
	return logger.Open(file, prefix, flags)
}

// Flags returns the set of flags
func Flags() int {
	return logger.Flags()
}

// SetFlags sets a new set of flags.
//
// NOTE: SetFlags must be called after calling Open, otherwise it will cause a panic.
func SetFlags(flags int) error {
	return logger.SetFlags(flags)
}

// SetDebug enables or disables debug mode. If debug mode is disabled (v == false),
// the debug message functions (D and Debug) do not write data to the log.
func SetDebug(v bool) {
	logger.SetDebug(v)
}

// SetStatFuncs sets the ef (for errors) and ew (for warnings) message statistics handlers.
// See [StatFunc] and the SetStatFuncs example for details.
func SetStatFuncs(ef, wf StatFunc) {
	logger.SetStatFuncs(ef, wf)
}

// D is an shortcut for Debug.
func D(format string, v ...any) {
	logger.D(format, v...)
}
// Debug writes a debug message to the log prefixed with <D>,
// but only if debug mode is enabled (see [SetDebug]).
func Debug(format string, v ...any) {
	logger.Debug(format, v...)
}

// I is an shortcut for Info.
func I(format string, v ...any) {
	logger.I(format, v...)
}
// Info writes an information message to the log. The message level prefix is not used.
func Info(format string, v ...any) {
	logger.Info(format, v...)
}

// W is an shortcut for Warn.
func W(format string, v ...any) {
	logger.W(format, v...)
}
// Warn writes a warning message prefixed with <WRN> to the log.
// It also calls the warning statistics handler, if previously set with the [SetStatFuncs] function.
func Warn(format string, v ...any) {
	logger.Warn(format, v...)
}

// E is an shortcut for Err.
func E(format string, v ...any) {
	logger.E(format, v...)
}
// Err writes a warning message prefixed with <ERR> to the log. The same message is duplicated to stderr.
// It also calls the error statistics handler, if previously set with the [SetStatFuncs] function.
func Err(format string, v ...any) {
	logger.Err(format, v...)
}

// F is an shortcut for Fatal.
func F(format string, v ...any) {
	logger.F(format, v...)
}
// Fatal writes a fatal message prefixed with <FATAL> to the log. The same message is duplicated to stderr.
// Then it causes program termination with the standard function log.Fatalf().
func Fatal(format string, v ...any) {
	logger.Fatal(format, v...)
}

// Close closes the log file. Attempts to write to the log file after closing it will cause the goroutine
// to block, which can lead to a panic when all the goroutines in the program are blocked.
//
// NOTE: [Close] must be called before exiting the progam to avoid loss of the last log messages.
func Close() error {
	return logger.Close()
}

func Reopen() error {
	return logger.Reopen()
}
