package typechecker

import (
	"fmt"
	goast "go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

var (
	builtinGoDocOnce sync.Once
	builtinGoDocPkg  *doc.Package
	builtinGoDocErr  error
)

// loadBuiltinGoDocPackage parses $GOROOT/src/builtin (Go's documented predeclared identifiers) and
// returns go/doc metadata. Safe to call repeatedly; parsing happens once.
func loadBuiltinGoDocPackage(log *logrus.Logger) (*doc.Package, error) {
	builtinGoDocOnce.Do(func() {
		ctx := build.Default
		dir := filepath.Join(ctx.GOROOT, "src", "builtin")
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			builtinGoDocErr = fmt.Errorf("builtin doc dir: %w", err)
			if log != nil {
				log.WithFields(logrus.Fields{"dir": dir, "err": builtinGoDocErr}).Debug("skip builtin go/doc")
			}
			return
		}
		fset := token.NewFileSet()
		filter := func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }
		pkgs, err := parser.ParseDir(fset, dir, filter, parser.ParseComments)
		if err != nil {
			builtinGoDocErr = err
			if log != nil {
				log.WithError(err).Debug("parse builtin package for doc")
			}
			return
		}
		astPkg, ok := pkgs["builtin"]
		if !ok || astPkg == nil {
			builtinGoDocErr = fmt.Errorf("no package builtin in %s", dir)
			return
		}
		builtinGoDocPkg = doc.New(astPkg, "builtin", doc.AllDecls)
	})
	if builtinGoDocErr != nil {
		return nil, builtinGoDocErr
	}
	return builtinGoDocPkg, nil
}

// builtinGoDocParagraph returns documentation text from the builtin package for a predeclared type
// name and method name. For interface types, per-method comments are used when present; otherwise
// the enclosing type's documentation is used (e.g. error + Error).
func builtinGoDocParagraph(pkg *doc.Package, goTypeName, methodName string) string {
	if pkg == nil {
		return ""
	}
	for _, t := range pkg.Types {
		if t.Name != goTypeName {
			continue
		}
		if s := builtinMethodOrTypeDocFromDecl(t, methodName); s != "" {
			return s
		}
	}
	return ""
}

func builtinMethodOrTypeDocFromDecl(t *doc.Type, methodName string) string {
	if t.Decl == nil {
		return strings.TrimSpace(t.Doc)
	}
	for _, spec := range t.Decl.Specs {
		ts, ok := spec.(*goast.TypeSpec)
		if !ok || ts.Name.Name != t.Name {
			continue
		}
		if it, ok := ts.Type.(*goast.InterfaceType); ok && it.Methods != nil {
			for _, field := range it.Methods.List {
				for _, id := range field.Names {
					if id.Name != methodName {
						continue
					}
					if d := strings.TrimSpace(field.Doc.Text()); d != "" {
						return d
					}
					return strings.TrimSpace(t.Doc)
				}
			}
			return ""
		}
		for _, m := range t.Methods {
			if m.Name == methodName {
				if d := strings.TrimSpace(m.Doc); d != "" {
					return d
				}
				return strings.TrimSpace(t.Doc)
			}
		}
	}
	return ""
}

// forstBuiltinReceiverGoType maps a Forst builtin (possibly under TypePointer) to the Go types.Type
// used for method-set lookup and the name of the type in package builtin's documentation.
func forstBuiltinReceiverGoType(t ast.TypeNode) (types.Type, string, bool) {
	if t.Ident == ast.TypePointer && len(t.TypeParams) == 1 {
		return forstBuiltinReceiverGoType(t.TypeParams[0])
	}
	switch t.Ident {
	case ast.TypeError:
		obj := types.Universe.Lookup("error")
		if obj == nil {
			return nil, "", false
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			return nil, "", false
		}
		return tn.Type(), "error", true
	case ast.TypeString:
		return types.Typ[types.String], "string", true
	case ast.TypeInt:
		return types.Typ[types.Int], "int", true
	case ast.TypeBool:
		return types.Typ[types.Bool], "bool", true
	case ast.TypeFloat:
		return types.Typ[types.Float64], "float64", true
	default:
		return nil, "", false
	}
}
