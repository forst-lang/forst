package ast

import "testing"

func TestConstraints_ValueConstraint_name(t *testing.T) {
	if ValueConstraint != "Value" {
		t.Fatal(ValueConstraint)
	}
}
