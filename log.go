package log

// Public constants
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

// StatFunc defines the interface for error and warning statistics functions
type StatFunc func(string, ...any)
type StatFuncs struct {
	Error	StatFunc
	Warning	StatFunc
}

//nolint:gochecknoglobals // Pointer to the default logger
var logger *Logger

func Open(file, prefix string, flags int) error {
	logger = NewLogger()
	return logger.Open(file, prefix, flags)
}

func Flags() int {
	return logger.Flags()
}

func SetFlags(flags int) error {
	return logger.SetFlags(flags)
}

func SetDebug(v bool) {
	logger.SetDebug(v)
}

func SetStatFuncs(ef, wf StatFunc) {
	logger.SetStatFuncs(ef, wf)
}

func D(format string, v ...any) {
	logger.D(format, v...)
}
func Debug(format string, v ...any) {
	logger.Debug(format, v...)
}

func I(format string, v ...any) {
	logger.I(format, v...)
}
func Info(format string, v ...any) {
	logger.Info(format, v...)
}

func W(format string, v ...any) {
	logger.W(format, v...)
}
func Warn(format string, v ...any) {
	logger.Warn(format, v...)
}

func E(format string, v ...any) {
	logger.E(format, v...)
}
func Err(format string, v ...any) {
	logger.Err(format, v...)
}

func F(format string, v ...any) {
	logger.F(format, v...)
}
func Fatal(format string, v ...any) {
	logger.Fatal(format, v...)
}

func Close() error {
	return logger.Close()
}

func Reopen() error {
	return logger.Reopen()
}
