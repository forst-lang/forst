---
Feature Name: anonymous-object-literals
Start Date: 2025-01-27
---

# Anonymous Object Literals for Shape Guards

## Summary

Introduce anonymous object literals in Forst that enable structural typing without requiring explicit type definitions. This feature allows developers to create objects that are validated structurally at runtime while maintaining type safety through shape guards. Anonymous object literals provide a TypeScript-like experience for creating objects that match specific structural requirements without nominal typing constraints.

This RFC extends the existing shape guard system to support anonymous object creation and validation, making it easier to work with user input and external data that must conform to specific structural requirements.

## Motivation

Currently, Forst requires explicit type definitions for all objects used in shape guards. This creates friction when working with:

1. **User input validation**: When receiving arbitrary JSON data from users
2. **API integration**: When working with external APIs that return objects with known structure
3. **Dynamic data processing**: When creating objects that must conform to structural requirements
4. **Prototyping**: When quickly creating objects for testing without defining full types

### Status Quo

In the current system, creating an object that matches a shape guard requires defining a named type:

```go
type UserInput = {
  name: String
  email: String
}

func createUser(input UserInput) {
  // Process the input
}
```

This works well for known, stable interfaces but becomes cumbersome when dealing with dynamic data or when the structure is only needed temporarily.

### Desired State

With anonymous object literals, developers can create objects that are validated structurally:

```go
func createUser(input { name: String, email: String }) {
  // input is validated structurally at runtime
  // No explicit type definition required
}

// Usage with anonymous object literal
createUser({
  name: "Alice",
  email: "alice@example.com"
})
```

The object literal is validated against the structural requirements at runtime, providing type safety without nominal typing constraints.

## Definitions

**Anonymous Object Literal**: An object created inline without an explicit type definition, validated structurally against shape guard requirements.

**Structural Validation**: Runtime checking that an object contains the required fields with the correct types, regardless of the object's nominal type.

**Shape Guard Parameter**: A function parameter that accepts anonymous object literals and validates them structurally.

**Runtime Type Safety**: Type checking that occurs at runtime to ensure objects conform to expected structural requirements.

## Guide-level explanation

Anonymous object literals allow you to create objects that are validated structurally when passed to functions with shape guard parameters. This provides a flexible way to work with dynamic data while maintaining type safety.

### Basic Usage

```go
// Function accepts anonymous object literal
func createUser(input { name: String, email: String }) {
  println(input.name)
  println(input.email)
}

// Call with anonymous object literal
createUser({
  name: "Alice",
  email: "alice@example.com"
})
```

The object literal is validated at runtime to ensure it has the required `name` and `email` fields with `String` types.

### Nested Anonymous Objects

Anonymous object literals can contain nested objects:

```go
func processOrder(order {
  customer: { name: String, email: String },
  items: Array({ id: String, quantity: Int })
}) {
  println(order.customer.name)
  for item in order.items {
    println(item.id)
  }
}

// Usage
processOrder({
  customer: {
    name: "Bob",
    email: "bob@example.com"
  },
  items: [
    { id: "item1", quantity: 2 },
    { id: "item2", quantity: 1 }
  ]
})
```

### Shape Guard Integration

Anonymous object literals work seamlessly with existing shape guards:

```go
type MutationArg = Shape

is (m MutationArg) Input(input Shape) {
  ensure m is { input }
}

func createUser(op MutationArg.Input({ name: String, email: String })) {
  // op.input is validated structurally
  println(op.input.name)
}

// Usage
createUser({
  input: {
    name: "Alice",
    email: "alice@example.com"
  }
})
```

### Validation and Error Handling

When an anonymous object literal doesn't match the expected structure, a runtime error is generated:

```go
func createUser(input { name: String, email: String }) {
  // Function body
}

// This would generate a runtime error
createUser({
  name: "Alice"
  // Missing email field
})
```

The error includes details about which fields are missing or have incorrect types.

## Reference-level explanation

