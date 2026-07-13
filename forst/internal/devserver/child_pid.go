package devserver

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"forst/nodert"

	"github.com/sirupsen/logrus"
)

const goChildPIDFileName = "go-child.pid"

func goChildPIDPath(boundaryRoot string) string {
	return filepath.Join(filepath.Clean(boundaryRoot), ".forst", goChildPIDFileName)
}

// WriteGoChildPID records the background go run child pid for orphan cleanup.
func WriteGoChildPID(boundaryRoot string, pid int) error {
	if pid <= 0 {
		return nil
	}
	path := goChildPIDPath(boundaryRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

// ClearGoChildPID removes the tracked go child pid file.
func ClearGoChildPID(boundaryRoot string) error {
	err := os.Remove(goChildPIDPath(boundaryRoot))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ReadGoChildPID returns the pid from boundaryRoot/.forst/go-child.pid, or 0 when missing.
func ReadGoChildPID(boundaryRoot string) int {
	raw, err := os.ReadFile(goChildPIDPath(boundaryRoot))
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

func goChildProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// ReapOrphanedGoChild terminates a stale go child from a prior forst dev session.
// currentPID is the live tracked child pid (0 when none); it is never killed.
func ReapOrphanedGoChild(boundaryRoot string, currentPID int, log *logrus.Logger) {
	pid := ReadGoChildPID(boundaryRoot)
	if pid <= 0 {
		return
	}
	if currentPID > 0 && pid == currentPID {
		return
	}
	if !goChildProcessAlive(pid) {
		_ = ClearGoChildPID(boundaryRoot)
		return
	}
	if log != nil {
		log.Infof("Reaping orphaned go child pid=%d (invoke port should be free)", pid)
	}
	if err := nodert.TerminateHostPID(pid, nodert.DefaultHostShutdownGrace()); err != nil && log != nil {
		log.Warnf("reap orphaned go child pid=%d: %v", pid, err)
	}
	_ = ClearGoChildPID(boundaryRoot)
}

// recordStartedChild writes go-child.pid when the child has a known pid.
func recordStartedChild(boundaryRoot string, child *runningChild) {
	if child == nil || child.pid <= 0 {
		return
	}
	if err := WriteGoChildPID(boundaryRoot, child.pid); err != nil {
		// best effort; orphan reap may still use stale file from prior session
		_ = err
	}
}

func formatInvokePortShiftLog(preferred, invokePort string) string {
	return FormatInvokePortShiftLog(preferred, invokePort)
}

// FormatInvokePortShiftLog formats invoke port shift log with reserved port skip info.
func FormatInvokePortShiftLog(preferred, invokePort string) string {
	msg := fmt.Sprintf("invoke listen port %s (configured %s in use)", invokePort, preferred)
	preferredNum, err1 := strconv.Atoi(preferred)
	chosenNum, err2 := strconv.Atoi(invokePort)
	if err1 != nil || err2 != nil || chosenNum <= preferredNum {
		return msg
	}
	reserved := reservedInvokePorts()
	var skipped []string
	for p := preferredNum + 1; p < chosenNum; p++ {
		port := strconv.Itoa(p)
		if _, ok := reserved[port]; ok {
			skipped = append(skipped, port)
		}
	}
	if len(skipped) > 0 {
		msg += fmt.Sprintf(", skipped reserved %s", strings.Join(skipped, ","))
	}
	return msg
}
