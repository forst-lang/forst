package goload

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const forstModulePath = "forst"

// ForstModuleLink describes how a user go.mod depends on the forst runtime module.
type ForstModuleLink struct {
	RequireVersion string // e.g. "v0.0.37"
	ReplaceDir     string // absolute path from replace directive
}

// ForstModuleLinkFromGoMod reads require/replace forst from go.mod in moduleRoot.
func ForstModuleLinkFromGoMod(moduleRoot string) (ForstModuleLink, bool) {
	if moduleRoot == "" {
		return ForstModuleLink{}, false
	}
	path := filepath.Join(moduleRoot, "go.mod")
	f, err := os.Open(path)
	if err != nil {
		return ForstModuleLink{}, false
	}
	defer func() { _ = f.Close() }()

	var link ForstModuleLink
	inRequire := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire {
			if line == ")" {
				inRequire = false
				continue
			}
			if ver := parseRequireForst(line); ver != "" {
				link.RequireVersion = ver
			}
			continue
		}
		if ver := parseRequireForst(line); ver != "" {
			link.RequireVersion = ver
			continue
		}
		if replaceDir := parseReplaceForst(line, GoModReplaceBase(moduleRoot)); replaceDir != "" {
			link.ReplaceDir = replaceDir
		}
	}
	if link.ReplaceDir != "" || link.RequireVersion != "" {
		return link, true
	}
	return ForstModuleLink{}, false
}

func parseRequireForst(line string) string {
	fields := strings.Fields(line)
	switch {
	case len(fields) >= 2 && fields[0] == forstModulePath:
		return fields[1]
	case len(fields) >= 3 && fields[0] == "require" && fields[1] == forstModulePath:
		return fields[2]
	default:
		return ""
	}
}

func parseReplaceForst(line, moduleRoot string) string {
	if !strings.HasPrefix(line, "replace ") {
		return ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "replace "))
	before, after, ok := strings.Cut(rest, "=>")
	if !ok {
		return ""
	}
	mod := strings.Fields(strings.TrimSpace(before))
	if len(mod) == 0 || mod[0] != forstModulePath {
		return ""
	}
	target := strings.TrimSpace(after)
	if target == "" {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(moduleRoot, target)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return ""
	}
	return filepath.Clean(abs)
}