Anonymous object literals are implemented as a natural extension of the existing shape guard system. The implementation involves:

### Compilation Process

1. **Parsing**: Anonymous object literals are parsed as `ShapeNode` instances without explicit base types
2. **Type Inference**: The typechecker infers the structural type from the shape guard parameter
3. **Validation Generation**: The transformer generates Go code that validates the object structure at runtime
4. **Error Handling**: Validation failures generate descriptive error messages

### Compile-Time Type Safety

The generated Go code leverages Go's type system for maximum performance:

1. **Concrete Type Generation**: Each anonymous object literal generates a unique Go struct type
2. **Compile-Time Validation**: Go's compiler handles all type checking
3. **Zero Runtime Overhead**: No validation code executed at runtime
4. **Optimized Memory Layout**: Direct struct access without interface overhead

```go
// Forst: func createUser(input { name: String, email: String })
// Generated Go:
type T_abc123 struct {
    name  string
    email string
}

func createUser(input T_abc123) {
    // Direct field access - no runtime validation needed
    fmt.Println(input.name)
    fmt.Println(input.email)
}

// Usage with compile-time type checking:
createUser(T_abc123{
    name:  "Alice",
    email: "alice@example.com",
})
```

### Type Safety

Anonymous object literals maintain type safety through:

1. **Compile-time shape inference**: The typechecker validates that object literals match expected shapes
2. **Go's type system**: Generated concrete types ensure compile-time type safety
3. **Field access safety**: Direct struct field access with compile-time validation

### Performance Characteristics

The implementation achieves maximum performance:

1. **Zero runtime overhead**: No validation code executed at runtime
2. **Compile-time optimization**: Go's compiler optimizes all type checking
3. **Direct memory access**: No interface or reflection overhead
4. **Type system leverage**: Uses Go's native type system for all validation

## Drawbacks

1. **Runtime overhead**: Additional validation code adds small performance cost
2. **Error messages**: Runtime errors for structural mismatches may be less clear than compile-time errors
3. **Debugging complexity**: Runtime validation errors can be harder to debug than compile-time type errors
4. **IDE support**: Anonymous object literals may be harder for IDEs to provide autocomplete for

## Rationale and alternatives

### Alternative Approaches

1. **Interface-based validation**: Use Go interfaces for structural validation

   - Drawback: Less type safety, more runtime overhead
   - Drawback: Harder to provide detailed error messages

2. **Code generation**: Generate validation code from type definitions

   - Drawback: Requires explicit type definitions
   - Drawback: More complex build process

3. **Reflection-based validation**: Use Go reflection for runtime validation
   - Drawback: Significant performance overhead
   - Drawback: Less type safety

### Why Anonymous Object Literals

Anonymous object literals provide the best balance of:

1. **Type safety**: Compile-time shape inference with runtime validation
2. **Performance**: Minimal overhead with direct field access
3. **Developer experience**: Familiar syntax similar to TypeScript
4. **Integration**: Seamless integration with existing shape guard system

### Prior Art

This feature is inspired by:

1. **TypeScript object literals**: Similar syntax and structural validation
2. **Go struct literals**: Familiar syntax for Go developers
3. **JSON Schema validation**: Runtime validation of structural requirements
4. **Protocol Buffers**: Structural validation of message formats

## Implementation Challenges and Solutions

The implementation of anonymous object literals faces several significant challenges due to the current architecture of Forst's type system and compilation pipeline. This section analyzes the key difficulties and proposes solutions.

### Challenge 1: Parameter Type Parsing Ambiguity

**Problem**: The current parser has difficulty distinguishing between anonymous object literals and other syntax constructs in function parameters.

**Current State**: In `parser/function.go`, the `parseSimpleParameter()` function handles shape types but doesn't have a clear path for anonymous object literals:

