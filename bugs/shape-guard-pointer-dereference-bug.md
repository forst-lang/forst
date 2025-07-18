# Shape Guard Pointer Dereference Bug

## Summary

The shape guard example fails to compile due to invalid Go syntax being generated for pointer fields. The transformer generates `&*User{name: "Alice"}` instead of the correct `&User{name: "Alice"}`.

## Error Message

```
invalid operation: cannot indirect User{…} (value of struct type User)
```

## Root Cause

The issue occurs in the pointer field handling logic in `transformShapeNodeWithExpectedType`. When a field is expected to be a pointer type (e.g., `*User`), the transformer incorrectly adds both `&` and `*` operators to struct literals.

### Evidence from Debug Logs

1. **User field is correctly identified as a pointer type:**

   ```
   DEBU[0000] === USER FIELD TYPE ANALYSIS ===              expectedFieldType=Pointer(User) fieldName=user fieldTypeName=*User function=transformShapeNodeWithExpectedType
   ```

2. **The field value is transformed into a pointer expression:**

   ```
   DEBU[0000] Field value after initial transformation      expectedFieldType=Pointer(User) fieldName=user fieldValue=&ast.UnaryExpr{OpPos:0, Op:17, X:(*ast.CompositeLit)(0x1400049a1c0)} fieldValueType=*ast.UnaryExpr function=transformShapeNodeWithExpectedType
   ```

3. **The pointer field logic correctly detects it's already a pointer:**

   ```
   DEBU[0000] Field value already pointer, emitting as-is (granular)  case=already-pointer fieldName=user fieldValue=&ast.UnaryExpr{OpPos:0, Op:17, X:(*ast.CompositeLit)(0x1400049a1c0)} function=transformShapeNodeWithExpectedType
   ```

4. **But the generated Go code shows the problematic syntax:**
   ```go
   user: &*User{name: "Alice"}
   ```

## Original Forst Code

```forst
func main() {
  sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
  name, err := createTask({
    ctx: {
      sessionId: &sessionId,
      user: {
        name: "Alice",
      },
    },
    input: {
      name: "Fix memory leak in Node.js app",
    },
  })
  // ...
}
```

The `user` field is a plain struct literal `{ name: "Alice" }` that should be converted to a pointer to match the `AppContext` type definition:

```forst
type AppContext = {
  sessionId: *String,
  user: *User,
}
```

## Expected Behavior

The transformer should generate:

```go
user: &User{name: "Alice"}
```

## Actual Behavior

The transformer generates:

```go
user: &*User{name: "Alice"}
```

## Technical Analysis

### AST Transformation Flow

1. **Forst AST Structure:**

   ```go
   // Original shape field in Forst AST
   ast.ShapeFieldNode{
       Shape: &ast.ShapeNode{
           Fields: map[string]ast.ShapeFieldNode{
               "name": ast.ShapeFieldNode{
                   Assertion: &ast.AssertionNode{
                       Constraints: []ast.ConstraintNode{
                           {Name: "Value", Args: []ast.ConstraintArgumentNode{
                               {Value: &ast.StringLiteralNode{Value: "Alice"}}
                           }}
                       }
                   }
               }
           }
       }
   }
   ```

2. **Expected Go AST:**

   ```go
   &goast.UnaryExpr{
       Op: token.AND,  // &
       X: &goast.CompositeLit{
           Type: goast.NewIdent("User"),
           Elts: []goast.Expr{
               &goast.KeyValueExpr{
                   Key:   goast.NewIdent("name"),
                   Value: &goast.BasicLit{Kind: token.STRING, Value: `"Alice"`},
               },
           },
       },
   }
   ```

3. **Actual Go AST (problematic):**
   ```go
   &goast.UnaryExpr{
       Op: token.AND,  // &
       X: &goast.UnaryExpr{
           Op: token.MUL,  // *
           X: &goast.CompositeLit{
               Type: goast.NewIdent("User"),
               Elts: []goast.Expr{...},
           },
       },
   }
   ```

### Code Location Analysis

