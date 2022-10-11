package log

type logFunction func(string, ...any)
type logCall struct {
	f		logFunction
	args	[]any
}

const stubLogFormat = `[#%d] %s log message`

var tests = []struct {
	flags		int
	pre		func()
	inputs		[]logCall
	expected	[]string
	post	func()
}{
	{
		pre:	func() {
			SetDebug(true)
		},
		flags:	NoFlags,
		inputs:	[]logCall {
			logCall{f: Debug, args: []any{0, "DEBUG"} },
			logCall{f: Info, args: []any{1, "INFO"} },
			logCall{f: Warn, args: []any{2, "WARNING"} },
			logCall{f: Err, args: []any{3, "ERROR"} },
		},
		expected: []string {
			stubApp + `[` + stubPID + `]: <D> [#0] DEBUG log message`,
			stubApp + `[` + stubPID + `]: [#1] INFO log message`,
			stubApp + `[` + stubPID + `]: <WRN> [#2] WARNING log message`,
			stubApp + `[` + stubPID + `]: <ERR> [#3] ERROR log message`,
		},
		post:	func() {
			SetDebug(true)
		},
	},
}