```go
if tok.Type == ast.TokenLBrace {
    shape := p.parseShape(nil)
    baseType := ast.TypeIdent(ast.TypeShape)
    return ast.SimpleParamNode{
        Ident: ast.Ident{ID: ast.Identifier(name)},
        Type: ast.TypeNode{
            Ident: ast.TypeShape,
            Assertion: &ast.AssertionNode{
                BaseType: &baseType,
                Constraints: []ast.ConstraintNode{{
                    Name: "Match",
                    Args: []ast.ConstraintArgumentNode{{
                        Shape: &shape,
                    }},
                }},
            },
        },
    }
}
```

**Impact**: This creates parsing conflicts between:

- Named shape types: `func f(x MyShape)`
- Anonymous object literals: `func f(x { name: String })`
- Shape guard parameters: `func f(x Shape)`

**Solution**: Implement a new parameter type `AnonymousShapeParamNode` that explicitly represents anonymous object literals:

```go
type AnonymousShapeParamNode struct {
    SimpleParamNode
    Shape ast.ShapeNode
}

func (p *Parser) parseAnonymousShapeParameter() ast.ParamNode {
    name := p.expect(ast.TokenIdentifier).Value
    p.expect(ast.TokenLBrace)
    shape := p.parseShape(nil)
    return ast.AnonymousShapeParamNode{
        Ident: ast.Ident{ID: ast.Identifier(name)},
        Shape: shape,
        Type: ast.TypeNode{
            Ident: ast.TypeShape,
            Assertion: &ast.AssertionNode{
                BaseType: &ast.TypeIdent(ast.TypeShape),
                Constraints: []ast.ConstraintNode{{
                    Name: "Anonymous",
                    Args: []ast.ConstraintArgumentNode{{
                        Shape: &shape,
                    }},
                }},
            },
        },
    }
}
```

### Challenge 2: Type Inference Complexity

**Problem**: The typechecker's inference system is designed for named types and shape guards, not anonymous structural validation.

**Current State**: In `typechecker/infer_assertion.go`, type inference relies on hash-based type identification:

```go
hash, err := tc.Hasher.HashNode(assertion)
if err != nil {
    return nil, fmt.Errorf("failed to hash assertion during inferAssertionType: %s", err)
}
typeIdent := hash.ToTypeIdent()
```

**Impact**: Anonymous object literals would generate different hash-based types for each usage, making type compatibility checking difficult.

**Solution**: Implement structural type compatibility checking that ignores hash-based names:

```go
func (tc *TypeChecker) isStructurallyCompatible(expected ast.ShapeNode, actual ast.ShapeNode) bool {
    // Check field names and types recursively
    for fieldName, expectedField := range expected.Fields {
        actualField, exists := actual.Fields[fieldName]
        if !exists {
            return false
        }
        if !tc.isFieldCompatible(expectedField, actualField) {
            return false
        }
    }
    return true
}

func (tc *TypeChecker) isFieldCompatible(expected ast.ShapeFieldNode, actual ast.ShapeFieldNode) bool {
    // Compare field types structurally, not by hash
    if expected.Type != nil && actual.Type != nil {
        return tc.IsTypeCompatible(*expected.Type, *actual.Type)
    }
    if expected.Shape != nil && actual.Shape != nil {
        return tc.isStructurallyCompatible(*expected.Shape, *actual.Shape)
    }
    return false
}
```

### Challenge 3: Runtime Validation Code Generation

**Problem**: The transformer currently generates Go code for named types but lacks infrastructure for runtime structural validation.

**Current State**: In `transformer/go/expression.go`, shape transformation focuses on type compatibility:

```go
func (t *Transformer) transformShapeNodeWithExpectedType(shape *ast.ShapeNode, expectedTypeName string) (goast.Expr, error) {
    // ... existing code for named types
    if structType == nil {
        // Fallback: use anonymous struct type
        var err error
        structTypePtr, err := t.transformShapeType(shape)
        if err != nil {
            return nil, err
        }
        structType = *structTypePtr
    }
    // ... rest of function
}
```

**Impact**: No validation code is generated for structural mismatches, leading to runtime panics instead of descriptive errors.