**File:** `forst/internal/transformer/go/expression.go`
**Function:** `transformShapeNodeWithExpectedType`
**Lines:** ~450-550 (pointer field logic section)

The bug occurs in the pointer field handling section:

```go
// Pointer field logic
if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer {
    // ... existing logic ...
    if unaryExpr, isPointer := fieldValue.(*goast.UnaryExpr); isPointer && unaryExpr.Op == token.AND {
        // This case is correctly detected
        // But the fieldValue already contains the problematic &* structure
    } else if _, isStruct := fieldValue.(*goast.CompositeLit); isStruct {
        // This case should handle struct literals but doesn't
    }
}
```

### Root Cause Identification

The issue stems from the **recursive transformation** of nested shapes. When processing the `user` field:

1. **First transformation:** `{ name: "Alice" }` → `User{name: "Alice"}` (struct literal)
2. **Second transformation:** The struct literal gets wrapped with `&` for pointer field
3. **Third transformation:** Somehow a `*` operator is added, creating `&*User{name: "Alice"}`

The problem is in the **nested shape transformation logic** where the inner shape `{ name: "Alice" }` is being processed as a pointer type, but the outer context is also trying to make it a pointer.

### Specific Code Path

1. **Entry point:** `transformShapeNodeWithExpectedType` called with `expectedType=Pointer(User)`
2. **Field processing:** `user` field has `Shape` (not `Assertion` or `Type`)
3. **Recursive call:** `transformShapeNodeWithExpectedType(field.Shape, expectedFieldType)`
4. **Inner transformation:** The inner shape `{ name: "Alice" }` gets transformed to `User{name: "Alice"}`
5. **Pointer wrapping:** The result gets wrapped with `&` for pointer field
6. **Bug:** An additional `*` operator is added somewhere in this process

## Location of the Bug

The bug is in the pointer field logic in `forst/internal/transformer/go/expression.go` in the `transformShapeNodeWithExpectedType` function. The issue occurs when:

1. A field is expected to be a pointer type (`expectedFieldType.Ident == ast.TypePointer`)
2. The field value is a struct literal (`CompositeLit`)
3. The pointer field logic incorrectly wraps the struct literal with both `&` and `*` operators

### Specific Code Sections

**Lines 450-550 in `expression.go`:**

```go
// Pointer field logic
if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer {
    // Log the raw AST node type and value before any wrapping
    t.log.WithFields(map[string]interface{}{
        "fieldName":         name,
        "expectedFieldType": expectedFieldType,
        "fieldValueType":    fmt.Sprintf("%T", fieldValue),
        "fieldValue":        fmt.Sprintf("%#v", fieldValue),
        "function":          "transformShapeNodeWithExpectedType",
    }).Debug("[granular] Before pointer field logic")

    // If value is nil, emit nil
    if ident, ok := fieldValue.(*goast.Ident); ok && ident.Name == "nil" {
        // emit nil
    } else if unaryExpr, isPointer := fieldValue.(*goast.UnaryExpr); isPointer && unaryExpr.Op == token.AND {
        // emit as-is
    } else if _, isStruct := fieldValue.(*goast.CompositeLit); isStruct {
        // For struct literals in pointer fields, we emit the struct literal directly
        // The Go compiler will handle the conversion to pointer type
        // This avoids the invalid &StructLiteral{} syntax
    } else {
        // Wrap other values in &
        fieldValue = &goast.UnaryExpr{
            Op: token.AND,
            X:  fieldValue,
        }
    }
}
```

## Debug Evidence

The debug logs show that the transformation process is working correctly up to the point where the field value is emitted. The issue appears to be in how the final Go AST is being constructed or serialized.

Key log entries:

- `Op:17` indicates `token.AND` (the `&` operator)
- The `UnaryExpr` contains a `CompositeLit` (struct literal)
- The pointer field logic correctly identifies this as "already-pointer" and emits it as-is
- But somehow the final Go code contains both `&` and `*` operators

### AST Node Analysis

The problematic AST structure being generated:

