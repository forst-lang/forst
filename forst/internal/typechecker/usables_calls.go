package typechecker

import (
	"strings"

	"forst/internal/ast"
)

func (tc *TypeChecker) checkCallUsablesSatisfied(caller, callee ast.Identifier, ambient map[string]ast.TypeNode, span ast.SourceSpan) error {
	slots := tc.FunctionUsables[callee]
	var missing []string
	for _, slot := range slots {
		if !tc.ambientSatisfiesSlot(slot, ambient) {
			missing = append(missing, string(slot.RootIdent))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return diagnosticfRelated(span, "usables-unsatisfied", tc.usablesObligationRelated(caller, callee),
		"%s requires %s; not supplied\n  required by: %s", callee, strings.Join(missing, ", "), CallSiteObligationChain(caller, callee, tc.FunctionUsables[callee]))
}

func (tc *TypeChecker) usablesObligationRelated(caller, callee ast.Identifier) []RelatedDiagnostic {
	var related []RelatedDiagnostic
	if sig, ok := tc.Functions[caller]; ok && sig.Ident.Span.IsSet() {
		related = append(related, RelatedDiagnostic{
			Msg:  "caller " + string(caller),
			Span: sig.Ident.Span,
		})
	}
	if sig, ok := tc.Functions[callee]; ok && sig.Ident.Span.IsSet() {
		related = append(related, RelatedDiagnostic{
			Msg:  string(callee) + " declares Usables requirements",
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
	rec := callSiteRecord{
		Callee:      callee,
		AmbientKeys: tc.currentMergedAmbient(),
		Span:        span,
	}
	tc.functionCallSites[fn] = append(tc.functionCallSites[fn], rec)
}
