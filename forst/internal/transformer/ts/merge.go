package transformerts

import (
	"fmt"
)

// MergeTypeScriptOutputs combines per-file TypeScript outputs into one declaration bundle suitable
// for a shared types.d.ts. Type blocks that are byte-identical are deduped. Function declarations
// with the same name must have the same signature; otherwise MergeTypeScriptOutputs returns an error.
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

	seenType := make(map[string]struct{})
	var mergedTypes []string
	for _, o := range outputs {
		if o == nil {
			continue
		}
		for _, t := range o.Types {
			if _, ok := seenType[t]; ok {
				continue
			}
			seenType[t] = struct{}{}
			mergedTypes = append(mergedTypes, t)
		}
	}

	byName := make(map[string]FunctionSignature)
	var mergedFuncs []FunctionSignature
	for _, o := range outputs {
		if o == nil {
			continue
		}
		for _, f := range o.Functions {
			if prev, ok := byName[f.Name]; ok {
				if !functionSignaturesEqual(prev, f) {
					return nil, fmt.Errorf("duplicate exported function %q with conflicting TypeScript signatures", f.Name)
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

	return &TypeScriptOutput{
		PackageName:       pkg,
		Types:             mergedTypes,
		ExportedTypeNames: mergedExports,
		Functions:           mergedFuncs,
	}, nil
}

func functionSignaturesEqual(a, b FunctionSignature) bool {
	if a.Name != b.Name || a.ReturnType != b.ReturnType {
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
