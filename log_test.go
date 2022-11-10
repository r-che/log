package log

import (
	"fmt"
	"testing"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sort"
	"io"
	stdLog "log"
)

const (
	stubPID	=	"19851996"
	stubApp	=	"test-log-app"
)

const (
	stubLogFormat	=	`Test #%d - %s log message`
	errIsOk			=	`(It's OK - it's just a test message)`
	tempLogPrefix	=	`go-test-rche-log.*`
)

// Disable exiting on fatal log messages for testing purposes
func init() {	//nolint:gochecknoinits
	fatalDoExit = false
}

// tempDir creates temporary directory inside of the temporary root directory configured by TestMain
var tempDir func() string //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	// Create root temporary directory
	var err error
	tempRoot, err := os.MkdirTemp("", tempLogPrefix)
	if err != nil {
		panic("Cannot create root temporary directory for tests: " + err.Error())
	}

	// Initiate tempDir function to create temporary directories inside tempRoot
	tempDir = func() string {
		dir, err := os.MkdirTemp(tempRoot, "subtest.*")
		if err != nil {
			panic("Cannot create temporary directory for tests: " + err.Error())
		}

		return dir
	}

	//
	// Run tests
	//
	ret := m.Run()

	// If all tests passed
	if ret == 0 {
		// Remove temporary data
		os.RemoveAll(tempRoot)
	} else {
		// Keep produced data to investigation, print notification where it can be found
		fmt.Fprintf(os.Stderr, "\nTests NOT passed," +
			" you can review produced data in the directory: %s\n\n", tempRoot)
	}

	os.Exit(ret)
}

func TestLogging(t *testing.T) {
	// Make sorted list of test names
	testNames := make([]string, 0, len(loggingTests))
	for testN := range loggingTests {
		testNames = append(testNames, testN)
	}
	// Sort it
	sort.Strings(testNames)

	// Create temporary directory to write test logs
	logDir := tempDir()

	// Run tests
	for _, testN := range testNames {
		// Get test configuration
		test := loggingTests[testN]

		// Create output filename
		file := filepath.Join(logDir, fmt.Sprintf("log-test_%s.log", testN))

		// Write test log data to file to check with expected
		if err := writeLogSample(testN, file); err != nil {
			t.Errorf("%v", err)
			t.FailNow()
		}

		//
		// Compare produced log content with expected
		//

		// Reading test output
		data, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("[%s] cannot read produced file: %v", testN, err)
			t.FailNow()
		}

		// Convert data to list of strings
		produced, err := removeNewLine(strings.Split(string(data), "\n"))
		if err != nil {
			t.Errorf("[%s] %v - %s", testN, err, file)
		}

		// Compare resulting lines with expected
		for ln := 0; ln < len(test.expected); ln++ {
			// Check that produced[ln] exists
			if ln == len(produced) {
				t.Errorf("[%s:%d] expected string %q but no other lines in the produced file %q",
					testN, ln, test.expected[ln], file)
				// Skip the rest of the test
				goto nextTest
			}

			// Compare produced and expected
			if produced[ln] != test.expected[ln] {
				t.Errorf("[%s:%d] want %q, got %q", testN, ln, test.expected[ln], produced[ln])
			}
		}

		// Check for unexpected lines
		if len(produced) > len(test.expected) {
			t.Errorf("[%s] extra lines were found in the produced file: %#v", testN, produced[len(test.expected):])
		}

		nextTest:
	}
}

func removeNewLine(produced []string) ([]string, error) {
	if len(produced) == 0 {
		// Nothing to handle
		return produced, nil
	}

	// Normally, produced contains empty line at the end because the must have "\n" at the end
	if last := produced[len(produced) - 1]; last != "" {
		// The list line is not empty - that means that the log file was not ended by "\n"
		return nil, fmt.Errorf(`log file was not ended by "\n"`)
	}

	// Remove empty line from produced and return
	return produced[0:len(produced)-1], nil
}

