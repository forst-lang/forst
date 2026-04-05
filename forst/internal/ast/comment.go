package ast

import "fmt"

// CommentNode is a line or block comment preserved from the source (// ... or /* ... */).
type CommentNode struct {
	Text string
}

// Kind returns NodeKindComment.
func (c CommentNode) Kind() NodeKind {
	return NodeKindComment
}

func (c CommentNode) String() string {
	return fmt.Sprintf("Comment(%d)", len(c.Text))
}
