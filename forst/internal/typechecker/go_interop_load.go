package typechecker

// recordGoPackagesLoadFailure stores a go/packages load error for each import path in paths.
func (tc *TypeChecker) recordGoPackagesLoadFailure(paths []string, err error) {
	if tc == nil || err == nil || len(paths) == 0 {
		return
	}
	if tc.goImportLoadErrors == nil {
		tc.goImportLoadErrors = make(map[string]error)
	}
	for _, p := range paths {
		if p != "" {
			tc.goImportLoadErrors[p] = err
		}
	}
}

func (tc *TypeChecker) goImportLoadErrorForPath(path string) error {
	if tc == nil || tc.goImportLoadErrors == nil || path == "" {
		return nil
	}
	return tc.goImportLoadErrors[path]
}

func (tc *TypeChecker) goImportLoadErrorForLocal(local string) error {
	if tc == nil || local == "" {
		return nil
	}
	path, ok := tc.ImportPathForLocal(local)
	if !ok {
		return nil
	}
	return tc.goImportLoadErrorForPath(path)
}
