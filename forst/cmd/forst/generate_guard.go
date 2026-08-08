package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

// warnMissingLifecycleScript warns when ephemeral mode has no package.json
// script that runs `forst generate`. Never edits package.json.
func warnMissingLifecycleScript(boundaryRoot string, genCfg ftconfig.GenerateConfig, log *logrus.Logger) {
	if !genCfg.IsEphemeral(boundaryRoot) {
		return
	}

	if has, err := anyAncestorRunsForstGenerate(boundaryRoot); err == nil && has {
		return
	}

	pkgName := genCfg.PackageName
	if pkgName == "" {
		pkgName = ftconfig.DefaultPackageName
	}
	// Emit as separate lines so the pasteable JSON stays readable in logrus text output.
	log.Warn("no lifecycle script runs forst generate")
	log.Warn("  The generated client lives in .forst/client, which is gitignored, and it is")
	log.Warn("  linked from node_modules, which npm ci deletes. On a fresh checkout both are")
	log.Warn("  gone and imports of " + pkgName + " will fail with Cannot find module.")
	log.Warn("  Add this to package.json:")
	log.Warn(`    { "scripts": { "postinstall": "forst generate ." } }`)
}

func findNearestPackageJSON(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, "package.json")
		st, err := os.Stat(candidate)
		if err == nil && !st.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func anyAncestorRunsForstGenerate(start string) (bool, error) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, "package.json")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			has, checkErr := packageJSONRunsForstGenerate(candidate)
			if checkErr != nil {
				return false, checkErr
			}
			if has {
				return true, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false, nil
}

func packageJSONRunsForstGenerate(pkgPath string) (bool, error) {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return false, err
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, err
	}
	for _, cmd := range pkg.Scripts {
		if strings.Contains(cmd, "forst generate") {
			return true, nil
		}
	}
	return false, nil
}
