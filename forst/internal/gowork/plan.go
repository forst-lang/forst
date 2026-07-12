package gowork

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/goload"
)

type LinkMode int

const (
	LinkNone LinkMode = iota
	LinkReplace
	LinkWorkspace
)

type LinkPlan struct {
	Mode      LinkMode
	GoModPath string
	Workspace string
}

// ForstRuntimeLink describes how a run sandbox links the forst runtime module.
type ForstRuntimeLink struct {
	ReplaceDir     string // local replace target (absolute)
	RequireVersion string // proxy version when no replace
}

const errForstRuntimeLinkNotFound = "forst runtime module not found: add replace forst or require forst to .forst-gomod/go.mod, or install via @forst/cli"

// ResolveForstRuntimeLink picks forst module linking for run sandboxes.
// Priority: user go.mod replace/require → binary-adjacent module → FORST_GOMOD_ROOT.
func ResolveForstRuntimeLink(boundaryRoot string) (ForstRuntimeLink, error) {
	boundaryRoot = filepath.Clean(boundaryRoot)
	userMod := goload.FindModuleRoot(boundaryRoot)
	if userMod != "" {
		if link, ok := goload.ForstModuleLinkFromGoMod(userMod); ok {
			if link.ReplaceDir != "" {
				return ForstRuntimeLink{ReplaceDir: link.ReplaceDir}, nil
			}
			if link.RequireVersion != "" {
				return ForstRuntimeLink{RequireVersion: link.RequireVersion}, nil
			}
		}
	}
	if compilerMod := goload.ForstCompilerModuleRoot(); compilerMod != "" {
		return ForstRuntimeLink{ReplaceDir: compilerMod}, nil
	}
	return ForstRuntimeLink{}, fmt.Errorf("%s", errForstRuntimeLinkNotFound)
}

// PlanForRun picks module linking for a sandbox under boundaryRoot/sessionDir.
func PlanForRun(boundaryRoot, sessionDir string, needsCompiler bool) (LinkPlan, error) {
	boundaryRoot = filepath.Clean(boundaryRoot)
	userMod := goload.FindModuleRoot(boundaryRoot)

	if !needsCompiler {
		if userMod != "" {
			return LinkPlan{Mode: LinkNone}, nil
		}
		return LinkPlan{Mode: LinkNone}, nil
	}
	link, err := ResolveForstRuntimeLink(boundaryRoot)
	if err != nil {
		return LinkPlan{}, err
	}

	goMod := filepath.Join(sessionDir, "go.mod")
	if link.ReplaceDir != "" && userMod != "" && filepath.Clean(userMod) == filepath.Clean(link.ReplaceDir) {
		return LinkPlan{Mode: LinkNone, GoModPath: goMod}, nil
	}

	// Mode A: temp module with replace forst =>
	return LinkPlan{
		Mode:      LinkReplace,
		GoModPath: goMod,
	}, nil
}

// ChildEnv returns environment for child go subprocesses.
func ChildEnv(base []string, plan LinkPlan, boundaryRoot string) []string {
	out := stripEnvPrefixes(base,
		"GOWORK=",
		"GOMOD=",
	)
	// Strip GOFLAGS that break replace sandboxes
	filtered := out[:0]
	for _, e := range out {
		if strings.HasPrefix(e, "GOFLAGS=") && strings.Contains(e, "-mod=readonly") {
			continue
		}
		filtered = append(filtered, e)
	}
	out = filtered

	switch plan.Mode {
	case LinkWorkspace:
		if plan.Workspace != "" {
			out = append(out, "GOWORK="+plan.Workspace)
		}
	case LinkReplace, LinkNone:
		// Explicitly clear inherited GOWORK
		out = append(out, "GOWORK=off")
	}
	if boundaryRoot != "" {
		out = append(out, "FORST_BOUNDARY_ROOT="+boundaryRoot)
	}
	return out
}

func stripEnvPrefixes(env []string, prefixes ...string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, p := range prefixes {
			if strings.HasPrefix(e, p) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, e)
		}
	}
	return out
}

