package transformergo

import (
	"strconv"

	"forst/internal/typechecker"
	goast "go/ast"
	"go/token"
)

const forstBridgeManifestVarName = "forstBridgeManifestJSON"

// EmitNeedsBridgeRuntime reports whether generated Go should embed the Node manifest.
func EmitNeedsBridgeRuntime(tc *typechecker.TypeChecker) bool {
	if tc == nil {
		return false
	}
	return tc.NeedsBridgeRuntime()
}

// AppendNodeManifestDecl appends `var forstBridgeManifestJSON string = ...` to output when manifestJSON is non-empty.
func AppendNodeManifestDecl(output *TransformerOutput, manifestJSON string) {
	if output == nil || manifestJSON == "" {
		return
	}
	if output.HasValueDecl(forstBridgeManifestVarName) {
		return
	}
	output.AddValueDecl(&goast.GenDecl{
		Tok: token.VAR,
		Specs: []goast.Spec{
			&goast.ValueSpec{
				Names: []*goast.Ident{goast.NewIdent(forstBridgeManifestVarName)},
				Type:  goast.NewIdent("string"),
				Values: []goast.Expr{
					&goast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(manifestJSON),
					},
				},
			},
		},
	})
}

// AppendNodeManifestIfNeeded embeds the manifest in the node runtime companion output.
func (t *Transformer) AppendNodeManifestIfNeeded() {
	if t == nil || t.TypeChecker == nil || !EmitNeedsBridgeRuntime(t.TypeChecker) {
		return
	}
	t.appendNodeManifestToRuntime()
}
