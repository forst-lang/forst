package transformerts

import (
	"fmt"
	"go/token"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidateForstPackageName rejects package names that cannot transpile to Go or collide with infra subpaths.
func ValidateForstPackageName(name string) error {
	if name == "" {
		return fmt.Errorf("generate: Forst package name must not be empty")
	}
	if strings.Contains(name, "$") {
		return fmt.Errorf(
			"generate: Forst package %q must not contain '$' (reserved for compiler infra subpaths like %s)",
			name, DefaultInfraTestingSubpath,
		)
	}
	// Efficient manual validation: first rune alpha/_; all runes alnum/_
	r, w := utf8.DecodeRuneInString(name)
	if !unicode.IsLetter(r) && r != '_' {
		return fmt.Errorf(
			"generate: Forst package %q is not a valid Go package name (must start with a letter or underscore)", name,
		)
	}
	for _, r := range name[w:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf(
				"generate: Forst package %q is not a valid Go package name (use letters, digits, and underscores only)", name,
			)
		}
	}
	if token.IsKeyword(name) {
		return fmt.Errorf(
			"generate: Forst package %q is a reserved Go keyword and cannot be used as a package name",
			name,
		)
	}
	return nil
}

// PackageNamespaceExport is the JS identifier for the compiler-owned package handle.
// Promise mode exports it as a namespace factory const; Effect mode as an Effect.Service class.
func PackageNamespaceExport(forstPackage string) string {
	return "$" + forstPackage
}

// GeneratedTypeExport is the TS export name for a Forst type or domain error class.
func GeneratedTypeExport(forstName string) string {
	return "$" + forstName
}

// GeneratedFailureAliasExport names a compiler-owned per-function failure union alias.
func GeneratedFailureAliasExport(fnName string) string {
	return "$" + fnName + "Failure"
}

// coreNamespaceIndexAlias is the index-only binding for a core namespace factory import.
// Effect mode index also imports the homonymous service class from pkg/, so core uses this alias.
func coreNamespaceIndexAlias(forstPackage string) string {
	return PackageNamespaceExport(forstPackage) + "Core"
}

// ValidateForstPackageNames validates every unique non-empty package name in outputs.
func ValidateForstPackageNames(packages []string) error {
	seen := make(map[string]struct{}, len(packages))
	var names []string
	for _, pkg := range packages {
		if pkg == "" {
			continue
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		names = append(names, pkg)
	}
	sort.Strings(names)
	for _, pkg := range names {
		if err := ValidateForstPackageName(pkg); err != nil {
			return err
		}
	}
	return nil
}

// PackageNames collects unique non-empty PackageName values from outputs.
func PackageNames(outputs []*TypeScriptOutput) []string {
	seen := make(map[string]struct{}, len(outputs))
	var names []string
	for _, o := range outputs {
		if o == nil || o.PackageName == "" {
			continue
		}
		if _, ok := seen[o.PackageName]; ok {
			continue
		}
		seen[o.PackageName] = struct{}{}
		names = append(names, o.PackageName)
	}
	sort.Strings(names)
	return names
}