func writeLogSample(name, file string) error {
	// Get test configuration
	test := loggingTests[name]

	// Open log file
	if err := Open(file, stubApp, test.flags); err != nil {
		return fmt.Errorf("[%s] cannot open test log file: %w", name, err)
	}

	// Set predefined PID
	SetPID(stubPID)

	// Call pre() if exists
	if test.pre != nil {
		test.pre()
	}

	// Run log functions from inputs
	for i, input := range test.inputs {
		// Call forEach() if exists
		if test.forEach != nil {
			if err := test.forEach(i); err != nil {
				return fmt.Errorf("[%s:%d] test.forEach failed: %w", name, i, err)
			}
		}

		// Write intput to log
		input.f(stubLogFormat, input.args...)
	}

	// Close opened file
	if err := Close(); err != nil {
		return fmt.Errorf("[%s] cannot close test log file: %w", name, err)
	}

	// OK
	return nil
}

func TestStatFunctions(t *testing.T) {
	// Dummy output file
	logFile := os.DevNull

	// Open log file
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open output file %q: %v", logFile, err)
		t.FailNow()
	}

	// Enable debug output to produce more messages
	SetDebug(true)

	//
	// Create statistic functions
	//

	// Errors and warnings messages that have to be produced by logging functions
	errs, wrns := []string{}, []string{}

	// Errors statistic function
	errStat := func(format string, args ...any) {
		// "Print" data to error messages
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	// Warnings statistic function
	wrnStat := func(format string, args ...any) {
		// "Print" data to error messages
		wrns = append(wrns, fmt.Sprintf(format, args...))
	}

	// Set statistic functions to log object
	SetStatFuncs(errStat, wrnStat)

	//
	// Run tests, get expected statistic results
	//
	expErrs, expWrns := runStatsTests()

	//
	// Close log file
	//
	if err := Close(); err != nil {
		t.Errorf("cannot close test log file %q: %v", logFile, err)
		t.FailNow()
	}

	//
	// Check results
	//

	// Check errors
	checkStatTestResults(t, errs, expErrs)

	// Check warnings
	checkStatTestResults(t, wrns, expWrns)
}

func runStatsTests() ([]string, []string) {
	// Expected statistic results
	expErrs, expWrns := []string{}, []string{}

	for i, call := range statisticTests {
		// Make suitable arguments to call
		args := append(append([]any{}, any(i)), call.args...)

		// Call logging function
		call.f(stubLogFormat, args...)

		// Update expectations
		switch call.fType {
		case tErr:
			expErrs = append(expErrs, fmt.Sprintf(stubLogFormat, args...))
		case tWarn:
			expWrns = append(expWrns, fmt.Sprintf(stubLogFormat, args...))
		case tDebug, tInfo: // do not register arguments
		case tFatal:
			panic("Fatal cannot be handled")
		default:
			panic(fmt.Sprintf("Unknown log function type: %d", call.fType))
		}
	}

	return expErrs, expWrns
}

//nolint:thelper
func checkStatTestResults(t *testing.T, gotData, expData []string) {
	// Check expected data
	for mn := 0; mn < len(expData); mn++ {
		// Check that expected lines can exist in produced log messages
		if mn == len(gotData) {
			t.Errorf("expected string %q but no other lines in the statistic report list",
				expData[mn])
			// Return now - no point to run other checks
			return
		}

		// Compare messages
		if expData[mn] != gotData[mn] {
			t.Errorf("want %q, got %q", expData[mn], gotData[mn])
		}
	}

	// Check for unexpected data produced by statistic function
	if len(gotData) > len(expData) {
		t.Errorf("extra messages were found in the produced report: %#v", gotData[len(expData):])
	}
}

