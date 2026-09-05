package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// builtinsNotPermittedInDeferGoOperand lists predeclared builtins that Go disallows as the sole
// call in a defer/go statement — same set as “not permitted in statement context” (Go spec:
// Expression statements). Receive operations are already excluded because defer/go require a call.
//
// See: https://go.dev/ref/spec#Expression_statements
var builtinsNotPermittedInDeferGoOperand = map[string]struct{}{
	"append":  {},
	"cap":     {},
	"complex": {},
	"imag":    {},
	"len":     {},
	"make":    {},
	"new":     {},
	"real":    {},
	// unsafe package (qualified name as in Forst call identifiers)
	"unsafe.Add":        {},
	"unsafe.Alignof":    {},
	"unsafe.Offsetof":   {},
	"unsafe.Sizeof":     {},
	"unsafe.Slice":      {},
	"unsafe.SliceData":  {},
	"unsafe.String":     {},
	"unsafe.StringData": {},
}

// validateDeferGoBuiltinRestriction returns an error if the call targets a builtin that Go
// forbids as a defer/go operand.
func validateDeferGoBuiltinRestriction(keyword string, call ast.FunctionCallNode) error {
	name := string(call.Function.ID)
	if _, bad := builtinsNotPermittedInDeferGoOperand[name]; bad {
		sp := firstSetSpan(call.CallSpan, call.Function.Span)
		return reportf(sp, "defer-go-builtin",
			fmt.Sprintf("%s cannot call built-in `%s`", keyword, name),
			fmt.Sprintf("Go disallows `%s` as the sole operand of `%s` (same rule as expression statements).", name, keyword),
			"wrap the call in a function or use a different statement form")
	}
	return nil
}
