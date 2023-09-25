package peer

type Debugger interface {
	Info(string, ...any)
	Error(string, ...any)
}

var _ Debugger = nilDebugger{}

type nilDebugger struct{}

func (d nilDebugger) Info(_ string, _ ...any)  {}
func (d nilDebugger) Error(_ string, _ ...any) {}
