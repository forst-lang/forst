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
		return fmt.Errorf("%s: built-in %q cannot be used as operand (Go: same restriction as expression statements for this builtin)", keyword, name)
	}
	return nil
}
