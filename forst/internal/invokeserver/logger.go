package invokeserver

import (
	"fmt"
	"os"
)

// StderrLogger writes invoke server logs to os.Stderr so they appear under forst run.
type StderrLogger struct{}

func (StderrLogger) Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}

func (StderrLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}

func (StderrLogger) Debugf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}
