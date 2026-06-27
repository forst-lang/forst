package typechecker

import (
	"strings"

	"forst/internal/ast"
	"forst/internal/providersgraph"
)

func (tc *TypeChecker) checkCallProvidersSatisfied(caller, callee ast.Identifier, scope map[string]ast.TypeNode, span ast.SourceSpan) error {
	slots := tc.FunctionProviders[callee]
	var missing []string
	for _, slot := range slots {
		if !tc.scopeSatisfiesSlot(slot, scope) {
			missing = append(missing, string(slot.RootIdent))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return diagnosticfRelated(span, "providers-unsatisfied", tc.providersObligationRelated(caller, callee),
		"%s requires %s; not supplied\n  required by: %s%s", callee, strings.Join(missing, ", "),
		tc.buildCallSiteObligationChain(caller, callee, missing), providersFixItHint(missing))
}

func (tc *TypeChecker) providersObligationRelated(caller, callee ast.Identifier) []RelatedDiagnostic {
	var related []RelatedDiagnostic
	if sig, ok := tc.Functions[caller]; ok && sig.Ident.Span.IsSet() {
		related = append(related, RelatedDiagnostic{
			Msg:  "caller " + string(caller),
			Span: sig.Ident.Span,
		})
	}
	if sig, ok := tc.Functions[callee]; ok && sig.Ident.Span.IsSet() {
		related = append(related, RelatedDiagnostic{
			Msg:  string(callee) + " declares Providers requirements",
			Span: sig.Ident.Span,
		})
	}
	return related
}

func (tc *TypeChecker) recordFunctionCall(callee ast.Identifier, span ast.SourceSpan) {
	fn := tc.currentFunctionIdent()
	if fn == "" {
		return
	}
	eng := tc.providersEngine()
	eng.CallEdges = append(eng.CallEdges, providersgraph.CallEdge{
		CallerFn: fn,
		CalleeFn: callee,
		Scope:    tc.currentMergedScope(),
		Span:     span,
	})
}
