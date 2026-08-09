package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	transformerts "forst/internal/transformer/ts"
)

// effectPeerFloor is the minimum resolved Effect version for generate.effect.
const effectPeerFloor = "3.17.0"

// requireEffectRuntime checks the resolved effect package under node_modules.
// It reads the installed version, not the declared range in package.json.
func requireEffectRuntime(boundaryRoot string) error {
	pkgPath, version, err := resolveEffectPackage(boundaryRoot)
	if err != nil {
		return fmt.Errorf(
			"generate: generate.effect is true but effect %s is required\n"+
				"  found:    none\n"+
				"  required: %s for Layer.mock in the generated testing module\n"+
				"  upgrade:  npm install effect@latest",
			transformerts.EffectPeerDependencyRange,
			transformerts.EffectPeerDependencyRange,
		)
	}
	if !effectVersionAtLeast(version, effectPeerFloor) {
		rel := pkgPath
		if r, relErr := filepath.Rel(boundaryRoot, pkgPath); relErr == nil {
			rel = r
		}
		return fmt.Errorf(
			"generate: generate.effect is true but effect %s is required\n"+
				"  found:    effect@%s at %s\n"+
				"  required: %s for Layer.mock in the generated testing module\n"+
				"  upgrade:  npm install effect@latest",
			transformerts.EffectPeerDependencyRange,
			version,
			rel,
			transformerts.EffectPeerDependencyRange,
		)
	}
	return nil
}

func resolveEffectPackage(boundaryRoot string) (pkgJSONPath string, version string, err error) {
	dir := filepath.Clean(boundaryRoot)
	for {
		candidate := filepath.Join(dir, "node_modules", "effect", "package.json")
		data, readErr := generateIO.ReadFile(candidate)
		if readErr == nil {
			var meta struct {
				Version string `json:"version"`
			}
			if jsonErr := json.Unmarshal(data, &meta); jsonErr != nil {
				return "", "", jsonErr
			}
			if meta.Version == "" {
				return "", "", fmt.Errorf("effect package.json missing version")
			}
			return candidate, meta.Version, nil
		}
		if !os.IsNotExist(readErr) {
			return "", "", readErr
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", fmt.Errorf("effect not found")
}

func effectVersionAtLeast(version, floor string) bool {
	vParts, ok := parseStableSemver(version)
	if !ok {
		return false
	}
	fParts, ok := parseStableSemver(floor)
	if !ok {
		return false
	}
	for i := 0; i < 3; i++ {
		if vParts[i] > fParts[i] {
			return true
		}
		if vParts[i] < fParts[i] {
			return false
		}
	}
	return true
}

// parseStableSemver parses a strict three-component semver without prerelease or build metadata.
func parseStableSemver(v string) ([3]int, bool) {
	v = strings.TrimPrefix(v, "v")
	if strings.ContainsAny(v, "-+") {
		return [3]int{}, false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || parts[i] == "" || n < 0 {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
