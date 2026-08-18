package ftconfig

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"forst/internal/codegen/layout"
)

// DefaultPackageName is the fixed generate.packageName default.
// It is never derived from the adopter's package.json or a directory name.
const DefaultPackageName = "@forst/gen"

// ReservedErrorsPackageName is the npm package name reserved for the shared error catalog.
const ReservedErrorsPackageName = "@forst/errors"

const defaultGenerateOutDir = ".forst/client"

// npmPackageNamePattern matches a valid npm package name (scoped or unscoped).
// Lowercase only; scoped form is @scope/name.
var npmPackageNamePattern = regexp.MustCompile(`^(@[a-z0-9][~a-z0-9._-]*/)?[a-z0-9][~a-z0-9._-]*$`)

// infraSubpathKeyPattern matches compiler-owned single-segment export keys ($ prefix allowed).
var infraSubpathKeyPattern = regexp.MustCompile(`^\$[a-zA-Z][a-zA-Z0-9._-]*$`)

// subpathKeyPattern matches a single-segment package.json exports subpath key for domain packages.
var subpathKeyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// EffectiveGenerateConfig merges generate defaults with cfg.Generate.
// packageName is never read from an adopter package.json.
func EffectiveGenerateConfig(cfg *Config, boundaryRoot string) GenerateConfig {
	_ = boundaryRoot
	def := defaultGenerateConfig()
	if cfg == nil {
		return def
	}
	g := cfg.Generate
	if g.PackageName == "" {
		g.PackageName = def.PackageName
	}
	if g.OutDir == "" {
		g.OutDir = def.OutDir
	}
	if g.Link == "" {
		g.Link = def.Link
	}
	if g.Emit == "" {
		g.Emit = def.Emit
	}
	if g.TestingSubpath == "" {
		g.TestingSubpath = def.TestingSubpath
	}
	return g
}

func defaultGenerateConfig() GenerateConfig {
	return GenerateConfig{
		PackageName:    DefaultPackageName,
		OutDir:         defaultGenerateOutDir,
		Link:           "auto",
		Emit:           "js",
		TestingSubpath: "$testing",
		Effect:         false,
		SSRModule:      "",
	}
}

// EffectiveOutDir resolves outDir against the ftconfig boundary root.
// The default outDir maps to layout.Root.ClientDir().
func (g GenerateConfig) EffectiveOutDir(boundaryRoot string) string {
	outDir := g.OutDir
	if outDir == "" || outDir == defaultGenerateOutDir {
		return layout.NewRoot(boundaryRoot).ClientDir()
	}
	return filepath.Join(filepath.Clean(boundaryRoot), filepath.Clean(outDir))
}

// EffectiveTestingSubpath returns the package.json export key for the testing module.
func (g GenerateConfig) EffectiveTestingSubpath() string {
	if g.TestingSubpath != "" {
		return g.TestingSubpath
	}
	return "$testing"
}

// IsEphemeral reports whether the resolved outDir sits under <boundaryRoot>/.forst.
func (g GenerateConfig) IsEphemeral(boundaryRoot string) bool {
	out := g.EffectiveOutDir(boundaryRoot)
	dotForst := filepath.Join(filepath.Clean(boundaryRoot), ".forst")
	rel, err := filepath.Rel(dotForst, out)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// ShouldLink reports whether generate should write the node_modules link.
// always always links; never never links; auto links only when IsEphemeral.
func (g GenerateConfig) ShouldLink(boundaryRoot string) bool {
	switch g.Link {
	case "always":
		return true
	case "never":
		return false
	case "auto", "":
		return g.IsEphemeral(boundaryRoot)
	default:
		return false
	}
}

// ReservedSubpaths maps compiler-owned exports subpath keys to a human-readable owner.
func (g GenerateConfig) ReservedSubpaths() map[string]string {
	key := g.TestingSubpath
	if key == "" {
		key = "$testing"
	}
	out := map[string]string{
		key:          "testing subpath",
		"$errors":     "errors subpath",
		"$transport": "transport subpath",
	}
	return out
}

// Validate checks generate config fields that can be set by an adopter.
func (g GenerateConfig) Validate() error {
	if g.Emit != "js" {
		return fmt.Errorf("generate.emit must be \"js\", got %q (\"ts\" is not supported)", g.Emit)
	}
	switch g.Link {
	case "auto", "always", "never":
	default:
		return fmt.Errorf("generate.link must be one of auto, always, never, got %q", g.Link)
	}
	if err := validateGenerateOutDir(g.OutDir); err != nil {
		return err
	}
	if err := validateNPMPackageName(g.PackageName); err != nil {
		return err
	}
	if err := validateTestingSubpath(g.TestingSubpath); err != nil {
		return err
	}
	if err := g.Go.Validate(); err != nil {
		return err
	}
	for i, p := range g.Plugins {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("generate.plugins[%d]: %w", i, err)
		}
	}
	key := g.TestingSubpath
	if key == "" {
		key = "$testing"
	}
	if key == "$errors" {
		return fmt.Errorf("generate: testingSubpath %q conflicts with infra errors subpath %q", g.TestingSubpath, "$errors")
	}
	if key == "$transport" {
		return fmt.Errorf("generate: testingSubpath %q conflicts with infra transport subpath %q", g.TestingSubpath, "$transport")
	}
	return nil
}

