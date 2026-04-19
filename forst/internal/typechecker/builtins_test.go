package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestCheckBuiltinMethod_implicitReceiver_returnsOpaque(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	for _, method := range []string{"Date", "GetFoo", "HeaderValues", "Parse"} {
		t.Run(method, func(t *testing.T) {
			got, err := tc.CheckBuiltinMethod(ast.TypeNode{Ident: ast.TypeImplicit}, method, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Ident != ast.TypeImplicit {
				t.Fatalf("want single implicit, got %v", got)
			}
		})
	}
}
