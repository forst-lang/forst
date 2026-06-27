package typechecker

import (
	"forst/internal/ast"
)

func splitQualifiedCallee(s string) (importLocal, fnName string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i], s[i+1:], true
		}
	}
	return "", s, false
}

// providerSlotsForCallee returns merged Provider slots for intra- or cross-package callees.
func (tc *TypeChecker) providerSlotsForCallee(callee ast.Identifier) []ProviderSlot {
	calleeStr := string(callee)
	if importLocal, fnName, ok := splitQualifiedCallee(calleeStr); ok {
		if tc.moduleResult != nil {
			if path, pathOK := tc.ImportPathForLocal(importLocal); pathOK && path != "" {
				if forstPkg := tc.importPathToForstPkgMap()[path]; forstPkg != "" {
					if sibling := tc.moduleResult.ForstPackageTypeChecker(forstPkg); sibling != nil {
						return sibling.FunctionProviders[ast.Identifier(fnName)]
					}
				}
			}
		}
		return nil
	}
	if slots := tc.FunctionProviders[callee]; len(slots) > 0 {
		return slots
	}
	if tc.providers != nil {
		return tc.providers.Slots[callee]
	}
	return nil
}

// RevalidateUnusedWiringKeysAfterModuleMerge re-runs unused-key analysis using merged FunctionProviders.
func (tc *TypeChecker) RevalidateUnusedWiringKeysAfterModuleMerge() {
	if tc.providers == nil {
		return
	}
	filtered := make([]Diagnostic, 0, len(tc.Warnings))
	for _, w := range tc.Warnings {
		if w.Code == "providers-unused-key" {
			continue
		}
		filtered = append(filtered, w)
	}
	tc.Warnings = filtered
	for _, check := range tc.providers.PendingWith {
		tc.checkUnusedWiringKeys(check)
	}
}

// RevalidateDeferredWiringKeysAfterModuleMerge re-checks wiring keys using merged module KnownRoots.
func (tc *TypeChecker) RevalidateDeferredWiringKeysAfterModuleMerge() error {
	if tc.providers == nil || !tc.providers.DeferWiringRootCheck {
		return nil
	}
	for _, check := range tc.providers.PendingWith {
		if err := tc.revalidateWiringKeysInWith(check.with); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TypeChecker) revalidateWiringKeysInWith(with ast.WithNode) error {
	eng := tc.providersEngine()
	switch w := with.Wiring.(type) {
	case ast.ShapeNode:
		for fieldName, field := range w.Fields {
			if field.IsMethod {
				continue
			}
			span := shapeFieldSpan(w, fieldName, field)
			if err := tc.validateWiringKey(fieldName, span); err != nil {
				return err
			}
			if _, ok := eng.KnownRoots[fieldName]; !ok {
				return diagnosticf(span, "providers-unknown-key", "unknown wiring key %q", fieldName)
			}
		}
	}
	return nil
}

// MergeModuleKnownRoots copies provider contract roots from all packages into each typechecker.
func MergeModuleKnownRoots(perPackage map[string]*TypeChecker) {
	merged := make(map[string]ast.TypeNode)
	for _, tc := range perPackage {
		if tc.providers == nil {
			continue
		}
		for k, v := range tc.providers.KnownRoots {
			merged[k] = v
		}
	}
	for _, tc := range perPackage {
		if tc.providers == nil {
			continue
		}
		for k, v := range merged {
			tc.providers.KnownRoots[k] = v
		}
	}
}
