package transformergo

import (
	"strconv"

	"forst/internal/typechecker"
	goast "go/ast"
	"go/token"
)

const forstNodeManifestVarName = "forstNodeManifestJSON"

// EmitNeedsNodeRuntime reports whether generated Go should embed the Node manifest.
func EmitNeedsNodeRuntime(tc *typechecker.TypeChecker) bool {
	if tc == nil {
		return false
	}
	return tc.NeedsNodeRuntime()
}

// AppendNodeManifestDecl appends `var forstNodeManifestJSON string = ...` to output when manifestJSON is non-empty.
func AppendNodeManifestDecl(output *TransformerOutput, manifestJSON string) {
	if output == nil || manifestJSON == "" {
		return
	}
	if output.HasValueDecl(forstNodeManifestVarName) {
		return
	}
	output.AddValueDecl(&goast.GenDecl{
		Tok: token.VAR,
		Specs: []goast.Spec{
			&goast.ValueSpec{
				Names: []*goast.Ident{goast.NewIdent(forstNodeManifestVarName)},
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
	if t == nil || t.TypeChecker == nil || !EmitNeedsNodeRuntime(t.TypeChecker) {
		return
	}
	t.appendNodeManifestToRuntime()
}
