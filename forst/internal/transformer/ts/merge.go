package transformerts

import (
	"fmt"
	"regexp"
	"strings"
)

// exportedTypeNamePattern extracts the identifier from an export interface/type/enum/class block.
var exportedTypeNamePattern = regexp.MustCompile(`(?m)^export\s+(?:interface|type|enum|class)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// MergeTypeScriptOutputs combines per-file TypeScript outputs into one declaration bundle suitable
// for a shared types.d.ts. Type blocks are keyed by exported name: identical bodies merge once,
// conflicting bodies fail naming both packages. Function names may repeat across packages (subpaths
// isolate them); identical signatures within the merged list are still deduped for StreamingResult.
func MergeTypeScriptOutputs(outputs []*TypeScriptOutput) (*TypeScriptOutput, error) {
	if len(outputs) == 0 {
		return &TypeScriptOutput{}, nil
	}

	var pkg string
	for _, o := range outputs {
		if o != nil && o.PackageName != "" {
			pkg = o.PackageName
			break
		}
	}

	type typeEntry struct {
		body string
		pkg  string
	}
	byTypeName := make(map[string]typeEntry)
	var mergedTypes []string
	for _, o := range outputs {
		if o == nil {
			continue
		}
		srcPkg := o.PackageName
		if srcPkg == "" {
			srcPkg = "(unknown)"
		}
		for _, t := range o.Types {
			name := exportedTypeName(t)
			if name == "" {
				// Keep unparseable blocks by full text so we never drop emit silently.
				if prev, ok := byTypeName[t]; ok {
					if prev.body == t {
						continue
					}
					return nil, conflictingTypeError(t, prev.pkg, srcPkg)
				}
				byTypeName[t] = typeEntry{body: t, pkg: srcPkg}
				mergedTypes = append(mergedTypes, t)
				continue
			}
			if prev, ok := byTypeName[name]; ok {
				if prev.body == t {
					continue
				}
				return nil, conflictingTypeError(name, prev.pkg, srcPkg)
			}
			byTypeName[name] = typeEntry{body: t, pkg: srcPkg}
			mergedTypes = append(mergedTypes, t)
		}
	}

	// Functions are no longer emitted into types.d.ts. Collect them without failing on
	// cross-package name collisions (subpaths isolate invoke helpers). Keep StreamingRowType
	// markers so shapes-only types.d.ts can still emit StreamingResult when needed.
	byName := make(map[string]FunctionSignature)
	var mergedFuncs []FunctionSignature
	for _, o := range outputs {
		if o == nil {
			continue
		}
		for _, f := range o.Functions {
			if prev, ok := byName[f.Name]; ok {
				if functionSignaturesEqual(prev, f) {
					continue
				}
				if f.StreamingRowType != "" && prev.StreamingRowType == "" {
					prev.StreamingRowType = f.StreamingRowType
					byName[f.Name] = prev
					for i := range mergedFuncs {
						if mergedFuncs[i].Name == f.Name {
							mergedFuncs[i].StreamingRowType = f.StreamingRowType
							break
						}
					}
				}
				continue
			}
			byName[f.Name] = f
			mergedFuncs = append(mergedFuncs, f)
		}
	}

	seenExport := make(map[string]struct{})
	var mergedExports []string
	for _, o := range outputs {
		if o == nil {
			continue
		}
		for _, n := range o.ExportedTypeNames {
			if n == "" {
				continue
			}
			if _, ok := seenExport[n]; ok {
				continue
			}
			seenExport[n] = struct{}{}
			mergedExports = append(mergedExports, n)
		}
	}
	mergedExports = sortDedupeStrings(mergedExports)

	var domainParts [][]ErrorClass
	for _, o := range outputs {
		if o == nil || len(o.DomainErrors) == 0 {
			continue
		}
		domainParts = append(domainParts, o.DomainErrors)
	}

	return &TypeScriptOutput{
		PackageName:       pkg,
		Types:             mergedTypes,
		ExportedTypeNames: mergedExports,
		Functions:         mergedFuncs,
		DomainErrors:      MergeDomainErrors(domainParts...),
	}, nil
}

func conflictingTypeError(typeName, pkgA, pkgB string) error {
	return fmt.Errorf(
		"generate: conflicting type %q defined in packages %q and %q\n"+
			"  both are re-exported from the package root, so the names collide\n"+
			"  rename one of the Forst types",
		typeName, pkgA, pkgB,
	)
}

func exportedTypeName(typeDecl string) string {
	m := exportedTypeNamePattern.FindStringSubmatch(strings.TrimSpace(typeDecl))
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func functionSignaturesEqual(a, b FunctionSignature) bool {
	if a.Name != b.Name || a.ReturnType != b.ReturnType {
		return false
	}
	if a.StreamingRowType != b.StreamingRowType {
		return false
	}
	if len(a.Parameters) != len(b.Parameters) {
		return false
	}
	for i := range a.Parameters {
		if a.Parameters[i].Name != b.Parameters[i].Name || a.Parameters[i].Type != b.Parameters[i].Type {
			return false
		}
	}
	return true
}
