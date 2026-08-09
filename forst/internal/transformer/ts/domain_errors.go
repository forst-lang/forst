package transformerts

import (
	"fmt"
	"sort"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

// UnknownFailureClass is the catch-all domain failure when wire tag is missing or unknown.
var UnknownFailureClass = ErrorClass{
	Name: "ForstUnknownFailure",
	Tag:  "ForstUnknownFailure",
	Fields: []ErrorField{
		{Name: "message", TSType: "string"},
		{Name: "serverError", TSType: "string", Optional: true},
		{Name: "tag", TSType: "string", Optional: true},
		{Name: "packageName", TSType: "string", Optional: true},
		{Name: "functionName", TSType: "string", Optional: true},
	},
}

// DomainErrorClassFromTypeDef builds a tagged error class from `error Name { ... }`.
func DomainErrorClassFromTypeDef(def ast.TypeDefNode, tc *typechecker.TypeChecker) (ErrorClass, error) {
	errEx, ok := def.Expr.(ast.TypeDefErrorExpr)
	if !ok {
		return ErrorClass{}, fmt.Errorf("not a nominal error typedef")
	}
	name := string(def.Ident)
	fields := make([]ErrorField, 0, len(errEx.Payload.Fields))
	names := make([]string, 0, len(errEx.Payload.Fields))
	for n := range errEx.Payload.Fields {
		names = append(names, n)
	}
	sort.Strings(names)
	mapper := NewTypeMapping()
	mapper.SetTypeChecker(tc)
	for _, fieldName := range names {
		field := errEx.Payload.Fields[fieldName]
		if field.Type == nil {
			continue
		}
		ts, err := mapper.GetTypeScriptType(field.Type)
		if err != nil {
			return ErrorClass{}, fmt.Errorf("field %s: %w", fieldName, err)
		}
		fields = append(fields, ErrorField{Name: fieldName, TSType: ts})
	}
	return ErrorClass{Name: name, Tag: name, Fields: fields}, nil
}

// MergeDomainErrors deduplicates domain error classes by name.
func MergeDomainErrors(list ...[]ErrorClass) []ErrorClass {
	seen := make(map[string]ErrorClass)
	var order []string
	for _, batch := range list {
		for _, c := range batch {
			if _, ok := seen[c.Name]; ok {
				continue
			}
			seen[c.Name] = c
			order = append(order, c.Name)
		}
	}
	sort.Strings(order)
	out := make([]ErrorClass, 0, len(order))
	for _, name := range order {
		out = append(out, seen[name])
	}
	return out
}

// CollectDomainErrorsFromTypeChecker returns ErrorClass entries for every nominal error typedef.
func CollectDomainErrorsFromTypeChecker(tc *typechecker.TypeChecker) ([]ErrorClass, error) {
	if tc == nil {
		return nil, nil
	}
	var names []string
	for id, def := range tc.Defs {
		node, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}
		if _, ok := node.Expr.(ast.TypeDefErrorExpr); ok {
			names = append(names, string(id))
		}
	}
	sort.Strings(names)
	out := make([]ErrorClass, 0, len(names))
	for _, name := range names {
		node := tc.Defs[ast.TypeIdent(name)].(ast.TypeDefNode)
		cls, err := DomainErrorClassFromTypeDef(node, tc)
		if err != nil {
			return nil, err
		}
		out = append(out, cls)
	}
	return out, nil
}

// FormatFunctionErrorUnion returns the TS union type for a function's failures.
func FormatFunctionErrorUnion(sig typechecker.FunctionSignature, domainByName map[string]ErrorClass) string {
	parts := make([]string, 0, 4+len(sig.ErrorSet.NominalErrors))
	for _, id := range sig.ErrorSet.NominalErrors {
		name := string(id)
		if _, ok := domainByName[name]; ok {
			parts = append(parts, name)
		}
	}
	if sig.ErrorSet.UnknownPossible {
		parts = append(parts, UnknownFailureClass.Name)
	}
	parts = append(parts, "InvokeFailure")
	if len(parts) == 1 {
		return parts[0]
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += " | " + p
	}
	return out
}
