package forstpkg

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ValidateGoPackageLayout enforces Go's one-directory-one-package rule in both directions:
// each package name occupies one directory, and each directory holds one package name
// (with an external-test exception: package foo and package foo_test in the same directory).
func ValidateGoPackageLayout(forstPkgToFiles map[string][]string) error {
	if err := validateOneDirectoryPerPackage(forstPkgToFiles); err != nil {
		return err
	}
	return validateOnePackagePerDirectory(forstPkgToFiles)
}

// ValidateOneDirectoryPerPackage is kept for callers that only need the package→directory check.
func ValidateOneDirectoryPerPackage(forstPkgToFiles map[string][]string) error {
	return validateOneDirectoryPerPackage(forstPkgToFiles)
}

func validateOneDirectoryPerPackage(forstPkgToFiles map[string][]string) error {
	if len(forstPkgToFiles) == 0 {
		return nil
	}
	packageNames := sortedPackageNames(forstPkgToFiles)

	var b strings.Builder
	for _, pkg := range packageNames {
		files := forstPkgToFiles[pkg]
		if len(files) == 0 {
			continue
		}
		dirs := uniqueDirs(files)
		if len(dirs) <= 1 {
			continue
		}
		appendLayoutError(&b, fmt.Sprintf("package %q spans %d directories (Go requires one directory per package):", pkg, len(dirs)))
		appendDirFileListing(&b, dirs, files)
	}
	if b.Len() == 0 {
		return nil
	}
	return fmt.Errorf("%s", b.String())
}

func validateOnePackagePerDirectory(forstPkgToFiles map[string][]string) error {
	if len(forstPkgToFiles) == 0 {
		return nil
	}
	dirToPackages := make(map[string]map[string]struct{})
	dirToFiles := make(map[string][]string)
	for pkg, files := range forstPkgToFiles {
		for _, file := range files {
			dir := filepath.Clean(filepath.Dir(file))
			if dirToPackages[dir] == nil {
				dirToPackages[dir] = make(map[string]struct{})
			}
			dirToPackages[dir][pkg] = struct{}{}
			dirToFiles[dir] = append(dirToFiles[dir], file)
		}
	}

	dirs := make([]string, 0, len(dirToPackages))
	for dir := range dirToPackages {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	var b strings.Builder
	for _, dir := range dirs {
		pkgs := sortedKeys(dirToPackages[dir])
		if len(pkgs) <= 1 || allowedExternalTestPackages(pkgs) {
			continue
		}
		appendLayoutError(&b, fmt.Sprintf("directory %q contains %d packages (Go allows one package per directory): %s",
			dir, len(pkgs), strings.Join(pkgs, ", ")))
		for _, file := range sortedPaths(dirToFiles[dir]) {
			fmt.Fprintf(&b, "\n    %s", file)
		}
	}
	if b.Len() == 0 {
		return nil
	}
	return fmt.Errorf("%s", b.String())
}

// allowedExternalTestPackages reports whether pkgs is {foo, foo_test} for some foo.
func allowedExternalTestPackages(pkgs []string) bool {
	if len(pkgs) != 2 {
		return false
	}
	a, b := pkgs[0], pkgs[1]
	if strings.HasSuffix(b, "_test") && strings.TrimSuffix(b, "_test") == a {
		return true
	}
	if strings.HasSuffix(a, "_test") && strings.TrimSuffix(a, "_test") == b {
		return true
	}
	return false
}

func sortedPackageNames(forstPkgToFiles map[string][]string) []string {
	names := make([]string, 0, len(forstPkgToFiles))
	for name := range forstPkgToFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedPaths(paths []string) []string {
	out := append([]string(nil), paths...)
	sort.Strings(out)
	return out
}

func appendLayoutError(b *strings.Builder, msg string) {
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	b.WriteString(msg)
}

func appendDirFileListing(b *strings.Builder, dirs, files []string) {
	for _, dir := range dirs {
		fmt.Fprintf(b, "\n  %s", dir)
		for _, file := range files {
			if filepath.Dir(filepath.Clean(file)) == dir {
				fmt.Fprintf(b, "\n    %s", file)
			}
		}
	}
}

func uniqueDirs(files []string) []string {
	seen := make(map[string]struct{}, len(files))
	var dirs []string
	for _, file := range files {
		dir := filepath.Clean(filepath.Dir(file))
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}
