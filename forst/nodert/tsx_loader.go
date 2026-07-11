package nodert

import (
	"fmt"
	"os"
	"path/filepath"
)

const tsxLoaderRel = "node_modules/tsx/dist/loader.mjs"

// ResolveTsxLoaderPath finds tsx's Node register hook (loader.mjs) by walking up from startDirs.
func ResolveTsxLoaderPath(startDirs ...string) (string, error) {
	seen := make(map[string]struct{})
	for _, start := range startDirs {
		if start == "" {
			continue
		}
		abs, err := filepath.Abs(start)
		if err != nil {
			continue
		}
		for dir := abs; ; dir = filepath.Dir(dir) {
			if _, ok := seen[dir]; ok {
				break
			}
			seen[dir] = struct{}{}
			candidate := filepath.Join(dir, tsxLoaderRel)
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return "", fmt.Errorf("node runtime: tsx loader not found (install tsx in the project or monorepo root)")
}