```go
&goast.UnaryExpr{
    Op: token.AND,  // &
    X: &goast.UnaryExpr{
        Op: token.MUL,  // *
        X: &goast.CompositeLit{
            Type: goast.NewIdent("User"),
            Elts: []goast.Expr{...},
        },
    },
}
```

This represents `&*User{name: "Alice"}` which is invalid Go syntax.

## Impact

This bug prevents the shape guard example from compiling, which is a critical feature for type safety in Forst. The shape guard functionality is essential for runtime type checking and validation.

## Files Affected

- `forst/internal/transformer/go/expression.go` - Main transformation logic
- `examples/in/rfc/guard/shape_guard.ft` - Test case that fails
- `examples/out/rfc/guard/shape_guard.go` - Generated output with the bug

## Priority

**High** - This is a core functionality bug that prevents shape guards from working, which is a fundamental feature of the Forst language.

## Detailed Fix Instructions

### Step 1: Identify the Source of Double Indirection

The fix requires tracing where the `*` operator is being introduced. Based on the debug logs, the issue is likely in the **recursive shape transformation** where nested shapes are processed.

**Investigation needed:**

1. Add debug logging in `transformShapeNodeWithExpectedType` to show the exact AST structure before and after each transformation step
2. Trace the recursive calls when processing nested shapes
3. Identify where the `*` operator is being added to an already-pointer expression

### Step 2: Fix the Pointer Field Logic

**Current problematic logic (lines ~450-550):**

```go
} else if unaryExpr, isPointer := fieldValue.(*goast.UnaryExpr); isPointer && unaryExpr.Op == token.AND {
    // emit as-is
} else if _, isStruct := fieldValue.(*goast.CompositeLit); isStruct {
    // For struct literals in pointer fields, we emit the struct literal directly
    // The Go compiler will handle the conversion to pointer type
    // This avoids the invalid &StructLiteral{} syntax
}
```

**Proposed fix:**

```go
} else if unaryExpr, isPointer := fieldValue.(*goast.UnaryExpr); isPointer && unaryExpr.Op == token.AND {
    // Check if this is already a pointer to a struct literal
    if innerUnary, isInnerPointer := unaryExpr.X.(*goast.UnaryExpr); isInnerPointer && innerUnary.Op == token.MUL {
        // This is &*StructLiteral - invalid! Remove the inner *
        fieldValue = &goast.UnaryExpr{
            Op: token.AND,
            X:  innerUnary.X, // Remove the * operator
        }
    }
    // emit as-is
} else if _, isStruct := fieldValue.(*goast.CompositeLit); isStruct {
    // For struct literals in pointer fields, wrap with & only
    fieldValue = &goast.UnaryExpr{
        Op: token.AND,
        X:  fieldValue,
    }
}
```

### Step 3: Fix the Recursive Shape Transformation

**File:** `forst/internal/transformer/go/expression.go`
**Function:** `transformShapeNodeWithExpectedType`
**Lines:** ~345-450 (recursive shape processing)

The issue may also be in how nested shapes are processed when the expected type is a pointer. Add specific handling:

```go
if field.Shape != nil {
    // For nested shapes in pointer contexts, ensure we don't double-wrap
    if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer {
        // Process the inner shape without pointer wrapping
        innerValue, err := t.transformShapeNodeWithExpectedType(field.Shape, nil)
        if err != nil {
            return nil, err
        }
        // Wrap with & only if it's a struct literal
        if _, isStruct := innerValue.(*goast.CompositeLit); isStruct {
            fieldValue = &goast.UnaryExpr{
                Op: token.AND,
                X:  innerValue,
            }
        } else {
            fieldValue = innerValue
        }
    } else {
        fieldValue, err = t.transformShapeNodeWithExpectedType(field.Shape, expectedFieldType)
    }
}
```

### Step 4: Add Comprehensive Tests

Create specific test cases in `forst/internal/transformer/go/expression_test.go`:

```go
func TestTransformShapeNodeWithExpectedType_PointerStructLiteral(t *testing.T) {
    // Test case for the specific bug scenario
    shape := &ast.ShapeNode{
        Fields: map[string]ast.ShapeFieldNode{
            "user": ast.ShapeFieldNode{
                Shape: &ast.ShapeNode{
                    Fields: map[string]ast.ShapeFieldNode{
                        "name": ast.ShapeFieldNode{
                            Assertion: &ast.AssertionNode{
                                Constraints: []ast.ConstraintNode{
                                    {Name: "Value", Args: []ast.ConstraintArgumentNode{
                                        {Value: &ast.StringLiteralNode{Value: "Alice"}}
                                    }}
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    expectedType := &ast.TypeNode{
        Ident: ast.TypePointer,
        TypeParams: []ast.TypeNode{{Ident: "User"}},
    }

    result, err := transformer.transformShapeNodeWithExpectedType(shape, expectedType)
    // Assert result is &User{name: "Alice"} not &*User{name: "Alice"}
}
```

### Minimal Unit Test to Reproduce the Issue

**File:** `forst/internal/transformer/go/expression_test.go`

```go
func TestTransformShapeNodeWithExpectedType_PointerStructLiteral_Bug(t *testing.T) {
    // This test reproduces the exact bug from shape_guard.ft
    transformer := &Transformer{
        log: logger.NewLogger("test"),
    }

    // Create the exact AST structure from the failing case
    shape := &ast.ShapeNode{
        Fields: map[string]ast.ShapeFieldNode{
            "user": ast.ShapeFieldNode{
                Shape: &ast.ShapeNode{
                    Fields: map[string]ast.ShapeFieldNode{
                        "name": ast.ShapeFieldNode{
                            Assertion: &ast.AssertionNode{
                                Constraints: []ast.ConstraintNode{
                                    {
                                        Name: "Value",
                                        Args: []ast.ConstraintArgumentNode{
                                            {
                                                Value: &ast.StringLiteralNode{
                                                    Value: "Alice",
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    // Expected type: *User (pointer to User)
    expectedType := &ast.TypeNode{
        Ident: ast.TypePointer,
        TypeParams: []ast.TypeNode{
            {Ident: "User"},
        },
    }

    // Transform the shape
    result, err := transformer.transformShapeNodeWithExpectedType(shape, expectedType)
    if err != nil {
        t.Fatalf("Transform failed: %v", err)
    }

    // Serialize to Go code to check the output
    var buf bytes.Buffer
    err = format.Node(&buf, token.NewFileSet(), result)
    if err != nil {
        t.Fatalf("Failed to format result: %v", err)
    }

    resultStr := buf.String()
    t.Logf("Generated Go code: %s", resultStr)

    // The bug: we should NOT have &*User{name: "Alice"}
    if strings.Contains(resultStr, "&*User") {
        t.Errorf("BUG REPRODUCED: Generated invalid Go syntax &*User{name: \"Alice\"}")
        t.Errorf("Expected: &User{name: \"Alice\"}")
        t.Errorf("Got: %s", resultStr)
    }

    // Verify we have the correct syntax
    if !strings.Contains(resultStr, "&User{") {
        t.Errorf("Missing correct pointer syntax: expected &User{...}")
    }

    // Verify we have the correct field
    if !strings.Contains(resultStr, `name: "Alice"`) {
        t.Errorf("Missing expected field value: name: \"Alice\"")
    }
}

func TestTransformShapeNodeWithExpectedType_PointerStructLiteral_Fixed(t *testing.T) {
    // This test should pass after the fix is implemented
    transformer := &Transformer{
        log: logger.NewLogger("test"),
    }

    // Same setup as the bug test
    shape := &ast.ShapeNode{
        Fields: map[string]ast.ShapeFieldNode{
            "user": ast.ShapeFieldNode{
                Shape: &ast.ShapeNode{
                    Fields: map[string]ast.ShapeFieldNode{
                        "name": ast.ShapeFieldNode{
                            Assertion: &ast.AssertionNode{
                                Constraints: []ast.ConstraintNode{
                                    {
                                        Name: "Value",
                                        Args: []ast.ConstraintArgumentNode{
                                            {
                                                Value: &ast.StringLiteralNode{
                                                    Value: "Alice",
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    expectedType := &ast.TypeNode{
        Ident: ast.TypePointer,
        TypeParams: []ast.TypeNode{
            {Ident: "User"},
        },
    }

    result, err := transformer.transformShapeNodeWithExpectedType(shape, expectedType)
    if err != nil {
        t.Fatalf("Transform failed: %v", err)
    }

    // Verify the AST structure is correct
    unaryExpr, ok := result.(*goast.UnaryExpr)
    if !ok {
        t.Fatalf("Expected UnaryExpr, got %T", result)
    }

    if unaryExpr.Op != token.AND {
        t.Errorf("Expected token.AND (&), got %v", unaryExpr.Op)
    }

    // The inner expression should be a CompositeLit, not another UnaryExpr
    compositeLit, ok := unaryExpr.X.(*goast.CompositeLit)
    if !ok {
        t.Errorf("Expected CompositeLit, got %T", unaryExpr.X)
        // If it's another UnaryExpr, that's the bug!
        if innerUnary, isUnary := unaryExpr.X.(*goast.UnaryExpr); isUnary {
            t.Errorf("BUG: Found nested UnaryExpr with Op: %v", innerUnary.Op)
        }
    }

    // Verify the struct type
    if ident, ok := compositeLit.Type.(*goast.Ident); !ok || ident.Name != "User" {
        t.Errorf("Expected User type, got %v", compositeLit.Type)
    }

    // Verify the field
    if len(compositeLit.Elts) != 1 {
        t.Errorf("Expected 1 field, got %d", len(compositeLit.Elts))
    }

    keyValue, ok := compositeLit.Elts[0].(*goast.KeyValueExpr)
    if !ok {
        t.Errorf("Expected KeyValueExpr, got %T", compositeLit.Elts[0])
    }

    if key, ok := keyValue.Key.(*goast.Ident); !ok || key.Name != "name" {
        t.Errorf("Expected 'name' key, got %v", keyValue.Key)
    }

    if lit, ok := keyValue.Value.(*goast.BasicLit); !ok || lit.Value != `"Alice"` {
        t.Errorf("Expected \"Alice\" value, got %v", keyValue.Value)
    }
}
```

