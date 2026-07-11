package typechecker

import (
	"testing"
)

func TestBuiltinFunctions_noIntConversionBuiltin(t *testing.T) {
	t.Parallel()
	if _, ok := BuiltinFunctions["Int"]; ok {
		t.Fatal("Int conversion builtin should be removed from BuiltinFunctions")
	}
}
