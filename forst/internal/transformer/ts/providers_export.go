package transformerts

import (
	"fmt"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

// OmittedFunction records a public function excluded from TypeScript emit
// because Providers(f) is non-empty (ADR-021).
type OmittedFunction struct {
	PackageName  string
	FunctionName string
	Reason       string
	// ExportDecl is the commented stub line body (e.g. "export function Login(): Promise<void>;").
	// Empty when the signature could not be transformed; emit falls back to a name-only stub.
	ExportDecl string
}

// ShouldEmitFunctionToTypeScript reports whether fn should appear in TS/sidecar artifacts (ADR-021).
// Public functions with outstanding Providers are omitted when typecheck is relaxed; strict generate errors instead.
func ShouldEmitFunctionToTypeScript(fn ast.FunctionNode, tc *typechecker.TypeChecker) bool {
	_, ok := ProviderOmissionReason(fn, tc)
	return ok
}

// ProviderOmissionReason returns a human-readable reason when fn must not appear in
// TypeScript emit. ok is true when the function should be emitted.
func ProviderOmissionReason(fn ast.FunctionNode, tc *typechecker.TypeChecker) (reason string, ok bool) {
	if fn.Receiver != nil || !ast.IsPublicExportIdent(fn.Ident.ID) {
		return "", false
	}
	if tc == nil {
		return "", true
	}
	slots := tc.FunctionProviders[fn.Ident.ID]
	if len(slots) == 0 {
		return "", true
	}
	return formatUnsatisfiedProvidersReason(typechecker.ProviderRootIdentsFromSlots(slots)), false
}

// CollectOmittedFunctions lists public functions skipped because providers are unsatisfied.
func CollectOmittedFunctions(packageName string, nodes []ast.Node, tc *typechecker.TypeChecker) []OmittedFunction {
	if tc == nil {
		return nil
	}
	var out []OmittedFunction
	for _, node := range nodes {
		fn, isFn := node.(ast.FunctionNode)
		if !isFn {
			continue
		}
		reason, emit := ProviderOmissionReason(fn, tc)
		if emit || reason == "" {
			continue
		}
		out = append(out, OmittedFunction{
			PackageName:  packageName,
			FunctionName: string(fn.Ident.ID),
			Reason:       reason,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PackageName != out[j].PackageName {
			return out[i].PackageName < out[j].PackageName
		}
		return out[i].FunctionName < out[j].FunctionName
	})
	return out
}

func formatUnsatisfiedProvidersReason(roots []string) string {
	if len(roots) == 0 {
		return "unsatisfied providers"
	}
	quoted := make([]string, len(roots))
	for i, r := range roots {
		quoted[i] = fmt.Sprintf("%q", r)
	}
	joined := strings.Join(quoted, ", ")
	if len(roots) == 1 {
		return "provider " + joined + " not satisfied"
	}
	return "providers " + joined + " not satisfied"
}
