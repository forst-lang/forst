package testrunner

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func stubGoTestSuccess(t *testing.T) {
	t.Helper()
	swapHook(t, &execGoTestHook, execGoTestFn(func(*exec.Cmd) error { return nil }))
}

func stubGoTestFailImport(t *testing.T, failImport string) {
	t.Helper()
	swapHook(t, &execGoTestHook, execGoTestFn(func(cmd *exec.Cmd) error {
		if len(cmd.Args) == 0 {
			return nil
		}
		imp := cmd.Args[len(cmd.Args)-1]
		if strings.Contains(imp, failImport) {
			return exitStatusErr(1)
		}
		return nil
	}))
}

func exitStatusErr(code int) error {
	c := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
	return c.Run()
}
