package transformergo

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestIsTypeGuardCompatible_subjectMatchesOrNot(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	tg := &ast.TypeGuardNode{
		Ident: "Strong",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("p")},
			Type:  ast.NewBuiltinType(ast.TypeString),
		},
		Body: []ast.Node{},
	}
	strVar := ast.NewBuiltinType(ast.TypeString)
	intVar := ast.NewBuiltinType(ast.TypeInt)

	if !tr.isTypeGuardCompatible(strVar, tg) {
		t.Fatal("String subject should accept String var")
	}
	if tr.isTypeGuardCompatible(intVar, tg) {
		t.Fatal("String guard should reject Int var")
	}
}
