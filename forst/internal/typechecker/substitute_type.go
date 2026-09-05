package typechecker

import "forst/internal/ast"

// SubstituteType replaces type parameters in t according to bindings.
func SubstituteType(t ast.TypeNode, bindings map[ast.TypeIdent]ast.TypeNode, isTypeParam func(ast.TypeNode) bool) ast.TypeNode {
	if isTypeParam != nil && isTypeParam(t) {
		if bound, ok := bindings[t.Ident]; ok {
			return bound
		}
		return t
	}
	out := t
	if t.Assertion != nil {
		asn := *t.Assertion
		if asn.BaseType != nil {
			base := SubstituteType(ast.TypeNode{Ident: *asn.BaseType, TypeKind: ast.TypeKindTypeParam}, bindings, isTypeParam)
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
								sub := SubstituteType(*f.Type, bindings, isTypeParam)
								sf.Type = &sub
							}
							fields[name] = sf
						}
						sh.Fields = fields
						args[j].Shape = &sh
					}
					if a.Type != nil {
						sub := SubstituteType(*a.Type, bindings, isTypeParam)
						args[j].Type = &sub
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
			out.TypeParams[i] = SubstituteType(p, bindings, isTypeParam)
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
					Type:     SubstituteType(param.Type, bindings, isTypeParam),
					Variadic: param.Variadic,
				}
			case ast.DestructuredParamNode:
				out.FuncParams[i] = ast.DestructuredParamNode{
					Fields: param.Fields,
					Type:   SubstituteType(param.Type, bindings, isTypeParam),
				}
			default:
				out.FuncParams[i] = p
			}
		}
	}
	if len(t.FuncReturns) > 0 {
		out.FuncReturns = make([]ast.TypeNode, len(t.FuncReturns))
		for i, r := range t.FuncReturns {
			out.FuncReturns[i] = SubstituteType(r, bindings, isTypeParam)
		}
	}
	return out
}

func (tc *TypeChecker) substituteTypeBindings(t ast.TypeNode, bindings map[ast.TypeIdent]ast.TypeNode) ast.TypeNode {
	return SubstituteType(t, bindings, tc.IsTypeParamType)
}