### Step 5: Update Debug Logging

Add more granular logging to track the AST transformation:

```go
// Add to transformShapeNodeWithExpectedType
t.log.WithFields(map[string]interface{}{
    "fieldName": name,
    "fieldValueType": fmt.Sprintf("%T", fieldValue),
    "fieldValue": fmt.Sprintf("%#v", fieldValue),
    "expectedFieldType": expectedFieldType,
    "function": "transformShapeNodeWithExpectedType",
}).Debug("Field value before pointer field logic")

// After pointer field logic
t.log.WithFields(map[string]interface{}{
    "fieldName": name,
    "fieldValueType": fmt.Sprintf("%T", fieldValue),
    "fieldValue": fmt.Sprintf("%#v", fieldValue),
    "function": "transformShapeNodeWithExpectedType",
}).Debug("Field value after pointer field logic")
```

## Reproduction Steps

1. Run `task example:shape-guard`
2. Observe the compilation error
3. Check the generated Go code at line 96 in `examples/out/rfc/guard/shape_guard.go`
4. Add debug logging with `FORST_LOG=debug` to trace the AST transformation

## Additional Notes

The debug logs show that the transformation process is working correctly for most cases, but there's a specific issue with how struct literals are being handled in pointer field contexts. The bug appears to be in the final AST construction or serialization phase rather than in the initial transformation logic.

### Compiler Architecture Impact

This bug reveals a deeper issue in the AST transformation pipeline where:

1. **Type inference** correctly identifies pointer types
2. **Expression transformation** correctly handles most cases
3. **Pointer field logic** has a specific edge case with nested shapes
4. **AST serialization** preserves the problematic structure

The fix requires careful coordination between the type system, expression transformer, and AST builder to ensure consistent pointer handling across all transformation stages.
