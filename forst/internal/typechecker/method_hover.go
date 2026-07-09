package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// ForstReceiverMethodHoverMarkdown returns hover text for recv.method on a Forst type: registered
// receiver methods (func (T) m) or method-only contract members (Provider shapes).
func (tc *TypeChecker) ForstReceiverMethodHoverMarkdown(receiverType ast.TypeNode, methodName string) (string, bool) {
	if tc == nil || methodName == "" {
		return "", false
	}
	typeIdent := receiverType.Ident
	if receiverType.Ident == ast.TypePointer && len(receiverType.TypeParams) == 1 {
		typeIdent = receiverType.TypeParams[0].Ident
	}
	if typeIdent == "" {
		return "", false
	}

	if sig, ok := tc.lookupTypeMethod(typeIdent, methodName); ok {
		display := tc.FormatFunctionSignatureDisplay(sig)
		body := fmt.Sprintf("func (%s) %s", typeIdent, display)
		return fmt.Sprintf("**Method** on `%s`\n\n```forst\n%s\n```", typeIdent, body), true
	}

	def, ok := tc.Defs[typeIdent]
	if !ok {
		return "", false
	}
	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		return "", false
	}
	fields := tc.typeDefMethodOnlyFields(typeDef)
	if fields == nil {
		return "", false
	}
	field, ok := fields[methodName]
	if !ok || !field.IsMethod {
		return "", false
	}
	sigLine := methodName + field.MethodSignatureString()
	return fmt.Sprintf("**Contract method** on `%s`\n\n```forst\n%s\n```", typeIdent, sigLine), true
}
