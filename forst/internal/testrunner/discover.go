package testrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const generatedTestGoName = "z_forst_gen_test.go"

// PackageUnderTest is one Forst package directory containing *_test.ft files.
type PackageUnderTest struct {
	Dir       string   // absolute package directory
	RelPath   string   // slash-separated path relative to module root (e.g. "auth")
	FtPaths   []string // all .ft files in Dir, sorted
	TestPaths []string // *_test.ft paths in Dir
}

// IsTestForstFile reports whether path is a Forst test source file (*_test.ft).
func IsTestForstFile(path string) bool {
	return strings.HasSuffix(filepath.Base(path), "_test.ft")
}

// DiscoverPackages finds package directories with *_test.ft under moduleRoot.
// When paths is non-empty, each path is resolved under moduleRoot and must live in a package dir with tests.
func DiscoverPackages(moduleRoot string, paths []string) ([]PackageUnderTest, error) {
	moduleRoot = filepath.Clean(moduleRoot)
	if paths == nil {
		paths = []string{}
	}
	var testDirs map[string]struct{}
	if len(paths) == 0 {
		testDirs = make(map[string]struct{})
		err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && path != moduleRoot {
					return filepath.SkipDir
				}
				return nil
			}
			if !IsTestForstFile(path) {
				return nil
			}
			testDirs[filepath.Dir(path)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("discover test files: %w", err)
		}
	} else {
		testDirs = make(map[string]struct{})
		for _, p := range paths {
			abs := p
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(moduleRoot, p)
			}
			abs, err := currentFilepathAbs()(abs)
			if err != nil {
				return nil, err
			}
			dir := abs
			if strings.HasSuffix(strings.ToLower(abs), ".ft") {
				dir = filepath.Dir(abs)
			}
			found := false
			entries, err := currentReadDirFn()(dir)
			if err != nil {
				return nil, fmt.Errorf("read package dir %s: %w", dir, err)
			}
			for _, e := range entries {
				if !e.IsDir() && IsTestForstFile(filepath.Join(dir, e.Name())) {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("no *_test.ft files in %s", dir)
			}
			testDirs[dir] = struct{}{}
		}
	}
	if len(testDirs) == 0 {
		return nil, fmt.Errorf("no Forst test files (*_test.ft) found under %s", moduleRoot)
	}

	var out []PackageUnderTest
	for dir := range testDirs {
		rel, err := currentFilepathRelDiscover()(moduleRoot, dir)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		entries, err := currentReadDirFn()(dir)
		if err != nil {
			return nil, err
		}
		var ftPaths, testPaths []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".ft") {
				continue
			}
			full := filepath.Join(dir, name)
			ftPaths = append(ftPaths, full)
			if IsTestForstFile(full) {
				testPaths = append(testPaths, full)
			}
		}
		if len(testPaths) == 0 {
			continue
		}
		sort.Strings(ftPaths)
		sort.Strings(testPaths)
		out = append(out, PackageUnderTest{
			Dir:       dir,
			RelPath:   rel,
			FtPaths:   ftPaths,
			TestPaths: testPaths,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out, nil
}
