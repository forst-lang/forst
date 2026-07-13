package modulecheck

import (
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

var passCount atomic.Uint64

// ResetPassCount clears the modulecheck invocation counter (call at reload start).
func ResetPassCount() {
	passCount.Store(0)
}

// PassCount returns how many times CheckModuleProviders has run since the last reset.
func PassCount() uint64 {
	return passCount.Load()
}

// ResetPassCountForTest clears the counter for unit tests.
func ResetPassCountForTest() {
	ResetPassCount()
}

func incPassCount(log *logrus.Logger) uint64 {
	n := passCount.Add(1)
	if log != nil {
		log.WithField("modulecheck_pass", n).Debug("CheckModuleProviders")
	}
	return n
}
