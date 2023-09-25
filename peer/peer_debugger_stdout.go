package peer

import "github.com/fatih/color"

var _ Debugger = debuggerStdout{}

type debuggerStdout struct{}

func DebuggerStdout() Debugger {
	return debuggerStdout{}
}

func (d debuggerStdout) Info(s string, args ...any) {
	if len(args) >= 2 {
		if b, ok := args[1].([]byte); ok {
			args[1] = string(b[20 : 20+b[19]])
		}
	}
	color.Cyan(s, args...)
}

func (d debuggerStdout) Notice(s string, args ...any) {
	color.Yellow(s, args...)
}

func (d debuggerStdout) Debug(s string, args ...any) {
	color.White(s, args...)
}

func (d debuggerStdout) Error(err error) {
	color.Red(err.Error())
}
