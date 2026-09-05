package transformergo_test

import (
	"slices"
	"testing"

	"forst/internal/semantic"
	transformergo "forst/internal/transformer/go"
	"forst/internal/typechecker"
)

func TestBuiltinConstraintNames_listsStayInSync(t *testing.T) {
	t.Parallel()
	names := slices.Clone(transformergo.BuiltinConstraintNames)
	slices.Sort(names)

	for _, name := range names {
		if !typechecker.IsBuiltinAssertionConstraintName(name) {
			t.Errorf("typechecker missing builtin %q", name)
		}
		if !semantic.IsKnownBuiltinConstraintName(name) {
			t.Errorf("semantic missing builtin %q", name)
		}
	}
}
