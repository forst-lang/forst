package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"forst/internal/codegen/layout"
	"forst/internal/goload"
	"forst/internal/gowork"
	"forst/internal/logger"
	"forst/nodert"
	transformer_go "forst/internal/transformer/go"
	"io"
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
	dir, runSources, err := runGoSourceFiles(outputPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", append([]string{"run"}, runSources...)...)
	cmd.Dir = dir
	env := os.Environ()
	if boundaryRoot != "" {
		env = setRunEnvBoundaryRoot(env, boundaryRoot)
		needsCompiler := tempDirHasForstCompanionFiles(filepath.Dir(outputPath))
		plan, _ := gowork.PlanForRun(boundaryRoot, filepath.Dir(outputPath), needsCompiler)
		env = gowork.ChildEnv(env, plan, boundaryRoot)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		return formatRunProgramError(err, stderr.String())
	}
	return nil
}

// formatRunProgramError wraps go run exit failures with actionable hints for forst run.
func formatRunProgramError(err error, stderr string) error {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}
	code := exitErr.ExitCode()
	hint := "see stderr above for details"
	if strings.Contains(stderr, "go.mod") || strings.Contains(stderr, "module not found") {
		hint = "Go module linking failed — add replace forst or require forst to .forst-gomod/go.mod, or install via @forst/cli"
	} else if code == 1 {
		hint = "node runtime or ensure check failed — verify tsx, @forst/node-runtime, and host shim args in ftconfig.json"
	}
	return fmt.Errorf("generated program exited with code %d (%s)", code, hint)
}

func runGoSourceFiles(outputPath string) (workDir string, sources []string, err error) {
	tempDir := filepath.Dir(outputPath)
	if _, statErr := os.Stat(filepath.Join(tempDir, "go.mod")); statErr == nil {
		return tempDir, []string{"."}, nil
	}
	sources, err = runGoSourceFilesList(outputPath)
	if err != nil {
		return "", nil, err
	}
	modRoot := goModuleRootForRun(outputPath)
	if modRoot == "" {
		return "", nil, fmt.Errorf("no module root for generated program")
	}
	abs := make([]string, len(sources))
	for i, rel := range sources {
		abs[i] = filepath.Join(tempDir, filepath.FromSlash(rel))
	}
	return modRoot, abs, nil
}

func runGoSourceFilesList(outputPath string) ([]string, error) {
	tempDir := filepath.Dir(outputPath)
	var matches []string
	err := filepath.WalkDir(tempDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".go") {
			return nil
		}
		rel, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		matches = append(matches, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk run sources: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no go files in %s", tempDir)
	}
	sort.Strings(matches)
	return matches, nil
}

// BuildGoProgram writes main and optional companion Go files and runs `go build` to verify they compile.
func BuildGoProgram(mainCode, nodeRuntimeCode, invokeServerCode string) error {
	outputPath, err := CreateTempOutputFiles(mainCode, nodeRuntimeCode, invokeServerCode, nil, nil, "")
	if err != nil {
		return err
	}
	tempDir := filepath.Dir(outputPath)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dir, sources, err := runGoSourceFiles(outputPath)
	if err != nil {
		return err
	}
	if dir == "" {
		return fmt.Errorf("go build: no module root for generated program")
	}
	outBin := filepath.Join(tempDir, "forst-build")
	cmd := exec.Command("go", append([]string{"build", "-o", outBin}, sources...)...)
	cmd.Dir = dir
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
	filtered = append(filtered, prefix+boundaryRoot)
	if v := os.Getenv(nodert.EnvNodeAttachOnly); v != "" {
		filtered = appendRunEnvVar(filtered, nodert.EnvNodeAttachOnly, v)
	}
	return filtered
}

func appendRunEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return append(out, prefix+value)
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

const errForstCompilerModuleRequired = "forst run: node runtime / invoke server require the forst runtime module; add replace forst or require forst to .forst-gomod/go.mod, or install via @forst/cli"

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
func CreateTempOutputFiles(mainCode, nodeRuntimeCode, invokeServerCode string, extraPackages map[string]string, extraImportPaths map[string]string, boundaryRoot string) (string, error) {
	needsCompiler := needsForstCompilerModule(nodeRuntimeCode, invokeServerCode)
	var tempDir string
	var goModPath string
	if boundaryRoot != "" {
		layoutRoot := layout.NewRoot(boundaryRoot)
		runBase := filepath.Join(layoutRoot.Boundary, ".forst", "run")
		if err := os.MkdirAll(runBase, 0o755); err != nil {
			return "", err
		}
		var err error
		tempDir, err = mkdirTemp(runBase, "forst-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp directory: %v", err)
		}
		goModPath = filepath.Join(tempDir, "go.mod")
	} else {
		baseDir, err := runTempBaseDir(needsCompiler)
		if err != nil {
			return "", err
		}
		tempDir, err = mkdirTemp(baseDir, "forst-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp directory: %v", err)
		}
		goModPath = filepath.Join(tempDir, "go.mod")
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", err
	}

	if needsCompiler && boundaryRoot != "" {
		forstLink, err := gowork.ResolveForstRuntimeLink(boundaryRoot)
		if err != nil {
			return "", err
		}
		userMod := goload.FindModuleRoot(boundaryRoot)
		userPath := goload.ModulePath(userMod)
		if err := gowork.WriteRunGoMod(goModPath, forstLink, userPath, userMod); err != nil {
			return "", err
		}
	}

	outputPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(outputPath, []byte(mainCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}
	for pkg, code := range extraPackages {
		pkgDir := filepath.Join(tempDir, pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			return "", err
		}
		pkgPath := filepath.Join(pkgDir, pkg+layout.SuffixGen)
		if err := os.WriteFile(pkgPath, []byte(code), 0644); err != nil {
			return "", fmt.Errorf("write extra package %q: %w", pkg, err)
		}
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
	if needsCompiler && boundaryRoot != "" {
		if err := tidyRunSandboxGoMod(goModPath, boundaryRoot); err != nil {
			return "", err
		}
	}
	return outputPath, nil
}

func tidyRunSandboxGoMod(goModPath, boundaryRoot string) error {
	dir := filepath.Dir(goModPath)
	env := os.Environ()
	if boundaryRoot != "" {
		env = gowork.ChildEnv(env, gowork.LinkPlan{Mode: gowork.LinkReplace}, boundaryRoot)
	}
	for _, args := range [][]string{
		{"go", "get", "forst@v0.0.0"},
		{"go", "mod", "tidy"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s in run sandbox: %w\n%s", strings.Join(args, " "), err, out)
		}
	}
	return nil
}

// CreateTempOutputFilesLegacy preserves the old 3-arg signature for gradual migration.
func CreateTempOutputFilesLegacy(mainCode, nodeRuntimeCode, invokeServerCode string) (string, error) {
	return CreateTempOutputFiles(mainCode, nodeRuntimeCode, invokeServerCode, nil, nil, "")
}

// CreateTempOutputFile creates a temporary directory and file for the output.
// When running inside the Forst Go module, files are placed under .forst/run so
// generated code may import forst/nodert during `go run`.
func CreateTempOutputFile(code string) (string, error) {
	return CreateTempOutputFiles(code, "", "", nil, nil, "")
}