func TestFlags(t *testing.T) {
	// Dummy output file
	logFile := os.DevNull

	// Open dummy log
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open output file %q: %v", logFile, err)
		t.FailNow()
	}
	defer func() {
		if err := Close(); err != nil {
			stdLog.Fatalf("Cannot close %v file: %v", os.DevNull, err)
		}
	}()

	flags := []int{
		// Flags owned by the package
		NoPID,

		// Standard log package's flags https://pkg.go.dev/log#pkg-constants
		stdLog.Ldate,
		stdLog.Ltime,
		stdLog.Lmicroseconds,
		stdLog.Llongfile,
		stdLog.Lshortfile,
		stdLog.LUTC,
		stdLog.Lmsgprefix,
		// stdLog.LstdFlags - this is flag defined as Ldate | Ltime, so, it should not be tested separately
	}

	for _, flag := range flags {
		// If this flag from always-set?
		if flag & logFlagsAlways != 0 {
			// Skip this flag
			continue
		}

		// Get the current flags value
		oldFlags := Flags()

		// Set new flag
		if err := SetFlags(oldFlags|flag); err != nil {
			t.Errorf("cannot set flags for log: %v", err)
			t.FailNow()
		}

		// Get new flags set
		newFlags := Flags()

		// Check correctness of old flags after set
		if r := oldFlags ^ newFlags; r != flag {
			t.Errorf("incorrect flags after set: oldSet(%#064b) ^ newSet(%#064b) = %#064b - want: %#064b",
				oldFlags, newFlags, r, flag)
		}
	}
}

func TestFatal(t *testing.T) {
	// Dummy output file
	logFile := os.DevNull

	// Open dummy log
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open output file %q: %v", logFile, err)
		t.FailNow()
	}
	defer func() {
		if err := Close(); err != nil {
			stdLog.Fatalf("Cannot close %v file: %v", os.DevNull, err)
		}
	}()

	Fatal("Test fatal error %s", errIsOk)
}

func TestFailOpen(t *testing.T) {
	// Create temporary directory to write test logs
	logDir := tempDir()

	// Create filename includes non-existing directory
	logFile := filepath.Join(logDir, "this-dir-does-not-exist", "fail-open.log")

	//nolint:errorlint // Try to open log on this file
	switch err := Open(logFile, stubApp, NoFlags); err.(type) {
	// No errors when error is expected
	case nil:
		// This should not happen, register abnormal behavior
		t.Errorf("anormal situation - log opened on non-existing path %q", logFile)

		// So, close log
		if err = Close(); err != nil {
			panic("Cannot close log opened on non-existing path: " + err.Error())
		}

	// Expected error
	case *FileError:
		// Additional error kind check
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("failed Open() error is %v, want - %v", err, fs.ErrNotExist)
		}

	// Unexpected error
	default:
		t.Errorf("unexpected error returned opening log on non-existing path %q: %v", logFile, err)
	}

	// Ok, test passed
}

func TestDefaultLog(t *testing.T) {
	// Open default log
	if err := Open(DefaultLog, stubApp, NoFlags); err != nil {
		t.Errorf("cannot log on default logger: %v", err)
		t.FailNow()
	}

	// Print debug message, no output will be produced because debug is not enabled
	Debug("Invisible message")

	// Test closing
	if err := Close(); err != nil {
		t.Errorf("cannot close log on default logger: %v", err)
	}

	// Ok, tests passed
}

func TestFailReopenNxFile(t *testing.T) {
	// Create temporary directory to write test logs
	logDir := tempDir()

	// Create output filename
	logFile := filepath.Join(logDir, "fail-reopen.log")

	// Open log file
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open test log file %q: %v", logFile, err)
		t.FailNow()
	}
	// Set predefined PID
	SetPID(stubPID)

	// Print debug message, no output will be produced because debug is not enabled
	Debug("Invisible message")

	// Replace normal filename by filename includes non-existing directory
	logger.logName = filepath.Join(logDir, "this-dir-does-not-exist", "fail-reopen.log")

	//nolint:errorlint // Try to reopen on changed location
	switch err := Reopen(); err.(type) {
	// No errors when error is expected
	case nil:
		// This should not happen, register abnormal behavior
		t.Errorf("anormal situation - log reopened on non-existing path %q", logger.logName)

		// So, close log
		if err = Close(); err != nil {
			panic("Cannot close log reopened on non-existing path: " + err.Error())
		}

	// Expected error
	case *FileError:
		// Additional error kind check
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("failed Reopen() error is %v, want - %v", err, fs.ErrNotExist)
		}

	// Unexpected error
	default:
		t.Errorf("unexpected error returned by reopening on non-existing path %q: %v", logFile, err)
	}

	// Ok, test passed
}

