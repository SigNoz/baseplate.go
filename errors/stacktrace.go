package errors

import (
	"fmt"
	"runtime"
	"strings"
)

type stacktrace []uintptr

type rawStacktrace string

func newStacktrace() stacktrace {
	pc := make([]uintptr, 16)

	// Skips runtime.Callers, newStacktrace and the constructor, so the trace starts where the error was created.
	n := runtime.Callers(3, pc)

	return stacktrace(pc[:n:n])
}

func (s stacktrace) String() string {
	var buf strings.Builder

	frames := runtime.CallersFrames(s)
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)

		if !more {
			break
		}
	}

	return buf.String()
}

func (r rawStacktrace) String() string {
	return string(r)
}
