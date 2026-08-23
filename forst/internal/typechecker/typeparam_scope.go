package typechecker

import (
	"forst/internal/ast"
)

// typeParamSet is the set of type parameter names declared on a generic function.
type typeParamSet map[ast.TypeIdent]struct{}

func newTypeParamSet(decls []ast.TypeParamDecl) typeParamSet {
	if len(decls) == 0 {
		return nil
	}
	s := make(typeParamSet, len(decls))
	for _, tp := range decls {
		s[ast.TypeIdent(tp.Name)] = struct{}{}
	}
	return s
}

func (s typeParamSet) contains(id ast.TypeIdent) bool {
	if s == nil {
		return false
	}
	_, ok := s[id]
	return ok
}

func (s typeParamSet) names() []ast.TypeIdent {
	if len(s) == 0 {
		return nil
	}
	out := make([]ast.TypeIdent, 0, len(s))
	for name := range s {
		out = append(out, name)
	}
	return out
}

// normalizeTypesWithTypeParams rewrites identifiers in params to TypeKindTypeParam.
func normalizeTypesWithTypeParams(t ast.TypeNode, params typeParamSet) ast.TypeNode {
	if params.contains(t.Ident) && t.Assertion == nil && len(t.TypeParams) == 0 && t.Ident != ast.TypeFunc {
		return ast.NewTypeParamType(t.Ident)
	}
	out := t
	if t.Assertion != nil {
		asn := *t.Assertion
		if asn.BaseType != nil {
			base := normalizeTypesWithTypeParams(ast.TypeNode{Ident: *asn.BaseType}, params)
			id := base.Ident
			asn.BaseType = &id
		}
		if len(asn.Constraints) > 0 {
			constraints := make([]ast.ConstraintNode, len(asn.Constraints))
			for i, c := range asn.Constraints {
				constraints[i] = c
				if len(c.Args) == 0 {
					continue
				}
				args := make([]ast.ConstraintArgumentNode, len(c.Args))
				for j, a := range c.Args {
					args[j] = a
					if a.Shape != nil {
						sh := *a.Shape
						fields := make(map[string]ast.ShapeFieldNode, len(sh.Fields))
						for name, f := range sh.Fields {
							sf := f
							if f.Type != nil {
								tn := normalizeTypesWithTypeParams(*f.Type, params)
								sf.Type = &tn
							}
							fields[name] = sf
						}
						sh.Fields = fields
						args[j].Shape = &sh
					}
					if a.Type != nil {
						tn := normalizeTypesWithTypeParams(*a.Type, params)
						args[j].Type = &tn
					}
				}
				constraints[i].Args = args
			}
			asn.Constraints = constraints
		}
		out.Assertion = &asn
	}
	if len(t.TypeParams) > 0 {
		out.TypeParams = make([]ast.TypeNode, len(t.TypeParams))
		for i, p := range t.TypeParams {
			out.TypeParams[i] = normalizeTypesWithTypeParams(p, params)
		}
	}
	if t.ArrayLen != nil {
		n := *t.ArrayLen
		out.ArrayLen = &n
	}
	if len(t.FuncParams) > 0 {
		out.FuncParams = make([]ast.ParamNode, len(t.FuncParams))
		for i, p := range t.FuncParams {
			switch param := p.(type) {
			case ast.SimpleParamNode:
				out.FuncParams[i] = ast.SimpleParamNode{
					Ident:    param.Ident,
					Type:     normalizeTypesWithTypeParams(param.Type, params),
					Variadic: param.Variadic,
				}
			case ast.DestructuredParamNode:
				out.FuncParams[i] = ast.DestructuredParamNode{
					Fields: param.Fields,
					Type:   normalizeTypesWithTypeParams(param.Type, params),
				}
			}
		}
	}
	if len(t.FuncReturns) > 0 {
		out.FuncReturns = make([]ast.TypeNode, len(t.FuncReturns))
		for i, r := range t.FuncReturns {
			out.FuncReturns[i] = normalizeTypesWithTypeParams(r, params)
		}
	}
	return out
}

func normalizeGenericSignature(fn ast.FunctionNode) FunctionSignature {
	tpSet := newTypeParamSet(fn.TypeParams)
	params := make([]ParameterSignature, len(fn.Params))
	for i, param := range fn.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			params[i] = ParameterSignature{
				Ident:    p.Ident,
				Type:     normalizeTypesWithTypeParams(p.Type, tpSet),
				Variadic: p.Variadic,
			}
		case ast.DestructuredParamNode:
			params[i] = ParameterSignature{
				Ident: ast.Ident{ID: ast.Identifier(p.GetIdent())},
				Type:  normalizeTypesWithTypeParams(p.Type, tpSet),
			}
		}
	}
	returnTypes := make([]ast.TypeNode, len(fn.ReturnTypes))
	for i, rt := range fn.ReturnTypes {
		if tpSet.contains(rt.Ident) && rt.Assertion == nil && len(rt.TypeParams) == 0 {
			returnTypes[i] = ast.NewTypeParamType(rt.Ident)
		} else if rt.TypeKind != ast.TypeKindHashBased && !isBuiltinTypeIdent(rt.Ident) {
			returnTypes[i] = normalizeTypesWithTypeParams(ensureUserDefinedType(rt), tpSet)
		} else {
			returnTypes[i] = normalizeTypesWithTypeParams(rt, tpSet)
		}
	}
	return FunctionSignature{
		Ident:       fn.Ident,
		TypeParams:  append([]ast.TypeParamDecl(nil), fn.TypeParams...),
		Parameters:  params,
		ReturnTypes: returnTypes,
		TypeParamNames: tpSet,
	}
}

func isBuiltinTypeIdent(id ast.TypeIdent) bool {
	switch id {
	case ast.TypeInt, ast.TypeFloat, ast.TypeString, ast.TypeBool, ast.TypeVoid, ast.TypeError,
		ast.TypeBytes, ast.TypeObject, ast.TypeArray, ast.TypeMap, ast.TypeShape, ast.TypePointer,
		ast.TypeResult, ast.TypeTuple, ast.TypeChannel, ast.TypeUnion, ast.TypeIntersection,
		ast.TypeAssertion, ast.TypeImplicit, ast.TypeComplex64, ast.TypeComplex128:
		return true
	default:
		return false
	}
}
