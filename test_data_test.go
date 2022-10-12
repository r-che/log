package log

import "fmt"

type logCall struct {
	f		logFunction
	args	[]any
	fType	logFuncType
}
type logFunction func(string, ...any)
type logFuncType uint8
const (
	tDebug	=	logFuncType(iota)
	tInfo
	tWarn
	tErr
	tFatal
)

const stubLogFormat = `Test #%d - %s log message`
const errIsOk = `(It's OK - this is testing error messages)`

var loggingTests = map[string]struct {
	pre			func()
	forEach		func(int) error
	flags		int
	inputs		[]logCall
	expected	[]string
}{
	`00-all-types`: {
		// All types of messages
		pre:	func() {
			SetDebug(true)
		},
		flags:	NoFlags,
		inputs:	[]logCall {
			logCall{f: Debug, args: []any{0, `DEBUG`} },
			logCall{f: Info, args: []any{1, `INFO`} },
			logCall{f: Warn, args: []any{2, `WARNING`} },
			logCall{f: Err, args: []any{3, `ERROR ` + errIsOk} },
		},
		expected: []string {
			stubApp + `[` + stubPID + `]: <D> Test #0 - DEBUG log message`,
			stubApp + `[` + stubPID + `]: Test #1 - INFO log message`,
			stubApp + `[` + stubPID + `]: <WRN> Test #2 - WARNING log message`,
			stubApp + `[` + stubPID + `]: <ERR> Test #3 - ERROR ` + errIsOk + ` log message`,
		},
	},
	`01-all-types-short`: {
		// All types of messages with short function names
		pre:	func() {
			SetDebug(true)
		},
		flags:	NoFlags,
		inputs:	[]logCall {
			logCall{f: D, args: []any{0, `DEBUG`} },
			logCall{f: I, args: []any{1, `INFO`} },
			logCall{f: W, args: []any{2, `WARNING`} },
			logCall{f: E, args: []any{3, `ERROR ` + errIsOk} },
		},
		expected: []string {
			stubApp + `[` + stubPID + `]: <D> Test #0 - DEBUG log message`,
			stubApp + `[` + stubPID + `]: Test #1 - INFO log message`,
			stubApp + `[` + stubPID + `]: <WRN> Test #2 - WARNING log message`,
			stubApp + `[` + stubPID + `]: <ERR> Test #3 - ERROR ` + errIsOk + ` log message`,
		},
	},
	`02-without-debug`: {
		// All types of messages except debug
		flags:	NoFlags,
		inputs:	[]logCall {
			logCall{f: Debug, args: []any{0, `DEBUG`} },
			logCall{f: Info, args: []any{1, `INFO`} },
			logCall{f: Warn, args: []any{2, `WARNING`} },
			logCall{f: Err, args: []any{3, `ERROR ` + errIsOk} },
		},
		expected: []string {
			stubApp + `[` + stubPID + `]: Test #1 - INFO log message`,
			stubApp + `[` + stubPID + `]: <WRN> Test #2 - WARNING log message`,
			stubApp + `[` + stubPID + `]: <ERR> Test #3 - ERROR ` + errIsOk + ` log message`,
		},
	},
	`03-with-NoPID`: {
		// All types of messages without PID
		pre:	func() {
			SetDebug(true)
		},
		flags:	NoPID,
		inputs:	[]logCall {
			logCall{f: Debug, args: []any{0, `DEBUG`} },
			logCall{f: Info, args: []any{1, `INFO`} },
			logCall{f: Warn, args: []any{2, `WARNING`} },
			logCall{f: Err, args: []any{3, `ERROR ` + errIsOk} },
		},
		expected: []string {
			stubApp + `: <D> Test #0 - DEBUG log message`,
			stubApp + `: Test #1 - INFO log message`,
			stubApp + `: <WRN> Test #2 - WARNING log message`,
			stubApp + `: <ERR> Test #3 - ERROR ` + errIsOk + ` log message`,
		},
	},
	`04-with-reopen`: {
		// All types of messages except with reopening before each message except first
		pre:	func() {
			SetDebug(true)
		},
		forEach:	func(inputN int) error {
			if inputN == 0 {
				// Skip
				return nil
			}

			Warn(`Test Reopen() #%d`, inputN)

			if err := Reopen(); err != nil {
				return fmt.Errorf(`Reopen() failed: %v`, err)
			}

			// Set fake PID again
			SetPID(stubPID)

			return nil
		},
		flags:	NoFlags,
		inputs:	[]logCall {
			logCall{f: Debug, args: []any{0, `DEBUG`} },
			logCall{f: Info, args: []any{1, `INFO`} },
			logCall{f: Warn, args: []any{2, `WARNING`} },
			logCall{f: Err, args: []any{3, `ERROR ` + errIsOk} },
		},
		expected: []string {
			stubApp + `[` + stubPID + `]: <D> Test #0 - DEBUG log message`,
			stubApp + `[` + stubPID + `]: <WRN> Test Reopen() #1`,
			stubApp + `[` + stubPID + `]: Test #1 - INFO log message`,
			stubApp + `[` + stubPID + `]: <WRN> Test Reopen() #2`,
			stubApp + `[` + stubPID + `]: <WRN> Test #2 - WARNING log message`,
			stubApp + `[` + stubPID + `]: <WRN> Test Reopen() #3`,
			stubApp + `[` + stubPID + `]: <ERR> Test #3 - ERROR ` + errIsOk + ` log message`,
		},
	},
}

var statisticTests = []logCall {
	{f: Debug, args: []any{`Statistic test - DEBUG #0`} },
	{f: Warn, args: []any{`Statistic test - WARNING #0`}, fType: tWarn },
	{f: Err, args: []any{`Statistic test - ERROR #0 ` + errIsOk}, fType: tErr },
	{f: Debug, args: []any{`Statistic test - INFO message #0`} },
	{f: Debug, args: []any{`Statistic test - DEBUG message #1`} },
	{f: Warn, args: []any{`Statistic test - WARNING #1`}, fType: tWarn },
	{f: Debug, args: []any{`Statistic test - DEBUG message #2`} },
	{f: Debug, args: []any{`Statistic test - DEBUG message #3`} },
	{f: Err, args: []any{`Statistic test - ERROR #1 ` + errIsOk}, fType: tErr },
	{f: Debug, args: []any{`Statistic test - INFO message #1`} },
	{f: Err, args: []any{`Statistic test - ERROR #2 ` + errIsOk}, fType: tErr },
	{f: Debug, args: []any{`Statistic test - INFO message #2`} },
	{f: Warn, args: []any{`Statistic test - WARNING #2`}, fType: tWarn },
	{f: Debug, args: []any{`Statistic test - INFO message #3`} },
}
