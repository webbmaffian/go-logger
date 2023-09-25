package peer

type Debugger interface {
	Info(string, ...any)
	Notice(string, ...any)
	Debug(string, ...any)
	Error(error)
}

var _ Debugger = nilDebugger{}

type nilDebugger struct{}

func (d nilDebugger) Info(_ string, _ ...any)   {}
func (d nilDebugger) Notice(_ string, _ ...any) {}
func (d nilDebugger) Debug(_ string, _ ...any)  {}
func (d nilDebugger) Error(_ error)             {}
