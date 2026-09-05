package typechecker

import (
	"strings"

	"forst/internal/ast"
)

// AccessPattern is a summary write/escape rooted at a parameter (or Reachable).
type AccessPattern struct {
	ParamIndex int // 0-based parameter; -1 = unknown/global
	Steps      []AccessStep
	Reachable  bool // whole argument / unknown interior
}

// AliasStore records sequential alias storage (filled further in 4d/4g).
type AliasStore struct {
	Dest AccessPattern
	Src  AccessPattern
}

// ReturnAlias records that a result may alias a parameter path.
type ReturnAlias struct {
	ResultIndex int
	Of          AccessPattern
}

// FunctionSummary is the inferred effect summary for a Forst function (phase 4c).
type FunctionSummary struct {
	Writes        []AccessPattern
	Stores        []AliasStore
	ReturnAliases []ReturnAlias
	Escapes       []AccessPattern
	SpawnsWith    []AccessPattern
}

// ensureSummaries lazily allocates the function-summary map.
func (tc *TypeChecker) ensureSummaries() {
	if tc.functionSummaries == nil {
		tc.functionSummaries = make(map[ast.Identifier]*FunctionSummary)
	}
}

// SummaryFor returns the inferred summary for fn, if any.
func (tc *TypeChecker) SummaryFor(fn ast.Identifier) *FunctionSummary {
	if tc == nil || tc.functionSummaries == nil {
		return nil
	}
	return tc.functionSummaries[fn]
}

// setFunctionSummary stores the summary for fn (overwriting any prior entry).
func (tc *TypeChecker) setFunctionSummary(fn ast.Identifier, sum *FunctionSummary) {
	tc.ensureSummaries()
	tc.functionSummaries[fn] = sum
}

// recordParamWriteDuringInfer notes a write through a parameter for the function being inferred.
// recordParamWriteDuringInfer accumulates a parameter-rooted write into the current function summary.
func (tc *TypeChecker) recordParamWriteDuringInfer(writePath *AccessPath) {
	if tc == nil || writePath == nil || tc.currentInferFn == "" {
		return
	}
	paramIdx, steps, ok := tc.paramIndexForPath(writePath)
	if !ok {
		return
	}
	tc.ensureSummaries()
	sum := tc.functionSummaries[tc.currentInferFn]
	if sum == nil {
		sum = &FunctionSummary{}
		tc.functionSummaries[tc.currentInferFn] = sum
	}
	sum.Writes = append(sum.Writes, AccessPattern{ParamIndex: paramIdx, Steps: steps})
}

// paramIndexForPath maps a write path's root to a parameter index of the function being inferred.
// paramIndexForPath maps an AccessPath root back to a function parameter index and suffix steps.
func (tc *TypeChecker) paramIndexForPath(p *AccessPath) (int, []AccessStep, bool) {
	if p == nil || tc.currentInferParams == nil {
		return -1, nil, false
	}
	for i, id := range tc.currentInferParams {
		symID, ok := tc.CurrentScope().LookupSymbolID(id)
		if !ok {
			// Walk parents for param binding.
			for s := tc.CurrentScope(); s != nil; s = s.Parent {
				if sid, ok := s.LookupSymbolID(id); ok {
					symID = sid
					ok = true
					break
				}
			}
		}
		if !ok {
			continue
		}
		if p.Root == symID {
			return i, p.CloneSteps(), true
		}
	}
	return -1, nil, false
}

// applyCallSummaryInvalidation substitutes summary writes onto call arguments and drops facts.
// applyCallSummaryInvalidation substitutes call-site args into fn's Writes and drops overlapping facts.
func (tc *TypeChecker) applyCallSummaryInvalidation(fn ast.Identifier, e ast.FunctionCallNode) {
	sum := tc.SummaryFor(fn)
	span := e.CallSpan
	if !span.IsSet() {
		span = e.Function.Span
	}
	if sum == nil {
		// Unknown Forst callee: conservative drop of facts on all args.
		for _, arg := range e.Arguments {
			if path := tc.accessPathForExpr(arg); path != nil {
				root := tc.paths.Intern(AccessPath{Root: path.Root})
				tc.invalidateOverlappingFactsWithReason(root, span, dropByCall)
			}
		}
		return
	}
	for _, w := range sum.Writes {
		if w.ParamIndex < 0 || w.ParamIndex >= len(e.Arguments) {
			continue
		}
		argPath := tc.accessPathForExpr(e.Arguments[w.ParamIndex])
		if argPath == nil {
			continue
		}
		if w.Reachable {
			root := tc.paths.Intern(AccessPath{Root: argPath.Root})
			tc.invalidateOverlappingFactsWithReason(root, span, dropByCall)
			continue
		}
		write := tc.paths.Intern(AccessPath{
			Root:  argPath.Root,
			Steps: append(argPath.CloneSteps(), append([]AccessStep(nil), w.Steps...)...),
		})
		tc.invalidateOverlappingFactsWithReason(write, span, dropByCall)
	}
}

// finalizeFunctionSummaryReturns coarsely treats returning a parameter field as
// aliasing that parameter (v1: whole-root writes when the alias is later mutated
// are approximated by marking the param Reachable in Writes).
func (tc *TypeChecker) finalizeFunctionSummaryReturns(fn ast.FunctionNode) {
	sum := tc.SummaryFor(fn.Ident.ID)
	if sum == nil {
		return
	}
	for _, stmt := range collectReturnStatements(fn.Body) {
		for _, val := range stmt.Values {
			id := dottedIdentFromExpr(toExpr(val))
			if id == "" {
				continue
			}
			for i, param := range tc.currentInferParams {
				ps := string(param)
				if id == ps || strings.HasPrefix(id, ps+".") {
					sum.ReturnAliases = append(sum.ReturnAliases, ReturnAlias{
						ResultIndex: 0,
						Of:          AccessPattern{ParamIndex: i, Steps: relativeFieldSteps(ps, id)},
					})
					// Coarse: treat as reachable write potential on the param root.
					sum.Writes = append(sum.Writes, AccessPattern{ParamIndex: i, Reachable: true})
				}
			}
		}
	}
}

func toExpr(v ast.ExpressionNode) ast.ExpressionNode { return v }
