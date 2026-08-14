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

// ServiceClassName returns the PascalCase Effect.Service class name for a Forst package.
// user_auth and userAuth both become UserAuth (collision caught by ValidateServiceClassNames).
func ServiceClassName(forstPackage string) string {
	if forstPackage == "" {
		return ""
	}
	parts := splitPackageNameParts(forstPackage)
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(part)
		if r == utf8.RuneError && size == 0 {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
		if size < len(part) {
			b.WriteString(strings.ToLower(part[size:]))
		}
	}
	return b.String()
}

// ValidateServiceClassNames fails when two Forst packages collapse to the same service class.
func ValidateServiceClassNames(packages []string) error {
	byClass := make(map[string][]string)
	seen := make(map[string]struct{})
	for _, pkg := range packages {
		if pkg == "" {
			continue
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		class := ServiceClassName(pkg)
		byClass[class] = append(byClass[class], pkg)
	}
	var classes []string
	for class, pkgs := range byClass {
		if len(pkgs) > 1 {
			classes = append(classes, class)
		}
	}
	sort.Strings(classes)
	if len(classes) == 0 {
		return nil
	}
	class := classes[0]
	pkgs := byClass[class]
	sort.Strings(pkgs)
	if class == "" {
		return fmt.Errorf(
			"generate: Forst packages %q and %q both map to an empty Effect service class\n"+
				"  rename one package so its name contains at least one letter or digit",
			pkgs[0], pkgs[1],
		)
	}
	return fmt.Errorf(
		"generate: Forst packages %q and %q both produce Effect service class %s\n"+
			"  rename one of the packages so their PascalCase forms differ",
		pkgs[0], pkgs[1], class,
	)
}

func splitPackageNameParts(name string) []string {
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		parts = append(parts, cur.String())
		cur.Reset()
	}
	prevLower := false
	for _, r := range name {
		switch {
		case r == '_' || r == '-' || r == '.':
			flush()
			prevLower = false
		case unicode.IsUpper(r) && prevLower:
			flush()
			cur.WriteRune(r)
			prevLower = false
		default:
			cur.WriteRune(r)
			prevLower = unicode.IsLower(r)
		}
	}
	flush()
	return parts
}
