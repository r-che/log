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
	stdLog "log"
)

const (
	stubPID	=	"19851996"
	stubApp	=	"test-log-app"
)

var tempDir string
const stubLogFormat = `Test #%d - %s log message`
const errIsOk = `(It's OK - it's just a test message)`

// Disable exiting on fatal log messages
func init() {
	fatalDoExit = false
}

func TestMain(m *testing.M) {
	// Temporary directory to write test logs
	var err error
	tempDir, err = os.MkdirTemp("", `go-test-rche-log.*`)
	if err != nil {
		panic("Cannot create temporary directory for tests: " + err.Error())
	}

	// Run tests
	ret := m.Run()

	// If all tests passed
	if ret == 0 {
		// Remove temporary data
		os.RemoveAll(tempDir)
	} else {
		// Print notification where produced data can be found
		fmt.Fprintf(os.Stderr, "Tests NOT passed, you can review produced data in: %s\n", tempDir)
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

	// Run tests
	for _, testN := range testNames {
		// Get test configuration
		test := loggingTests[testN]

		// Create output filename
		file := filepath.Join(tempDir, fmt.Sprintf("log-test_%s.log", testN))

		// Open log file
		if err := Open(file, stubApp, test.flags); err != nil {
			t.Errorf("[%s] cannot open test log file: %v", testN, err)
			t.FailNow()
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
					t.Errorf("[%s:%d] test.forEach failed: %v", testN, i, err)
					t.FailNow()
				}
			}

			// Write intput to log
			input.f(stubLogFormat, input.args...)
		}

		// Close opened file
		if err := Close(); err != nil {
			t.Errorf("[%s] cannot close test log file: %v", testN, err)
			t.FailNow()
		}

		//
		// Compare produced with expected
		//

		// Reading test output
		data, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("[%s] cannot read produced file: %v", testN, err)
			t.FailNow()
		}

		// Convert data to list of strings
		produced := strings.Split(string(data), "\n")

		// Normally, produced contains empty line at the end because the must have "\n" at the end
		if len(produced) > 0 {
			if last := produced[len(produced) - 1]; last != "" {
				t.Errorf("[%s] log file %q does not end with a new line", testN, file)
			} else {
				// Remove empty line from produced
				produced = produced[0:len(produced)-1]
			}
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

	// Counters for errors and warnings
	errN, wrnN := 0, 0
	// Errors and warnings messages that have to be produced by logging functions
	errs, wrns := []string{}, []string{}

	// Errors statistic function
	errStat := func(format string, args ...any) {
		// Increment errors counter
		errN++
		// "Print" data to error messages
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	// Warnings statistic function
	wrnStat := func(format string, args ...any) {
		// Increment errors counter
		wrnN++
		// "Print" data to error messages
		wrns = append(wrns, fmt.Sprintf(format, args...))
	}

	// Expected statistic results
	expErrs, expWrns := []string{}, []string{}
	expEN, expWN := 0, 0

	// Set statistic functions to log object
	SetStatFuncs(&StatFuncs{Error: errStat, Warning: wrnStat})

	//
	// Run tests set
	//
	for i, call := range statisticTests {
		// Make suitable arguments to call
		args := append(append([]any{}, any(i)), call.args...)

		// Call logging function
		call.f(stubLogFormat, args...)

		// Update expectations
		switch call.fType {
		case tErr:
			expEN++
			expErrs = append(expErrs, fmt.Sprintf(stubLogFormat, args...))
		case tWarn:
			expWN++
			expWrns = append(expWrns, fmt.Sprintf(stubLogFormat, args...))
		case tDebug, tInfo:
			// Do not register agruments
		default:
			panic(fmt.Sprintf("Unknown log function type: %d", call.fType))
		}

	}
	// Close log file
	if err := Close(); err != nil {
		t.Errorf("cannot close test log file %q: %v", logFile, err)
		t.FailNow()
	}

	//
	// Check statistic results
	//

	// Check errors
	for mn := 0; mn < len(expErrs); mn++ {
		// Check that expected lines can exist in produced errors
		if mn == len(errs) {
			t.Errorf("[%d] expected error string %q but no other lines in the statistic report list",
				mn, expErrs[mn])
			// Skip the rest of the test
			goto checkWarns
		}

		// Compare messages
		if expErrs[mn] != errs[mn] {
			t.Errorf("[%d] want %q, got %q", mn, expErrs[mn], errs[mn])
		}
	}
	// Check for unexpected errors produced by statistic function
	if len(errs) > len(expErrs) {
		t.Errorf("extra error messages were found in the produced report: %#v", errs[len(expErrs):])
	}

	// Check warnings
	checkWarns:
	for mn := 0; mn < len(expWrns); mn++ {
		// Check that expected lines can exist in produced warnings
		if mn == len(wrns) {
			t.Errorf("[%d] expected warnings string %q but no other lines in the statistic report list",
				mn, expWrns[mn])
			// Skip the rest of the test
			break
		}

		// Compare messages
		if expWrns[mn] != wrns[mn] {
			t.Errorf("[%d] want %q, got %q", mn, expWrns[mn], wrns[mn])
		}
	}
	// Check for unexpected warnings produced by statistic function
	if len(wrns) > len(expWrns) {
		t.Errorf("extra warnings messages were found in the produced report: %#v", wrns[len(expWrns):])
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
	defer Close()

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
		SetFlags(oldFlags|flag)

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
	defer Close()

	Fatal("Test fatal error %s", errIsOk)
}

func TestFailOpen(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", `go-test-rche-log.*`)
	if err != nil {
		t.Errorf("cannot create temporary directory: %v", err)
		t.FailNow()
	}
	defer os.RemoveAll(tmpDir)

	// Create filename includes non-existing directory
	logFile := filepath.Join(tempDir, "this-dir-does-not-exist", "fail-open.log")

	// Try to open log on this file
	if err = Open(logFile, stubApp, NoFlags); err == nil {
		// This should not happen, register abnormal behavior
		t.Errorf("anormal situation - log opened on non-existing path %q", logFile)

		// So, close log
		if err = Close(); err != nil {
			panic("Cannot close log opened on non-existing path: " + err.Error())
		}

		return
	}

	// Need to check error type
	if !errors.Is(err, fs.ErrNotExist) {
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
