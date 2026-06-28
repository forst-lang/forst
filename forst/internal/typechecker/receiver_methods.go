package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func receiverTypeIdentFromFn(fn *ast.FunctionNode) ast.TypeIdent {
	if fn == nil || fn.Receiver == nil {
		return ""
	}
	recvType := fn.Receiver.Type.Ident
	if fn.Receiver.Type.Ident == ast.TypePointer && len(fn.Receiver.Type.TypeParams) == 1 {
		recvType = fn.Receiver.Type.TypeParams[0].Ident
	}
	return recvType
}

// IsVoidReturnTypes reports whether a function return type list is empty or explicit Void.
func IsVoidReturnTypes(types []ast.TypeNode) bool {
	if len(types) == 0 {
		return true
	}
	return len(types) == 1 && types[0].Ident == ast.TypeVoid
}

func (tc *TypeChecker) registerTypeMethod(recvType ast.TypeIdent, methodName string, fn ast.FunctionNode) {
	if tc.TypeMethods == nil {
		tc.TypeMethods = make(map[ast.TypeIdent]map[string]FunctionSignature)
	}
	if tc.TypeMethods[recvType] == nil {
		tc.TypeMethods[recvType] = make(map[string]FunctionSignature)
	}

	params := make([]ParameterSignature, len(fn.Params))
	for i, param := range fn.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			params[i] = ParameterSignature{Ident: p.Ident, Type: p.Type}
		case ast.DestructuredParamNode:
			params[i] = ParameterSignature{
				Ident: ast.Ident{ID: ast.Identifier(p.GetIdent())},
				Type:  p.Type,
			}
		}
	}

	processedReturnTypes := make([]ast.TypeNode, len(fn.ReturnTypes))
	for i, returnType := range fn.ReturnTypes {
		if returnType.TypeKind != ast.TypeKindHashBased && !tc.isBuiltinType(returnType.Ident) {
			processedReturnTypes[i] = ensureUserDefinedType(returnType)
		} else {
			processedReturnTypes[i] = returnType
		}
	}

	tc.TypeMethods[recvType][methodName] = FunctionSignature{
		Ident:       fn.Ident,
		Parameters:  params,
		ReturnTypes: processedReturnTypes,
	}

	tc.log.WithFields(logrus.Fields{
		"function":   "registerTypeMethod",
		"recvType":   recvType,
		"methodName": methodName,
	}).Trace("Registered receiver method")
}

func (tc *TypeChecker) lookupTypeMethod(typeIdent ast.TypeIdent, methodName string) (FunctionSignature, bool) {
	if tc.TypeMethods == nil {
		return FunctionSignature{}, false
	}
	methods, ok := tc.TypeMethods[typeIdent]
	if !ok {
		return FunctionSignature{}, false
	}
	sig, ok := methods[methodName]
	return sig, ok
}

func (tc *TypeChecker) checkUserTypeMethod(typ ast.TypeNode, methodName string, args []ast.ExpressionNode) ([]ast.TypeNode, error) {
	typeIdent := typ.Ident
	if typ.Ident == ast.TypePointer && len(typ.TypeParams) == 1 {
		typeIdent = typ.TypeParams[0].Ident
	}

	sig, ok := tc.lookupTypeMethod(typeIdent, methodName)
	if !ok {
		return nil, fmt.Errorf("method %s() is not defined on type %s", methodName, typeIdent)
	}

	if len(args) != len(sig.Parameters) {
		return nil, fmt.Errorf("method %s() expects %d arguments, got %d", methodName, len(sig.Parameters), len(args))
	}

	for i, arg := range args {
		argTypes, err := tc.inferExpressionType(arg)
		if err != nil {
			return nil, err
		}
		if len(argTypes) != 1 {
			return nil, fmt.Errorf("method %s() argument %d must have a single type", methodName, i+1)
		}
		if !tc.IsTypeCompatible(argTypes[0], sig.Parameters[i].Type) {
			return nil, fmt.Errorf("method %s() argument %d: type %s is not compatible with %s",
				methodName, i+1, argTypes[0].Ident, sig.Parameters[i].Type.Ident)
		}
	}

	if len(sig.ReturnTypes) == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil
	}
	return sig.ReturnTypes, nil
}

func (tc *TypeChecker) checkContractShapeMethod(typ ast.TypeNode, methodName string, args []ast.ExpressionNode) ([]ast.TypeNode, error) {
	def, ok := tc.Defs[typ.Ident]
	if !ok {
		return nil, fmt.Errorf("method %s() is not defined on type %s", methodName, typ.Ident)
	}
	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("method %s() is not defined on type %s", methodName, typ.Ident)
	}
	fields := tc.typeDefMethodOnlyFields(typeDef)
	if fields == nil {
		return nil, fmt.Errorf("method %s() is not defined on type %s", methodName, typ.Ident)
	}
	field, ok := fields[methodName]
	if !ok || !field.IsMethod {
		return nil, fmt.Errorf("method %s() is not defined on type %s", methodName, typ.Ident)
	}
	if len(args) != len(field.MethodParams) {
		return nil, fmt.Errorf("method %s() expects %d arguments, got %d", methodName, len(field.MethodParams), len(args))
	}
	for i, arg := range args {
		argTypes, err := tc.inferExpressionType(arg)
		if err != nil {
			return nil, err
		}
		if len(argTypes) != 1 {
			return nil, fmt.Errorf("method %s() argument %d must have a single type", methodName, i+1)
		}
		sp, ok := field.MethodParams[i].(ast.SimpleParamNode)
		if !ok {
			return nil, fmt.Errorf("method %s() has invalid parameter %d", methodName, i+1)
		}
		if !tc.IsTypeCompatible(argTypes[0], sp.Type) {
			return nil, fmt.Errorf("method %s() argument %d: type %s is not compatible with %s",
				methodName, i+1, argTypes[0].Ident, sp.Type.Ident)
		}
	}
	if len(field.MethodReturnTypes) == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil
	}
	return field.MethodReturnTypes, nil
}

