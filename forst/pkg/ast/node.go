package ast

// AST Node interface
type Node interface {
	NodeType() NodeType
	String() string
}

type NodeType string

const (
	NodeTypeFunction         NodeType = "Function"
	NodeTypeBlock            NodeType = "Block"
	NodeTypeIf               NodeType = "If"
	NodeTypeElse             NodeType = "Else"
	NodeTypeWhile            NodeType = "While"
	NodeTypeUnaryExpression  NodeType = "UnaryExpression"
	NodeTypeBinaryExpression NodeType = "BinaryExpression"
	NodeTypeFunctionCall     NodeType = "FunctionCall"
	NodeTypeIntLiteral       NodeType = "IntLiteral"
	NodeTypeFloatLiteral     NodeType = "FloatLiteral"
	NodeTypeStringLiteral    NodeType = "StringLiteral"
	NodeTypeBoolLiteral      NodeType = "BoolLiteral"
	NodeTypeIdentifier       NodeType = "Identifier"
	NodeTypeEnsure           NodeType = "Ensure"
	NodeTypeAssertion        NodeType = "Assertion"
	NodeTypeImport           NodeType = "Import"
	NodeTypeImportGroup      NodeType = "ImportGroup"
	NodeTypePackage          NodeType = "Package"
	NodeTypeVariable         NodeType = "Variable"
	NodeTypeParam            NodeType = "Param"
	NodeTypeReturn           NodeType = "Return"
	NodeTypeType             NodeType = "Type"
	NodeTypeEnsureBlock      NodeType = "EnsureBlock"
	NodeTypeAssignment       NodeType = "Assignment"
)
