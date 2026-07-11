package compiler

import (
	"errors"
	"fmt"
	"forst/internal/goload"
	"forst/internal/logger"
	"forst/nodert"
	transformer_go "forst/internal/transformer/go"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

var mkdirTemp = os.MkdirTemp

// Compiler represents the Forst compiler and its arguments.
type Compiler struct {
	Args Args
	log  *logrus.Logger
}

func New(args Args, log *logrus.Logger) *Compiler {
	if log == nil {
		log = logger.New()
	}

	return &Compiler{
		Args: args,
		log:  log,
	}
}

func (c *Compiler) readSourceFile() ([]byte, error) {
	source, err := os.ReadFile(c.Args.FilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	return source, nil
}

func getMemStats() runtime.MemStats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem
}

func RunGoProgram(outputPath string, boundaryRoot string) error {
	runSources, err := runGoSourceFiles(outputPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", append([]string{"run"}, runSources...)...)
	if dir := goModuleRootForRun(outputPath); dir != "" {
		cmd.Dir = dir
	}
	if boundaryRoot != "" {
		cmd.Env = setRunEnvBoundaryRoot(os.Environ(), boundaryRoot)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return formatRunProgramError(err)
	}
	return nil
}

// formatRunProgramError wraps go run exit failures with actionable hints for forst run.
func formatRunProgramError(err error) error {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}
	code := exitErr.ExitCode()
	hint := "see stderr above for details"
	if code == 1 {
		hint = "node runtime or ensure check failed — verify tsx, @forst/node-runtime, and host shim args in ftconfig.json"
	}
	return fmt.Errorf("generated program exited with code %d (%s)", code, hint)
}

func runGoSourceFiles(outputPath string) ([]string, error) {
	tempDir := filepath.Dir(outputPath)
	matches, err := filepath.Glob(filepath.Join(tempDir, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("glob run sources: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no go files in %s", tempDir)
	}
	sort.Strings(matches)
	return matches, nil
}

// BuildGoProgram writes main and optional companion Go files and runs `go build` to verify they compile.
func BuildGoProgram(mainCode, nodeRuntimeCode, invokeServerCode string) error {
	outputPath, err := CreateTempOutputFiles(mainCode, nodeRuntimeCode, invokeServerCode)
	if err != nil {
		return err
	}
	tempDir := filepath.Dir(outputPath)
	defer func() { _ = os.RemoveAll(tempDir) }()

	sources, err := runGoSourceFiles(outputPath)
	if err != nil {
		return err
	}
	modRoot := goModuleRootForRun(outputPath)
	if modRoot == "" {
		return fmt.Errorf("go build: no module root for generated program")
	}
	outBin := filepath.Join(tempDir, "forst-build")
	cmd := exec.Command("go", append([]string{"build", "-o", outBin}, sources...)...)
	cmd.Dir = modRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, out)
	}
	return nil
}

func setRunEnvBoundaryRoot(env []string, boundaryRoot string) []string {
	filtered := make([]string, 0, len(env)+1)
	prefix := nodert.EnvBoundaryRoot + "="
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, prefix+boundaryRoot)
}

// RunBoundaryRoot returns the ftconfig project root to pass when running generated Go.
func RunBoundaryRoot(args Args) string {
	if args.PackageRoot != "" {
		return args.PackageRoot
	}
	if args.FilePath == "" {
		return ""
	}
	abs, err := filepath.Abs(args.FilePath)
	if err != nil {
		return ""
	}
	return filepath.Dir(abs)
}

const errForstCompilerModuleRequired = "forst run: node runtime / invoke server require the Forst Go module; set FORST_GOMOD_ROOT or reinstall the compiler"

func needsForstCompilerModule(nodeRuntimeCode, invokeServerCode string) bool {
	return nodeRuntimeCode != "" || invokeServerCode != ""
}

func tempDirHasForstCompanionFiles(tempDir string) bool {
	for _, stem := range []string{
		transformer_go.ForstNodeRuntimeFileName(),
		transformer_go.ForstInvokeServerFileName(),
	} {
		if _, err := os.Stat(filepath.Join(tempDir, stem+".go")); err == nil {
			return true
		}
	}
	return false
}

func goModuleRootForRun(outputPath string) string {
	if outputPath != "" {
		tempDir := filepath.Dir(outputPath)
		if tempDirHasForstCompanionFiles(tempDir) {
			if root := goload.ForstCompilerModuleRoot(); root != "" {
				return root
			}
		}
		if root := goload.FindModuleRoot(tempDir); root != "" {
			if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
				return root
			}
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return goload.FindModuleRoot(wd)
	}
	return ""
}

func runTempBaseDir(needsCompilerModule bool) (string, error) {
	if needsCompilerModule {
		root := goload.ForstCompilerModuleRoot()
		if root == "" {
			return "", fmt.Errorf("%s", errForstCompilerModuleRequired)
		}
		runDir := filepath.Join(root, ".forst", "run")
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return "", fmt.Errorf("create run dir: %w", err)
		}
		return runDir, nil
	}
	if wd, err := os.Getwd(); err == nil {
		if modRoot := goload.FindModuleRoot(wd); modRoot != "" {
			runDir := filepath.Join(modRoot, ".forst", "run")
			if err := os.MkdirAll(runDir, 0o755); err == nil {
				return runDir, nil
			}
		}
	}
	return "", nil
}

func (c *Compiler) reportPhase(phase string) {
	if c.Args.ReportPhases {
		c.log.Info(phase)
	}
}

// CreateTempOutputFiles writes main and optional companion Go files into a temp dir for `go run`.
func CreateTempOutputFiles(mainCode, nodeRuntimeCode, invokeServerCode string) (string, error) {
	needsCompiler := needsForstCompilerModule(nodeRuntimeCode, invokeServerCode)
	baseDir, err := runTempBaseDir(needsCompiler)
	if err != nil {
		return "", err
	}
	tempDir, err := mkdirTemp(baseDir, "forst-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	outputPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(outputPath, []byte(mainCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}
	if nodeRuntimeCode != "" {
		runtimePath := filepath.Join(tempDir, transformer_go.ForstNodeRuntimeFileName()+".go")
		if err := os.WriteFile(runtimePath, []byte(nodeRuntimeCode), 0644); err != nil {
			return "", fmt.Errorf("failed to write node runtime temp file: %v", err)
		}
	}
	if invokeServerCode != "" {
		invokePath := filepath.Join(tempDir, transformer_go.ForstInvokeServerFileName()+".go")
		if err := os.WriteFile(invokePath, []byte(invokeServerCode), 0644); err != nil {
			return "", fmt.Errorf("failed to write invoke server temp file: %v", err)
		}
	}
	return outputPath, nil
}

// CreateTempOutputFile creates a temporary directory and file for the output.
// When running inside the Forst Go module, files are placed under .forst/run so
// generated code may import forst/nodert during `go run`.
func CreateTempOutputFile(code string) (string, error) {
	return CreateTempOutputFiles(code, "", "")
}