func (tc *TypeChecker) typeMethodsSatisfyContract(typeIdent ast.TypeIdent, contract ast.ShapeNode) bool {
	for name, contractField := range contract.Fields {
		if !contractField.IsMethod {
			return false
		}
		sig, ok := tc.lookupTypeMethod(typeIdent, name)
		if !ok {
			return false
		}
		if len(contractField.MethodParams) != len(sig.Parameters) {
			return false
		}
		for i, param := range contractField.MethodParams {
			sp, ok := param.(ast.SimpleParamNode)
			if !ok {
				return false
			}
			if !tc.IsTypeCompatible(sp.Type, sig.Parameters[i].Type) &&
				!tc.IsTypeCompatible(sig.Parameters[i].Type, sp.Type) {
				return false
			}
		}
		if len(contractField.MethodReturnTypes) != len(sig.ReturnTypes) {
			return false
		}
		for i, ret := range contractField.MethodReturnTypes {
			if !tc.IsTypeCompatible(sig.ReturnTypes[i], ret) &&
				!tc.IsTypeCompatible(ret, sig.ReturnTypes[i]) {
				return false
			}
		}
	}
	return true
}

// validateInferredReceiverMethodReturn errors when a receiver method's inferred return
// conflicts with a void method on a method-only contract type in the same package.
func (tc *TypeChecker) validateInferredReceiverMethodReturn(fn ast.FunctionNode, inferred []ast.TypeNode) error {
	if fn.Receiver == nil || len(fn.ReturnTypes) > 0 {
		return nil
	}
	if IsVoidReturnTypes(inferred) {
		return nil
	}
	recvType := receiverTypeIdentFromFn(&fn)
	methodName := string(fn.Ident.ID)
	sig, ok := tc.lookupTypeMethod(recvType, methodName)
	if !ok {
		return nil
	}

	for contractIdent, def := range tc.Defs {
		shape, ok := tc.getShapeFromTypeDef(def)
		if !ok || !shape.IsMethodOnlyContract() {
			continue
		}
		contractField, ok := shape.Fields[methodName]
		if !ok || !contractField.IsMethod {
			continue
		}
		if len(contractField.MethodReturnTypes) > 0 {
			continue
		}
		if len(contractField.MethodParams) != len(sig.Parameters) {
			continue
		}
		paramsMatch := true
		for i, param := range contractField.MethodParams {
			sp, ok := param.(ast.SimpleParamNode)
			if !ok {
				paramsMatch = false
				break
			}
			if !tc.IsTypeCompatible(sp.Type, sig.Parameters[i].Type) &&
				!tc.IsTypeCompatible(sig.Parameters[i].Type, sp.Type) {
				paramsMatch = false
				break
			}
		}
		if !paramsMatch {
			continue
		}
		return fmt.Errorf(
			"method %s on %s: inferred return %s from function body, but %s contract requires void (use println for side effects, or declare an explicit return type to opt in)",
			methodName, recvType, formatTypeList(inferred), contractIdent,
		)
	}
	return nil
}

func (tc *TypeChecker) registerGoQualifiedTypeAlias(ident ast.TypeIdent, base ast.TypeIdent) {
	if !strings.Contains(string(base), ".") {
		return
	}
	if tc.goQualifiedTypeAliases == nil {
		tc.goQualifiedTypeAliases = make(map[ast.TypeIdent]string)
	}
	tc.goQualifiedTypeAliases[ident] = string(base)
}

func (tc *TypeChecker) resolveGoQualifiedTypeName(ident ast.TypeIdent) (string, bool) {
	if tc.goQualifiedTypeAliases != nil {
		if q, ok := tc.goQualifiedTypeAliases[ident]; ok {
			return q, true
		}
	}
	if strings.Contains(string(ident), ".") {
		return string(ident), true
	}
	// Follow simple alias chains: type W = io.Writer stored as TypeDefAssertionExpr.
	if def, ok := tc.Defs[ident]; ok {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if assertionExpr, ok := typeDefAssertionFromExpr(typeDef.Expr); ok && assertionExpr.Assertion != nil &&
				assertionExpr.Assertion.BaseType != nil && len(assertionExpr.Assertion.Constraints) == 0 {
				base := *assertionExpr.Assertion.BaseType
				if strings.Contains(string(base), ".") {
					return string(base), true
				}
			}
		}
	}
	return "", false
}

// ProviderContractKey returns a stable identity string for a Provider contract type (named or Go-qualified).
func (tc *TypeChecker) ProviderContractKey(t ast.TypeNode) string {
	if q, ok := tc.resolveGoQualifiedTypeName(t.Ident); ok {
		return q
	}
	return string(t.Ident)
}
