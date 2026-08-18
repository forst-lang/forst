package typechecker

import (
	"go/types"

	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

type goInteropHost TypeChecker

func (tc *TypeChecker) goInteropHost() gointerop.Host {
	return (*goInteropHost)(tc)
}

func (tc *TypeChecker) goInteropDiag() gointerop.Diagnose {
	return func(span ast.SourceSpan, code, format string, args ...any) error {
		return diagnosticf(span, code, format, args...)
	}
}

func (h *goInteropHost) ForstTypeForGoType(g types.Type) (ast.TypeNode, bool) {
	return (*TypeChecker)(h).forstTypeForGoType(g)
}

func (h *goInteropHost) IsTypeCompatible(a, b ast.TypeNode) bool {
	return (*TypeChecker)(h).IsTypeCompatible(a, b)
}

func (h *goInteropHost) GoTypeForForstType(f ast.TypeNode) types.Type {
	return (*TypeChecker)(h).goTypeForForstType(f)
}

func (h *goInteropHost) InferExpressionType(expr ast.ExpressionNode) ([]ast.TypeNode, error) {
	return (*TypeChecker)(h).inferExpressionType(expr)
}

// GoPackageForImportLocal returns the loaded Go package for an import local name, if any.
func (tc *TypeChecker) GoPackageForImportLocal(local string) *types.Package {
	return tc.goPackageForImportLocal(local)
}

func (tc *TypeChecker) goTypeForForstType(f ast.TypeNode) types.Type {
	if gt := tc.goTypeForQualifiedImportTypeIdent(f.Ident); gt != nil {
		return gt
	}
	switch f.Ident {
	case ast.TypeBool:
		return types.Typ[types.Bool]
	case ast.TypeInt:
		return types.Typ[types.Int]
	case ast.TypeIdent("int8"):
		return types.Typ[types.Int8]
	case ast.TypeIdent("int16"):
		return types.Typ[types.Int16]
	case ast.TypeIdent("int32"):
		return types.Typ[types.Int32]
	case ast.TypeIdent("int64"):
		return types.Typ[types.Int64]
	case ast.TypeIdent("uint"):
		return types.Typ[types.Uint]
	case ast.TypeIdent("uint8"), ast.TypeIdent("byte"):
		return types.Typ[types.Uint8]
	case ast.TypeIdent("uint16"):
		return types.Typ[types.Uint16]
	case ast.TypeIdent("uint32"):
		return types.Typ[types.Uint32]
	case ast.TypeIdent("uint64"):
		return types.Typ[types.Uint64]
	case ast.TypeIdent("uintptr"):
		return types.Typ[types.Uintptr]
	case ast.TypeIdent("rune"):
		return types.Typ[types.Int32]
	case ast.TypeFloat:
		return types.Typ[types.Float64]
	case ast.TypeIdent("float32"):
		return types.Typ[types.Float32]
	case ast.TypeComplex64:
		return types.Typ[types.Complex64]
	case ast.TypeComplex128:
		return types.Typ[types.Complex128]
	case ast.TypeString:
		return types.Typ[types.String]
	case ast.TypeError:
		return gointerop.ErrorInterfaceType()
	case ast.TypePointer:
		if len(f.TypeParams) != 1 {
			return nil
		}
		inner := tc.goTypeForForstType(f.TypeParams[0])
		if inner == nil {
			return nil
		}
		return types.NewPointer(inner)
	case ast.TypeArray:
		if len(f.TypeParams) != 1 {
			return nil
		}
		elem := tc.goTypeForForstType(f.TypeParams[0])
		if elem == nil {
			return nil
		}
		if f.ArrayLen != nil {
			return types.NewArray(elem, *f.ArrayLen)
		}
		return types.NewSlice(elem)
	case ast.TypeMap:
		if len(f.TypeParams) != 2 {
			return nil
		}
		key := tc.goTypeForForstType(f.TypeParams[0])
		val := tc.goTypeForForstType(f.TypeParams[1])
		if key == nil || val == nil {
			return nil
		}
		return types.NewMap(key, val)
	case ast.TypeChannel:
		if len(f.TypeParams) != 1 {
			return nil
		}
		elem := tc.goTypeForForstType(f.TypeParams[0])
		if elem == nil {
			return nil
		}
		return types.NewChan(types.SendRecv, elem)
	default:
		return nil
	}
}
