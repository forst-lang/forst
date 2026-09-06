package forstdep

import (
	"io"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
)

// DiscoveredPackage is a Forst package found outside the consumer module walk
// via the Go import graph (replace, vendor, or module cache).
type DiscoveredPackage struct {
	Loc      goload.PackageLoc
	ForstPkg string
	Files    []string
	Nodes    []ast.Node
}

// Discover walks the import graph from seedImportPaths and returns Forst packages
// whose directories contain .ft files. knownImportPaths are skipped (already local).
func Discover(log *logrus.Logger, moduleRoot string, knownImportPaths map[string]string, seedImportPaths []string) ([]DiscoveredPackage, error) {
	if moduleRoot == "" {
		return nil, nil
	}
	moduleRoot = goload.FindModuleRoot(moduleRoot)
	if moduleRoot == "" {
		return nil, nil
	}
	if log == nil {
		log = logrus.New()
		log.SetOutput(io.Discard)
	}

	seen := make(map[string]struct{})
	for path := range knownImportPaths {
		seen[path] = struct{}{}
	}

	queue := uniquePaths(seedImportPaths)
	var out []DiscoveredPackage

	for len(queue) > 0 {
		batch := queue
		queue = nil
		pending := make([]string, 0, len(batch))
		for _, path := range batch {
			if _, ok := seen[path]; ok {
				continue
			}
			pending = append(pending, path)
		}
		if len(pending) == 0 {
			continue
		}
		locs, err := goload.LocatePackageDirs(moduleRoot, pending)
		if err != nil {
			log.WithError(err).Debug("forstdep: LocatePackageDirs failed")
			continue
		}
		for _, path := range pending {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			loc, ok := locs[path]
			if !ok || loc.Dir == "" {
				continue
			}
			ftFiles, err := ForstFilesInDir(loc.Dir)
			if err != nil {
				return out, err
			}
			if len(ftFiles) == 0 {
				continue
			}
			sort.Strings(ftFiles)
			parsed := forstpkg.ParseFilesLenientParallel(log, ftFiles)
			var astLists [][]ast.Node
			var files []string
			byPkg := make(map[string][]string)
			for _, f := range ftFiles {
				nodes := parsed[f]
				pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
				byPkg[pkg] = append(byPkg[pkg], f)
				astLists = append(astLists, nodes)
				files = append(files, f)
			}
			if err := forstpkg.ValidateGoPackageLayout(byPkg); err != nil {
				return out, err
			}
			pkgName := ""
			for name := range byPkg {
				pkgName = name
				break
			}
			merged := forstpkg.MergePackageASTs(astLists)
			dp := DiscoveredPackage{
				Loc:      loc,
				ForstPkg: pkgName,
				Files:    files,
				Nodes:    merged,
			}
			out = append(out, dp)
			for _, next := range ImportPathsFromNodes(merged) {
				if _, ok := seen[next]; ok {
					continue
				}
				queue = append(queue, next)
			}
			log.WithFields(logrus.Fields{
				"importPath": path,
				"dir":        loc.Dir,
				"forstPkg":   pkgName,
				"files":      len(files),
			}).Debug("forstdep: discovered external Forst package")
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Loc.ImportPath < out[j].Loc.ImportPath
	})
	return out, nil
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
