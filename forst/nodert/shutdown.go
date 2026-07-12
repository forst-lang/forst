package nodert

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForShutdown blocks until SIGINT, SIGTERM, or SIGQUIT, then shuts down the Node supervisor.
// Generated host-mode binaries call this from main to keep the process alive while the app shim runs.
func WaitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigCh
	_ = Shutdown()
}
