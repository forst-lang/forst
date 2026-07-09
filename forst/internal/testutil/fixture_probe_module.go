package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

// JobsExecFtSource is a minimal Forst file that imports os/exec and calls exec.Command.
const JobsExecFtSource = `package jobs

import "os/exec"

func runCmd(): Int {
  cmd := exec.Command("true")
  err := cmd.Run()
  if err != nil {
    return 1
  }
  return 0
}
`

const jobsWriteFtSource = `package jobs

import "os"

func writeFile(): Error {
  data := []byte("ok")
  return os.WriteFile("/tmp/forst-jobs-write.txt", data, 420)
}
`

// WriteProbeModuleFixture creates a temp module with go.mod and internal/jobs/exec.ft.
// When withWrite is true, internal/jobs/write.ft is also written (os.WriteFile + []byte).
// Returns the module root and the absolute path to exec.ft.
func WriteProbeModuleFixture(tb testing.TB, withWrite bool) (moduleRoot, execFtPath string) {
	tb.Helper()
	root := tb.TempDir()
	jobsDir := filepath.Join(root, "internal", "jobs")
	if err := os.MkdirAll(jobsDir, 0o755); err != nil {
		tbFail(tb, err)
		return "", ""
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(testmod.GoModContent("probemod")), 0o644); err != nil {
		tbFail(tb, err)
		return "", ""
	}
	execFtPath = filepath.Join(jobsDir, "exec.ft")
	if err := os.WriteFile(execFtPath, []byte(JobsExecFtSource), 0o644); err != nil {
		tbFail(tb, err)
		return "", ""
	}
	if withWrite {
		if err := os.WriteFile(filepath.Join(jobsDir, "write.ft"), []byte(jobsWriteFtSource), 0o644); err != nil {
			tbFail(tb, err)
			return "", ""
		}
	}
	return root, execFtPath
}

// ProbeExecFtSource is an alias for JobsExecFtSource (legacy test name).
const ProbeExecFtSource = JobsExecFtSource
