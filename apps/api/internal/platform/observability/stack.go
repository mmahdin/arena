package observability

import (
	"runtime"
	"strconv"
	"strings"
)

// SafeStack captures only source locations. runtime/debug.Stack prints argument
// representations which can contain private data; never send it to telemetry.
func SafeStack(skip int) string {
	var pcs [32]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var b strings.Builder
	for i := 0; i < 24; i++ {
		frame, more := frames.Next()
		if frame.File != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(sourceFile(frame.File))
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(frame.Line))
		}
		if !more {
			break
		}
	}
	return b.String()
}
