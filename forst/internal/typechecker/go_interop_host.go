package typechecker

import (
	"sort"
	"strconv"

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
	case ast.TypeFunc:
		return tc.goSignatureFromForstFunctionType(f)
	case ast.TypeShape:
		return tc.goTypeFromForstShapeType(f)
	default:
		if gt := tc.goTypeForForstUserType(f.Ident); gt != nil {
			return gt
		}
		return nil
	}
}

func (tc *TypeChecker) goSignatureFromForstFunctionType(f ast.TypeNode) types.Type {
	if !f.IsFunctionType() {
		return nil
	}
	params := make([]*types.Var, 0, len(f.FuncParams))
	for i, p := range f.FuncParams {
		sp, ok := p.(ast.SimpleParamNode)
		if !ok {
			return nil
		}
		pt := tc.goTypeForForstType(sp.Type)
		if pt == nil {
			return nil
		}
		name := string(sp.Ident.ID)
		if name == "" {
			name = "arg" + strconv.Itoa(i)
		}
		params = append(params, types.NewParam(0, nil, name, pt))
	}
	paramTuple := types.NewTuple(params...)

	var resultVars []*types.Var
	for _, rt := range f.FuncReturns {
		if rt.Ident == ast.TypeVoid {
			continue
		}
		gt := tc.goTypeForForstType(rt)
		if gt == nil {
			return nil
		}
		resultVars = append(resultVars, types.NewParam(0, nil, "", gt))
	}
	var resultTuple *types.Tuple
	if len(resultVars) == 0 {
		resultTuple = types.NewTuple()
	} else {
		resultTuple = types.NewTuple(resultVars...)
	}
	return types.NewSignatureType(nil, nil, nil, paramTuple, resultTuple, false)
}

func (tc *TypeChecker) goTypeFromForstShapeType(f ast.TypeNode) types.Type {
	fields, ok := tc.ShapeFieldsFromParamType(f)
	if !ok || len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	vars := make([]*types.Var, 0, len(names))
	for _, name := range names {
		sf := fields[name]
		tn, ok := ShapeFieldTypeNode(sf)
		if !ok {
			return nil
		}
		gt := tc.goTypeForForstType(tn)
		if gt == nil {
			return nil
		}
		goName := gointerop.ExportedFieldName(name)
		vars = append(vars, types.NewField(0, nil, goName, gt, false))
	}
	return types.NewStruct(vars, nil)
}

// goTypeForForstUserType builds a go/types named type with methods for nominal Forst types
// that declare receiver methods, enabling interface satisfaction checks at the FFI boundary.
func (tc *TypeChecker) goTypeForForstUserType(ident ast.TypeIdent) types.Type {
	if tc.TypeMethods == nil {
		return nil
	}
	methods, ok := tc.TypeMethods[ident]
	if !ok || len(methods) == 0 {
		return nil
	}
	obj := types.NewTypeName(0, nil, string(ident), nil)
	underlying := types.NewStruct(nil, nil)
	named := types.NewNamed(obj, underlying, nil)
	recv := types.NewVar(0, nil, "", named)
	for methodName, msig := range methods {
		goSig := tc.goSignatureFromForstFunctionSignature(msig, recv)
		if goSig == nil {
			continue
		}
		m := types.NewFunc(0, nil, methodName, goSig)
		named.AddMethod(m)
	}
	return named
}

func (tc *TypeChecker) goSignatureFromForstFunctionSignature(msig FunctionSignature, recv *types.Var) *types.Signature {
	params := make([]*types.Var, 0, len(msig.Parameters))
	for i, p := range msig.Parameters {
		pt := tc.goTypeForForstType(p.Type)
		if pt == nil {
			return nil
		}
		name := string(p.Ident.ID)
		if name == "" {
			name = "arg" + strconv.Itoa(i)
		}
		params = append(params, types.NewParam(0, nil, name, pt))
	}
	paramTuple := types.NewTuple(params...)
	var resultVars []*types.Var
	for _, rt := range msig.ReturnTypes {
		if rt.Ident == ast.TypeVoid {
			continue
		}
		gt := tc.goTypeForForstType(rt)
		if gt == nil {
			return nil
		}
		resultVars = append(resultVars, types.NewParam(0, nil, "", gt))
	}
	var resultTuple *types.Tuple
	if len(resultVars) == 0 {
		resultTuple = types.NewTuple()
	} else {
		resultTuple = types.NewTuple(resultVars...)
	}
	return types.NewSignatureType(recv, nil, nil, paramTuple, resultTuple, false)
}