// PackageReplace maps a Go import path to a directory holding generated .go for that package.
type PackageReplace struct {
	ImportPath string
	Dir        string // absolute path to generated package dir
}

// WriteTestGoMod writes a temp test module go.mod with per-package replace directives.
func WriteTestGoMod(path, compilerMod string, replaces []PackageReplace, testImportPath string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	modDir := filepath.Dir(path)
	var b strings.Builder
	b.WriteString("module forst.test.temp\n\ngo 1.26.0\n\n")
	if compilerMod != "" {
		b.WriteString("replace forst => " + filepath.ToSlash(compilerMod) + "\n")
	}
	seen := make(map[string]struct{})
	for _, rep := range replaces {
		if rep.ImportPath == "" || rep.Dir == "" {
			continue
		}
		if _, ok := seen[rep.ImportPath]; ok {
			continue
		}
		seen[rep.ImportPath] = struct{}{}
		rel, err := filepath.Rel(modDir, rep.Dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "replace %s => %s\n", rep.ImportPath, filepath.ToSlash(rel))
	}
	b.WriteString("\nrequire (\n")
	if compilerMod != "" {
		b.WriteString("\tforst v0.0.0\n")
	}
	if testImportPath != "" {
		fmt.Fprintf(&b, "\t%s v0.0.0\n", testImportPath)
	}
	for imp := range seen {
		if imp == testImportPath {
			continue
		}
		fmt.Fprintf(&b, "\t%s v0.0.0\n", imp)
	}
	b.WriteString(")\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// AppendGoModReplaces appends replace/require entries for generated packages to an existing go.mod.
func AppendGoModReplaces(path string, replaces []PackageReplace) error {
	if len(replaces) == 0 {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	modDir := filepath.Dir(path)
	lines := strings.Split(string(raw), "\n")
	seen := make(map[string]struct{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "replace ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				seen[fields[1]] = struct{}{}
			}
		}
	}
	var extra []string
	for _, rep := range replaces {
		if rep.ImportPath == "" || rep.Dir == "" {
			continue
		}
		if _, ok := seen[rep.ImportPath]; ok {
			continue
		}
		rel, err := filepath.Rel(modDir, rep.Dir)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != "." && !strings.HasPrefix(rel, "./") && !strings.HasPrefix(rel, "../") {
			rel = "./" + rel
		}
		extra = append(extra, fmt.Sprintf("replace %s => %s", rep.ImportPath, rel))
	}
	if len(extra) == 0 {
		return nil
	}
	out := strings.TrimRight(string(raw), "\n") + "\n"
	for _, line := range extra {
		out += line + "\n"
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

// WriteRunGoMod writes a temp module go.mod linking the forst runtime.
func WriteRunGoMod(path string, forstLink ForstRuntimeLink, userModulePath, userModuleDir string) error {
	if forstLink.ReplaceDir == "" && forstLink.RequireVersion == "" {
		return fmt.Errorf("%s", errForstRuntimeLinkNotFound)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	modDir := filepath.Dir(path)
	var b strings.Builder
	b.WriteString("module forst.run.temp\n\ngo 1.26.0\n\n")
	if forstLink.ReplaceDir != "" {
		relCompiler, err := filepath.Rel(modDir, forstLink.ReplaceDir)
		if err != nil {
			return err
		}
		b.WriteString("replace forst => " + filepath.ToSlash(relCompiler) + "\n")
	}
	if userModulePath != "" && userModuleDir != "" {
		relUser, relErr := filepath.Rel(modDir, userModuleDir)
		if relErr != nil {
			return relErr
		}
		fmt.Fprintf(&b, "replace %s => %s\n", userModulePath, filepath.ToSlash(relUser))
	}
	b.WriteString("\nrequire forst ")
	if forstLink.RequireVersion != "" && forstLink.ReplaceDir == "" {
		b.WriteString(forstLink.RequireVersion)
	} else {
		b.WriteString("v0.0.0")
	}
	b.WriteString("\n")
	if userModulePath != "" {
		fmt.Fprintf(&b, "require %s v0.0.0\n", userModulePath)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