func validateGenerateOutDir(outDir string) error {
	if outDir == "" {
		return fmt.Errorf("generate.outDir must not be empty")
	}
	if filepath.IsAbs(outDir) {
		return fmt.Errorf("generate.outDir must be relative to the boundary root, got absolute path %q", outDir)
	}
	cleaned := filepath.Clean(outDir)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("generate.outDir %q escapes the boundary root", outDir)
	}
	return nil
}

func validateNPMPackageName(name string) error {
	if name == "" {
		return fmt.Errorf("generate.packageName must not be empty")
	}
	for _, r := range name {
		if unicode.IsUpper(r) {
			return fmt.Errorf("generate.packageName %q is not a valid npm package name (uppercase letters are not allowed)", name)
		}
	}
	if !npmPackageNamePattern.MatchString(name) {
		return fmt.Errorf("generate.packageName %q is not a valid npm package name", name)
	}
	if name == ReservedErrorsPackageName {
		return fmt.Errorf("generate.packageName %q is reserved (%s is the shared invoke/harness error package)", name, name)
	}
	return nil
}

func validateTestingSubpath(key string) error {
	if key == "" {
		return fmt.Errorf("generate.testingSubpath must not be empty")
	}
	if strings.Contains(key, "/") || strings.Contains(key, `\`) {
		return fmt.Errorf("generate.testingSubpath must be a single segment, got %q", key)
	}
	if key == "." || key == ".." {
		return fmt.Errorf("generate.testingSubpath %q is not a valid subpath key", key)
	}
	if infraSubpathKeyPattern.MatchString(key) {
		return nil
	}
	if subpathKeyPattern.MatchString(key) {
		return fmt.Errorf(
			"generate.testingSubpath %q must use a $ prefix (for example %q) so it cannot collide with Forst package names",
			key, "$testing",
		)
	}
	return fmt.Errorf("generate.testingSubpath %q is not a valid subpath key", key)
}

// Validate checks generate.go fields when Go source emission is configured.
func (g GenerateGoConfig) Validate() error {
	hasEntry := strings.TrimSpace(g.Entry) != ""
	hasOut := strings.TrimSpace(g.Out) != ""
	if !hasEntry && !hasOut && g.Root == "" {
		return nil
	}
	if hasEntry != hasOut {
		if !hasEntry {
			return fmt.Errorf("generate.go.entry is required when generate.go.out is set")
		}
		return fmt.Errorf("generate.go.out is required when generate.go.entry is set")
	}
	if g.Root != "" && !g.IsConfigured() {
		return fmt.Errorf("generate.go.root requires generate.go.entry and generate.go.out")
	}
	if filepath.IsAbs(g.Out) {
		return fmt.Errorf("generate.go.out must be relative to the boundary root, got absolute path %q", g.Out)
	}
	cleaned := filepath.Clean(g.Out)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("generate.go.out %q escapes the boundary root", g.Out)
	}
	if g.Root != "" {
		rootClean := filepath.Clean(g.Root)
		if rootClean == ".." || strings.HasPrefix(rootClean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("generate.go.root %q escapes the boundary root", g.Root)
		}
	}
	return nil
}

// IsConfigured reports whether Go source emission is configured (entry and out both set).
func (g GenerateGoConfig) IsConfigured() bool {
	return strings.TrimSpace(g.Entry) != "" && strings.TrimSpace(g.Out) != ""
}

// EffectiveGoRoot resolves generate.go.root against boundaryRoot.
func (g GenerateGoConfig) EffectiveGoRoot(boundaryRoot string) string {
	if g.Root == "" {
		return boundaryRoot
	}
	return filepath.Join(filepath.Clean(boundaryRoot), filepath.Clean(g.Root))
}

// EffectiveGoOut resolves generate.go.out against boundaryRoot.
func (g GenerateGoConfig) EffectiveGoOut(boundaryRoot string) string {
	return filepath.Join(filepath.Clean(boundaryRoot), filepath.Clean(g.Out))
}
