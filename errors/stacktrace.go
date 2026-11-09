package errors

import (
	"fmt"
	"runtime"
)

type stacktrace []uintptr

type rawStacktrace string

func newStacktrace() stacktrace {
	pc := make([]uintptr, 16)

	// Skips runtime.Callers, newStacktrace and the constructor, so the trace
	// starts where the error was created.
	n := runtime.Callers(3, pc)

	return stacktrace(pc[:n:n])
}

func (s stacktrace) String() string {
	var buf []byte

	frames := runtime.CallersFrames(s)
	for {
		frame, more := frames.Next()
		buf = fmt.Appendf(buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)

		if !more {
			break
		}
	}

	return string(buf)
}

func (r rawStacktrace) String() string {
	return string(r)
}