func TestFailReopenCloseErr(t *testing.T) {
	// Dummy output file
	logFile := os.DevNull

	// Open log file
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open test log file %q: %v", logFile, err)
		t.FailNow()
	}

	// Close log bypassing Close function
	if closer, ok := logger.logger.Writer().(io.Closer); ok {
		if err := closer.Close(); err != nil {
			t.Errorf("cannot close log file %q: %v", logFile, err)
			t.FailNow()
		}
	} else {
		panic(fmt.Sprintf("logger object contains invalid writter that cannot be closed," +
			" type: %T", logger.logger.Writer()))
	}

	//nolint:errorlint // Try to reopen closed file
	switch err := Reopen(); err.(type) {
	// No errors when error is expected
	case nil:
		// This should not happen, register abnormal behavior
		t.Errorf("anormal situation - log reopened on non-existing path %q", logger.logName)

		// So, close log
		if err = Close(); err != nil {
			panic("Cannot close log reopened on non-existing path: " + err.Error())
		}

	// Expected error
	case *FileError:
		// Additional error kind check
		if !errors.Is(err, fs.ErrClosed) {
			t.Errorf("failed Reopen() error is %v, want - %v", err, fs.ErrClosed)
		}

	// Unexpected error
	default:
		t.Errorf("unexpected error returned by reopening closed file %q: %v", logFile, err)
	}

	// Ok, test passed
}

func TestFailDoubleClose(t *testing.T) {
	// Create temporary directory to write test logs
	logDir := tempDir()

	// Create output filename
	logFile := filepath.Join(logDir, "fail-double-close.log")

	// Open log file
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open test log file %q: %v", logFile, err)
		t.FailNow()
	}

	// Normally close log file
	if err := Close(); err != nil {
		t.Errorf("cannot close log file %q: %v", logFile, err)
		t.FailNow()
	}

	//nolint:errorlint // Double close - expected error
	switch err := Close(); err {
	// No errors but expected
	case nil:
		t.Errorf("double Close() return no error but must")
		t.FailNow()

	// Expected error
	case &ErrLogClosed:
		// Nothing to do

	// Some unexpected error
	default:
		t.Errorf("double Close() returned unexpected error: %v", err)
		t.FailNow()
	}
}

func TestFailClose(t *testing.T) {
	// Create temporary directory to write test logs
	logDir := tempDir()

	// Create output filename
	logFile := filepath.Join(logDir, "fail-close.log")

	// Open log file
	if err := Open(logFile, stubApp, NoFlags); err != nil {
		t.Errorf("cannot open test log file %q: %v", logFile, err)
		t.FailNow()
	}

	// Close log bypassing Close function
	if closer, ok := logger.logger.Writer().(io.Closer); ok {
		if err := closer.Close(); err != nil {
			t.Errorf("cannot close log file %q: %v", logFile, err)
			t.FailNow()
		}
	} else {
		panic(fmt.Sprintf("logger object contains invalid writter that cannot be closed," +
			" type: %T", logger.logger.Writer()))
	}

	//nolint:errorlint // Try to call Close() which believes that
	// the log file is not closed, and should get the a close error
	switch err := Close(); err.(type) {
	// No errors but expected
	case nil:
		t.Errorf("failure of the Close() expected, but it did not fail")
		t.FailNow()

	// Expected error
	case *FileError:
		// Additional error kind check
		if !errors.Is(err, fs.ErrClosed) {
			t.Errorf("failed Close() error is %v, want - %v", err, fs.ErrClosed)
		}
		// OK

	// Some unexpected error
	default:
		t.Errorf("Close() returned unexpected error: %v (%T) %#v", err, err, err)
		t.FailNow()
	}
}

func TestError(t *testing.T) {
	const testErr = "test OpError"

	err := OpError{errors.New(testErr)}
	if errStr := err.Error(); errStr != testErr {
		t.Errorf("got error %q, want - %q", errStr, testErr)
	}
}

//
// log methods required only for testing
//
func (l *Logger) SetPID(pidStr string) {
	// Do nothing if PID should not be shown
	if l.logFlags & NoPID != 0 {
		return
	}

	// Replace prefix by predefined value
	l.logger.SetPrefix(fmt.Sprintf("%s[%s]: ", l.origPrefix, pidStr))
}

func SetPID(pidStr string) {
	logger.SetPID(pidStr)
}
