package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"forst/internal/gowork"
)

// BuildGoProgramInSandbox compiles generated Go sources to binPath without running.
func BuildGoProgramInSandbox(mainGoPath, binPath, boundaryRoot string) error {
	return BuildGoProgramInSandboxWithTarget(mainGoPath, binPath, boundaryRoot, "", "")
}

// BuildGoProgramInSandboxWithTarget compiles generated Go sources for an optional GOOS/GOARCH.
func BuildGoProgramInSandboxWithTarget(mainGoPath, binPath, boundaryRoot, goos, goarch string) error {
	dir, sources, err := runGoSourceFiles(mainGoPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", append([]string{"build", "-o", binPath}, sources...)...)
	cmd.Dir = dir
	env := os.Environ()
	if goos != "" {
		env = appendRunEnvVar(env, "GOOS", goos)
	}
	if goarch != "" {
		env = appendRunEnvVar(env, "GOARCH", goarch)
	}
	if (goos != "" && goos != runtime.GOOS) || (goarch != "" && goarch != runtime.GOARCH) {
		env = appendRunEnvVar(env, "CGO_ENABLED", "0")
	}
	if boundaryRoot != "" {
		env = setRunEnvBoundaryRoot(env, boundaryRoot)
		needsCompiler := tempDirHasForstCompanionFiles(filepath.Dir(mainGoPath))
		plan, _ := gowork.PlanForRun(boundaryRoot, filepath.Dir(mainGoPath), needsCompiler)
		env = gowork.ChildEnv(env, plan, boundaryRoot)
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build in sandbox: %w\n%s", err, out)
	}
	return nil
}
