package log

import (
	"fmt"
	"os"
)

// Example_logFileTest will print log messages to the /tmp/test-app.log file,
// then it causes program exit with exit code 1
func Example_logFileTest() {
	Open("/tmp/test-app.log", "test-app", NoPID)
	defer Close()

	D("[#%d] DEBUG log message BEFORE SetDebug", 0)	// not logged, because debug is not enabled
	I("[#%d] INFO log message", 1)
	W("[#%d] WARNING log message", 2)
	E("[#%d] ERROR log message", 3)

	SetDebug(true)	// enable debug
	D("[#%d] DEBUG log message after SetDebug", 4)	// the debug message is now logged

	F("[#%d] FATAL log message", 5)	// this line causes program exit with exit code 1

	// The log file will contain the following messages:
	//  test-app: [#1] INFO log message
	//  test-app: <WRN> [#2] WARNING log message
	//  test-app: <ERR> [#3] ERROR log message
	//  test-app: <D> [#4] DEBUG log message after SetDebug
	//  test-app: <FATAL> [#5] FATAL log message
}

func Example_reopenLog() {
	Open("/tmp/test-app.log", "test-app", NoPID)
	defer Close()

	I("Information log message before deletion of the log file")

	os.Remove("/tmp/test-app.log") // remove log file on the fly

	I("Information message to the deleted log file") // this message will be lost

	// Reopen log file
	if err := Reopen(); err != nil {
		panic(err)
	}

	I("Log file succesfully reopened") // reopened file will start with this message

	// If you run tail -f /tmp/test-app.log, you should get:
	//
	//  test-app: Information log message before deletion of the log file
	//  test-app: Information message to the deleted log file
}

func Example_setStatFuncs() {
	Open(os.DevNull, "test-app", NoPID)
	defer Close()

	errs, wrns := []string{}, []string{}
	SetStatFuncs(
		func(format string, args ...any) { // define errors statistic function
			errs = append(errs, fmt.Sprintf(format, args...))
		},
		func(format string, args ...any) { // define warnings statistic function
			wrns = append(wrns, fmt.Sprintf(format, args...))
		},
	)

	for i := 0; i < 3; i++ {
		Err("This is a test error #%d", i)
		Warn("This is a test warning #%d", i)
	}

	fmt.Println("Collected statistics:")
	fmt.Printf("Errors: %#v\n", errs)
	fmt.Printf("Warnings: %#v\n", wrns)
	// Output:
	// Collected statistics:
	// Errors: []string{"This is a test error #0", "This is a test error #1", "This is a test error #2"}
	// Warnings: []string{"This is a test warning #0", "This is a test warning #1", "This is a test warning #2"}
}