**Solution**: Implement validation code generation for anonymous object literals:

```go
func (t *Transformer) generateAnonymousObjectValidation(shape *ast.ShapeNode, paramName string) ([]goast.Stmt, error) {
    var validationStmts []goast.Stmt

    for fieldName, field := range shape.Fields {
        // Generate field existence check
        existenceCheck := &goast.IfStmt{
            Cond: &goast.BinaryExpr{
                X:  &goast.SelectorExpr{X: goast.NewIdent(paramName), Sel: goast.NewIdent(fieldName)},
                Op: token.EQL,
                Y:  goast.NewIdent("nil"),
            },
            Body: &goast.BlockStmt{
                List: []goast.Stmt{
                    &goast.ReturnStmt{
                        Results: []goast.Expr{
                            &goast.CallExpr{
                                Fun: goast.NewIdent("fmt.Errorf"),
                                Args: []goast.Expr{
                                    &goast.BasicLit{
                                        Kind:  gotoken.STRING,
                                        Value: fmt.Sprintf("\"missing required field '%s'\"", fieldName),
                                    },
                                },
                            },
                        },
                    },
                },
            },
        }
        validationStmts = append(validationStmts, existenceCheck)

        // Generate type validation for the field
        if field.Type != nil {
            typeCheck := t.generateFieldTypeValidation(paramName, fieldName, field.Type)
            validationStmts = append(validationStmts, typeCheck)
        }
    }

    return validationStmts, nil
}
```

### Challenge 4: Error Handling Integration

**Problem**: The current error handling system in `transformer/go/ensure_constraint.go` is designed for built-in constraints, not structural validation.

**Current State**: Error handling is constraint-specific:

```go
func (at *AssertionTransformer) TransformBuiltinConstraint(typeIdent ast.TypeIdent, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
    handlerMap, ok := builtinConstraints[typeIdent]
    if !ok {
        return nil, fmt.Errorf("unknown typeIdent %s for built-in constraints: %s", typeIdent, constraint.Name)
    }
    // ... rest of function
}
```

**Impact**: Anonymous object literal validation errors would not integrate with the existing error handling system.

**Solution**: Extend the error handling system to support structural validation:

```go
type StructuralValidationError struct {
    FieldName string
    Expected  string
    Actual    string
    Message   string
}

func (e StructuralValidationError) Error() string {
    return fmt.Sprintf("field '%s': expected %s, got %s: %s",
        e.FieldName, e.Expected, e.Actual, e.Message)
}

func (t *Transformer) generateStructuralValidationError(fieldName, expected, actual string) goast.Expr {
    return &goast.CallExpr{
        Fun: goast.NewIdent("fmt.Errorf"),
        Args: []goast.Expr{
            &goast.BasicLit{
                Kind:  gotoken.STRING,
                Value: fmt.Sprintf("\"field '%s': expected %s, got %%T\"", fieldName, expected),
            },
            &goast.SelectorExpr{
                X:   goast.NewIdent("input"),
                Sel: goast.NewIdent(fieldName),
            },
        },
    }
}
```

### Challenge 5: Runtime Performance Optimization

**Problem**: While we can generate any number of Go types, we need to minimize runtime overhead while preserving Forst's type safety features.

**Current State**: The transformer generates minimal validation code, relying on Go's type system for safety.

**Impact**: Anonymous object literals could generate expensive runtime validation code for each function parameter.

**Solution**: Implement aggressive compile-time optimization strategies:

1. **Leverage Go's Type System**: Generate concrete Go types that use Go's compile-time type checking
2. **Zero-Cost Abstractions**: Use Go's type system for validation where possible
3. **Minimal Runtime Checks**: Only generate runtime validation for cases that Go's type system cannot handle

