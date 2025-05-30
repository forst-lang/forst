package ast

// AST Node interface
type Node interface {
	Kind() NodeKind
	String() string
}

type NodeKind string

const (
	NodeKindFunction          NodeKind = "Function"
	NodeKindBlock             NodeKind = "Block"
	NodeKindIf                NodeKind = "If"
	NodeKindElse              NodeKind = "Else"
	NodeKindWhile             NodeKind = "While"
	NodeKindUnaryExpression   NodeKind = "UnaryExpression"
	NodeKindBinaryExpression  NodeKind = "BinaryExpression"
	NodeKindFunctionCall      NodeKind = "FunctionCall"
	NodeKindIntLiteral        NodeKind = "IntLiteral"
	NodeKindFloatLiteral      NodeKind = "FloatLiteral"
	NodeKindStringLiteral     NodeKind = "StringLiteral"
	NodeKindBoolLiteral       NodeKind = "BoolLiteral"
	NodeKindIdentifier        NodeKind = "Identifier"
	NodeKindEnsure            NodeKind = "Ensure"
	NodeKindAssertion         NodeKind = "Assertion"
	NodeKindImport            NodeKind = "Import"
	NodeKindImportGroup       NodeKind = "ImportGroup"
	NodeKindPackage           NodeKind = "Package"
	NodeKindVariable          NodeKind = "Variable"
	NodeKindSimpleParam       NodeKind = "SimpleParam"
	NodeKindDestructuredParam NodeKind = "DestructuredParam"
	NodeKindReturn            NodeKind = "Return"
	NodeKindType              NodeKind = "Type"
	NodeKindTypeDef           NodeKind = "TypeDef"
	NodeKindEnsureBlock       NodeKind = "EnsureBlock"
	NodeKindAssignment        NodeKind = "Assignment"
	NodeKindShape             NodeKind = "Shape"
	NodeKindTypeGuard         NodeKind = "TYPE_GUARD"
)
