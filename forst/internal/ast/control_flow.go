package ast

import "fmt"

// IfNode represents an if statement in the AST
type IfNode struct {
	// The condition to be evaluated
	Condition Node
	// The body of statements to execute if condition is true
	Body []Node
	// Optional else-if branches (nil if no else-if clauses)
	ElseIfs []ElseIfNode
	// Optional else branch (nil if no else clause)
	Else *ElseBlockNode
	// Optional initialization statement (nil if no init)
	Init Node
}

type ElseIfNode struct {
	Condition Node
	Body      []Node
}

type ElseBlockNode struct {
	Body []Node
}

// Kind returns the node kind for if statements
func (i IfNode) Kind() NodeKind {
	return NodeKindIf
}

func (i IfNode) String() string {
	var result string
	if i.Init != nil {
		result = fmt.Sprintf("If(%v; %v)", i.Init, i.Condition)
	} else {
		result = fmt.Sprintf("If(%v)", i.Condition)
	}

	for _, elseIf := range i.ElseIfs {
		result += fmt.Sprintf(" ElseIf(%v)", elseIf.Condition)
	}

	if i.Else != nil {
		result += " Else"
	}
	return result
}

// Kind returns the node kind for else-if blocks
func (e ElseIfNode) Kind() NodeKind {
	return NodeKindElseIf
}

func (e ElseIfNode) String() string {
	return fmt.Sprintf("ElseIf(%v){%v}", e.Condition, e.Body)
}

// Kind returns the node kind for else blocks
func (e ElseBlockNode) Kind() NodeKind {
	return NodeKindElseBlock
}

func (e ElseBlockNode) String() string {
	return fmt.Sprintf("Else{%v}", e.Body)
}
