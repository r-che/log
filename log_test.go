package log

import (
	"fmt"
	"testing"
	"os"
	"path/filepath"
	"strings"
	"sort"
)

var tempDir string
const (
	stubPID	=	"19851996"
	stubApp	=	"test-log-app"
)

func TestMain(m *testing.M) {
	// Temporary directory to write test logs
	var err error
	tempDir, err = os.MkdirTemp("", `go-test-rche-log.*`)
	if err != nil {
		panic("Cannot create temporary directory for tests: " + err.Error())
	}

	// TODO Need to print messge - do not worry about <ERR> messages to STDERR

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
	testNames := make([]string, 0, len(tests))
	for testN := range tests {
		testNames = append(testNames, testN)
	}
	// Sort it
	sort.Strings(testNames)

	// Run tests
	for _, testN := range testNames {
		// Get test configuration
		test := tests[testN]

		// Create output filename
		file := filepath.Join(tempDir, fmt.Sprintf("log-test_%s.log", testN))

		// Reinit internal package's logger variable to reset any configured parameters
		logger = NewLogger()

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
			if ln + 1 > len(produced) {
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
