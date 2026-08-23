package gointerop

import (
	"go/types"

	"forst/internal/ast"
)

// FuncCall describes a Go package-level function call at the Forst↔Go boundary.
type FuncCall struct {
	Pkg             *types.Package
	QualDisplay     string
	FuncName        string
	Call            ast.FunctionCallNode
	ArgTypes        [][]ast.TypeNode
	WantSingleValue bool
	// RequireExported rejects unexported symbols (imported packages). Same-package calls leave this false.
	RequireExported bool
}

// SignatureCheck validates arguments and maps results for a Go function/method signature.
type SignatureCheck struct {
	Sig             *types.Signature
	Qual            string
	Call            ast.FunctionCallNode
	ArgTypes        [][]ast.TypeNode
	WantSingleValue bool
}

// MethodCall describes a Go method call when the receiver has a tracked go/types type.
type MethodCall struct {
	Recv            types.Type
	MethodName      string
	Call            ast.FunctionCallNode
	ArgTypes        [][]ast.TypeNode
	WantSingleValue bool
}

// ParamAssignability checks one Forst argument against one Go parameter.
type ParamAssignability struct {
	Qual    string
	Index   int
	GoParam types.Type
	ArgType []ast.TypeNode
	Call    ast.FunctionCallNode
	ArgIdx  int
}

// SpreadAssignability checks a variadic spread argument against a Go slice element type.
type SpreadAssignability struct {
	Qual       string
	Elem       types.Type
	SpreadType ast.TypeNode
	Call       ast.FunctionCallNode
	ArgIdx     int
}