```go
func (t *Transformer) generateOptimizedAnonymousObjectType(shape *ast.ShapeNode) (*goast.TypeSpec, error) {
    // Generate a concrete Go struct type
    structType, err := t.transformShapeType(shape)
    if err != nil {
        return nil, err
    }

    // Create a unique type name for this shape
    hash, err := t.TypeChecker.Hasher.HashNode(shape)
    if err != nil {
        return nil, err
    }
    typeName := hash.ToTypeIdent()

    return &goast.TypeSpec{
        Name: goast.NewIdent(string(typeName)),
        Type: *structType,
    }, nil
}

func (t *Transformer) generateFunctionWithAnonymousObject(shape *ast.ShapeNode, funcName string) (*goast.FuncDecl, error) {
    // Generate the type first
    typeSpec, err := t.generateOptimizedAnonymousObjectType(shape)
    if err != nil {
        return nil, err
    }

    // Generate function that uses the concrete type
    return &goast.FuncDecl{
        Name: goast.NewIdent(funcName),
        Type: &goast.FuncType{
            Params: &goast.FieldList{
                List: []*goast.Field{
                    {
                        Names: []*goast.Ident{goast.NewIdent("input")},
                        Type:  goast.NewIdent(typeSpec.Name.Name),
                    },
                },
            },
        },
        Body: &goast.BlockStmt{
            // Function body - no runtime validation needed
            // Go's type system ensures type safety
        },
    }, nil
}
```

**Key Optimization Strategies:**

1. **Concrete Type Generation**: Instead of `interface{}` or reflection, generate concrete Go struct types
2. **Compile-Time Type Safety**: Let Go's compiler handle type checking
3. **No Runtime Validation**: Eliminate runtime validation code for Go-compatible types
4. **Type Aliasing**: Use Go's type system to ensure structural compatibility

```go
// Forst: func createUser(input { name: String, email: String })
// Generated Go:
type T_abc123 struct {
    name  string
    email string
}

func createUser(input T_abc123) {
    // No runtime validation needed - Go's type system ensures safety
    fmt.Println(input.name)
    fmt.Println(input.email)
}

// Usage generates compile-time type checking:
createUser(T_abc123{
    name:  "Alice",
    email: "alice@example.com",
})
```

**Performance Benefits:**

- **Zero Runtime Overhead**: No validation code executed at runtime
- **Compile-Time Safety**: Go's type system catches all type errors
- **Optimized Code Generation**: Generated code is as fast as hand-written Go
- **Memory Efficiency**: No reflection or interface overhead

## Implementation Plan

### Phase 1: Basic Support

1. Extend parser to support anonymous object literals in function parameters
2. Update typechecker to infer structural types from shape guard parameters
3. Modify transformer to generate validation code for anonymous objects
4. Add basic error handling for structural mismatches

### Phase 2: Advanced Features

1. Support nested anonymous objects
2. Add support for optional fields
3. Implement union types in anonymous objects
4. Add support for generic anonymous objects

### Phase 3: Optimization

1. Optimize validation code generation
2. Add compile-time validation where possible
3. Implement caching for validation functions
4. Add performance benchmarks

## Examples

### User Input Validation

```go
func handleUserRegistration(input {
  username: String,
  email: String,
  password: String,
  preferences: { theme: String, notifications: Bool }
}) {
  // input is validated structurally at runtime
  // No need to define UserRegistrationData type
}
```

### API Response Processing

```go
func processApiResponse(response {
  status: String,
  data: { id: String, name: String },
  metadata: { timestamp: String, version: String }
}) {
  // response is validated against expected API structure
}
```

### Configuration Objects

```go
func configureApp(config {
  port: Int,
  database: { host: String, port: Int, name: String },
  logging: { level: String, file: String }
}) {
  // config is validated structurally
}
```

## Conclusion

Anonymous object literals provide a powerful and flexible way to work with dynamic data while maintaining type safety. They integrate seamlessly with Forst's existing shape guard system and provide a familiar developer experience similar to TypeScript.

This feature addresses real-world use cases for user input validation, API integration, and dynamic data processing while maintaining Forst's focus on type safety and Go interoperability.
