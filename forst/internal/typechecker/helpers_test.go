package typechecker

import (
	"forst/internal/ast"
	"reflect"
	"testing"
)

// TestMergeShapeFieldsFromConstraints tests the MergeShapeFieldsFromConstraints function
// which merges shape fields from constraint arguments, with later constraints taking precedence.
func TestMergeShapeFieldsFromConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints []ast.ConstraintNode
		expected    map[string]ast.ShapeFieldNode
	}{
		{
			name:        "returns empty map for empty constraints",
			constraints: []ast.ConstraintNode{},
			expected:    map[string]ast.ShapeFieldNode{},
		},
		{
			name: "merges fields from single constraint",
			constraints: []ast.ConstraintNode{
				makeConstraint("HasFields", makeShape(map[string]ast.ShapeFieldNode{
					"name": makeTypeField(ast.TypeString),
					"age":  makeTypeField(ast.TypeInt),
				})),
			},
			expected: map[string]ast.ShapeFieldNode{
				"name": makeTypeField(ast.TypeString),
				"age":  makeTypeField(ast.TypeInt),
			},
		},
		{
			name: "merges fields from multiple constraints",
			constraints: []ast.ConstraintNode{
				makeConstraint("HasName", makeShape(map[string]ast.ShapeFieldNode{
					"name": makeTypeField(ast.TypeString),
				})),
				makeConstraint("HasAge", makeShape(map[string]ast.ShapeFieldNode{
					"age": makeTypeField(ast.TypeInt),
				})),
			},
			expected: map[string]ast.ShapeFieldNode{
				"name": makeTypeField(ast.TypeString),
				"age":  makeTypeField(ast.TypeInt),
			},
		},
		{
			name: "resolves field conflicts with last constraint winning",
			constraints: []ast.ConstraintNode{
				makeConstraint("FirstConstraint", makeShape(map[string]ast.ShapeFieldNode{
					"field": makeTypeField(ast.TypeString),
				})),
				makeConstraint("SecondConstraint", makeShape(map[string]ast.ShapeFieldNode{
					"field": makeTypeField(ast.TypeInt),
				})),
			},
			expected: map[string]ast.ShapeFieldNode{
				"field": makeTypeField(ast.TypeInt),
			},
		},
		{
			name: "ignores non-shape arguments",
			constraints: []ast.ConstraintNode{
				{
					Name: "HasValue",
					Args: []ast.ConstraintArgumentNode{
						{Value: makeValueNode(42)},
						{Shape: makeShape(map[string]ast.ShapeFieldNode{
							"value": makeTypeField(ast.TypeInt),
						})},
					},
				},
			},
			expected: map[string]ast.ShapeFieldNode{
				"value": makeTypeField(ast.TypeInt),
			},
		},
		{
			name: "handles complex nested shapes",
			constraints: []ast.ConstraintNode{
				makeConstraint("UserProfile", makeShape(map[string]ast.ShapeFieldNode{
					"profile": makeShapeField(map[string]ast.ShapeFieldNode{
						"avatar": makeTypeField(ast.TypeString),
					}),
					"settings": makeShapeField(map[string]ast.ShapeFieldNode{
						"theme": makeTypeField(ast.TypeString),
					}),
				})),
			},
			expected: map[string]ast.ShapeFieldNode{
				"profile": makeShapeField(map[string]ast.ShapeFieldNode{
					"avatar": makeTypeField(ast.TypeString),
				}),
				"settings": makeShapeField(map[string]ast.ShapeFieldNode{
					"theme": makeTypeField(ast.TypeString),
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeShapeFieldsFromConstraints(tt.constraints)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeShapeFieldsFromConstraints() = %v, want %v", result, tt.expected)
			}
		})
	}
}
