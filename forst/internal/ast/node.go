package ast

// Node is the interface for all AST nodes
type Node interface {
	Kind() NodeKind
	String() string
}

// NodeKind is the kind of AST node
type NodeKind string

// NodeKindFunction is the kind for function nodes
// NodeKindBlock is the kind for block nodes
// NodeKindIf is the kind for if nodes
// NodeKindElse is the kind for else nodes
// NodeKindWhile is the kind for while nodes
// NodeKindUnaryExpression is the kind for unary expression nodes
// NodeKindBinaryExpression is the kind for binary expression nodes
// NodeKindFunctionCall is the kind for function call nodes
// NodeKindIntLiteral is the kind for integer literal nodes
// NodeKindFloatLiteral is the kind for float literal nodes
// NodeKindStringLiteral is the kind for string literal nodes
// NodeKindBoolLiteral is the kind for boolean literal nodes
// NodeKindIdentifier is the kind for identifier nodes
// NodeKindEnsure is the kind for ensure nodes
// NodeKindAssertion is the kind for assertion nodes
// NodeKindImport is the kind for import nodes
// NodeKindImportGroup is the kind for import group nodes
// NodeKindPackage is the kind for package nodes
// NodeKindVariable is the kind for variable nodes
// NodeKindSimpleParam is the kind for simple parameter nodes
// NodeKindDestructuredParam is the kind for destructured parameter nodes
// NodeKindReturn is the kind for return nodes
// NodeKindType is the kind for type nodes
// NodeKindTypeDef is the kind for type definition nodes
// NodeKindTypeDefAssertion is the kind for type definition assertion nodes
// NodeKindTypeDefBinaryExpr is the kind for type definition binary expression nodes
// NodeKindEnsureBlock is the kind for ensure block nodes
// NodeKindAssignment is the kind for assignment nodes
// NodeKindShape is the kind for shape nodes
// NodeKindTypeGuard is the kind for type guard nodes
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
	NodeKindTypeDefAssertion  NodeKind = "TypeDefAssertion"
	NodeKindTypeDefBinaryExpr NodeKind = "TypeDefBinaryExpr"
	NodeKindEnsureBlock       NodeKind = "EnsureBlock"
	NodeKindAssignment        NodeKind = "Assignment"
	NodeKindShape             NodeKind = "Shape"
	NodeKindTypeGuard         NodeKind = "TYPE_GUARD"
)
