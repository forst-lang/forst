package discovery

import "forst/internal/forstpkg"

// BuildForstPackageImportPaths maps Go import paths to Forst package names.
// Deprecated: use forstpkg.BuildForstPackageImportPaths.
func BuildForstPackageImportPaths(moduleRoot, modulePath string, forstPkgToFiles map[string][]string) map[string]string {
	return forstpkg.BuildForstPackageImportPaths(moduleRoot, modulePath, forstPkgToFiles)
}
