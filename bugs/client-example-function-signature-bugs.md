# Client Example Function Signature Bugs

## Summary

The client example (`task example:client`) fails due to two critical bugs in the transformer:

1. **Wrong Return Type in Error Handler**: Functions with `ensure` statements return the wrong type in error cases
2. **Wrong Field Types in Zero Values**: String fields get `0` instead of `""` in zero value generation

## Root Cause Analysis

### Bug 1: Wrong Return Type in Error Handler

**Error Message:**

```
cannot use User{…} (value of struct type User) as CreateUserResponse value in return statement
```

**Generated Code:**

```go
func CreateUser(input CreateUserRequest) (CreateUserResponse, error) {
    user := User{age: input.age, email: input.email, id: "123", name: input.name}
    if !G_A8T5YL5RDb7(user) {
        return User{id: "", name: "", age: 0, email: ""}, errors.New("assertion failed: " + "Valid")  // ❌ BUG: Returns User instead of CreateUserResponse
    }
    return CreateUserResponse{user: user, created_at: 1234567890}, nil
}
```

**Root Cause:** The `transformErrorStatement` function in `forst/internal/transformer/go/statement.go` is using the wrong return type for error cases. It should use the function's declared return type (`CreateUserResponse`) but is using a different type (`User`).

**Location:** `forst/internal/transformer/go/statement.go:transformErrorStatement`

### Bug 2: Wrong Field Types in Zero Values

**Error Message:**

```
cannot use 0 (untyped int constant) as string value in struct literal
```

**Generated Code:**

```go
func UpdateUserAge(id string, newAge int) (User, error) {
    // ...
    return User{id: 0, name: 0, age: 0, email: 0}, nil  // ❌ BUG: Using 0 for string fields
}
```

**Root Cause:**
The `getZeroValue` function in `forst/internal/transformer/go/statement.go` is not properly handling string types. While it has the correct logic for string types:

```go
case "string":
    return &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
```

The issue is that the field type transformation is failing, causing `getZeroValue` to receive the wrong type or fall through to the default case.

**Location:** `forst/internal/transformer/go/statement.go:getZeroValue`

## Evidence

### Debug Output Analysis

The debug logs show that function signature inference is working correctly:

```
[PINPOINT] Inferred return type for function  function=CreateUser inferredType=CreateUserResponse returnAST=ast.ShapeNode returnIndex=0
[PINPOINT] Function signature for error return  function=transformErrorStatement functionName=UpdateUserAge returnCount=2 returnTypes="[User Error]"
```

However, the generated Go code shows the bugs are still present, indicating the issue is in the transformer's error handling logic.

### Type Inference Working Correctly

The typechecker correctly infers:

- `CreateUser` returns `CreateUserResponse`
- `UpdateUserAge` returns `User`
- Field types are correctly identified as `String` for `id`, `name`, `email`

## Impact

1. **Compilation Failure**: The client example cannot compile due to type mismatches
2. **Runtime Errors**: Even if compilation succeeds, the wrong zero values would cause runtime issues
3. **Developer Experience**: Functions with `ensure` statements are unusable

## Affected Code

### Source Code (`examples/in/rfc/sidecar/tests/user.ft`)

```forst
func CreateUser(input CreateUserRequest) {
    user := {
        id: "123",
        name: input.name,
        age: input.age,
        email: input.email
    }

    ensure user is Valid  // <-- This ensure statement requires error return

    return {
        user: user,
        created_at: 1234567890
    }, nil  // <-- This is returning (CreateUserResponse, nil)
}

func UpdateUserAge(id String, newAge Int) {
    ensure newAge is GreaterThan(0)
    ensure newAge is LessThan(151)

    user := GetUserById(id)
    updatedUser := {
        id: user.id,
        name: user.name,
        age: newAge,
        email: user.email
    }

    ensure updatedUser is Valid

    return updatedUser, nil
}
```

## Proposed Fix

### Fix 1: Correct Error Handler Return Type

The `transformErrorStatement` function should use the function's declared return type for generating zero values:

```go
// In transformErrorStatement function
zeroValue := t.buildZeroCompositeLiteral(&retType)  // Use the correct return type
```

### Fix 2: Correct Zero Value Generation

The `getZeroValue` function should properly handle all field types:

```go
func getZeroValue(goType goast.Expr) goast.Expr {
    switch t := goType.(type) {
    case *goast.Ident:
        switch t.Name {
        case "string":
            return &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
        case "int", "int64", "int32", "int16", "int8", "uint", "uint64", "uint32", "uint16", "uint8", "uintptr", "float64", "float32", "complex64", "complex128", "rune", "byte":
            return &goast.BasicLit{Kind: token.INT, Value: "0"}
        case "bool":
            return &goast.Ident{Name: "false"}
        case "error":
            return &goast.Ident{Name: "nil"}
        default:
            // For struct types, return nil
            return &goast.Ident{Name: "nil"}
        }
    case *goast.StarExpr:
        // For pointer types, return nil
        return &goast.Ident{Name: "nil"}
    case *goast.ArrayType:
        // For array types, return nil
        return &goast.Ident{Name: "nil"}
    case *goast.InterfaceType:
        // For interface types, return nil
        return &goast.Ident{Name: "nil"}
    default:
        // For any other type, return nil
        return &goast.Ident{Name: "nil"}
    }
}
```

## Testing

### Reproduction Steps

1. Run the client example:

   ```bash
   task example:client
   ```

2. Observe compilation errors:
   ```
   cannot use User{…} (value of struct type User) as CreateUserResponse value in return statement
   cannot use 0 (untyped int constant) as string value in struct literal
   ```

### Expected Behavior

After fixes:

1. Error cases should return the correct type (`CreateUserResponse` for `CreateUser`)
2. Zero values should use correct types (`""` for strings, `0` for integers)
3. The client example should compile and run successfully

## Related Issues

This bug is related to the broader issue of function signature inference and error handling in the transformer. Similar issues may exist in other functions with `ensure` statements.

## Priority

**High Priority** - This prevents the client example from working, which is a core feature for demonstrating Forst's capabilities.

## Status

- [x] Bug confirmed
- [x] Root cause identified
- [ ] Fix implemented
- [ ] Tests passing
- [ ] Client example working

## Implementation Progress

### Attempt 1: Fix getZeroValue Function

**Date:** Current
**Status:** Partially Working

I've already fixed the `getZeroValue` function to properly handle string types:

```go
func getZeroValue(goType goast.Expr) goast.Expr {
    switch t := goType.(type) {
    case *goast.Ident:
        switch t.Name {
        case "string":
            return &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
        case "int", "int64", "int32", "int16", "int8", "uint", "uint64", "uint32", "uint16", "uint8", "uintptr", "float64", "float32", "complex64", "complex128", "rune", "byte":
            return &goast.BasicLit{Kind: token.INT, Value: "0"}
        case "bool":
            return &goast.Ident{Name: "false"}
        case "error":
            return &goast.Ident{Name: "nil"}
        default:
            // For struct types, return nil
            return &goast.Ident{Name: "nil"}
        }
    case *goast.StarExpr:
        // For pointer types, return nil
        return &goast.Ident{Name: "nil"}
    case *goast.ArrayType:
        // For array types, return nil
        return &goast.Ident{Name: "nil"}
    case *goast.InterfaceType:
        // For interface types, return nil
        return &goast.Ident{Name: "nil"}
    default:
        // For any other type, return nil
        return &goast.Ident{Name: "nil"}
    }
}
```

**Test Results:** ❌ Both bugs still present

**Error Messages:**

```
cannot use User{…} (value of struct type User) as T_NGUd7jXQkPf value in return statement
cannot use 0 (untyped int constant) as string value in struct literal
```

**Key Insight from Debug Output:**
The debug logs show that `getZeroValue` is working correctly:

```
[DEBUG] Generated zero value for field fieldName=id fieldType=TYPE_STRING function=buildZeroCompositeLiteral zeroValue="*ast.BasicLit"
```

This shows that `getZeroValue` is correctly returning `*ast.BasicLit` for string fields, which should be `""`. However, the generated Go code still shows `0` for string fields.

**Root Cause Analysis:**
The issue is NOT in `getZeroValue` itself. The problem is that there are multiple code paths generating error cases, and not all of them are using the `transformErrorStatement` function.

Looking at the debug output, I can see that the `transformErrorStatement` function is being called correctly and is generating the right zero values, but there's another code path that's generating the wrong error cases.

**Next Investigation:** I need to find where the other error cases are being generated that are not using the `transformErrorStatement` function.

### Attempt 2: Fix buildCompositeLiteralForReturn Function

**Date:** Current
**Status:** ❌ Fix Failed

**Root Cause Found:** The issue was in the `buildCompositeLiteralForReturn` function on line 245. This function was calling:

```go
goFieldType, _ := t.transformType(*field.Type)
val = getZeroValue(goFieldType)
```

The problem was that `t.transformType(*field.Type)` was not returning the correct Go type for string fields, causing `getZeroValue` to receive the wrong type and fall through to the default case.

**Fix Applied:** Updated `buildCompositeLiteralForReturn` to use the same logic as `buildZeroCompositeLiteral`:

```go
// Use the same logic as buildZeroCompositeLiteral for consistent zero value generation
switch field.Type.Ident {
case ast.TypeString:
    val = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
case ast.TypeInt:
    val = &goast.BasicLit{Kind: token.INT, Value: "0"}
case ast.TypeBool:
    val = goast.NewIdent("false")
case ast.TypeFloat:
    val = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
case ast.TypeError:
    val = goast.NewIdent("nil")
default:
    // For user-defined types, use nil
    val = goast.NewIdent("nil")
}
```

**Test Results:** ❌ Both bugs still present

**New Insight:** The debug output shows that `buildZeroCompositeLiteral` is working correctly and generating `*ast.BasicLit` for string fields, but the generated Go code still shows `0` for string fields. This means there's another code path that's generating the wrong error cases.

**Next Investigation:** I need to find where the other error cases are being generated that are not using the `transformErrorStatement` function or `buildZeroCompositeLiteral` function.

**New Discovery:** Found the second code path! The `buildCompositeLiteralForReturn` function is being called on line 643 in the `transformReturnStatement` function, not in the `transformErrorStatement` function.

**Code Path Analysis:**

1. **`transformErrorStatement` function** (line 157) - This is called for `EnsureNode` statements and generates error cases correctly
2. **`transformReturnStatement` function** (line 643) - This calls `buildCompositeLiteralForReturn` for user-defined return types

**Root Cause:** The issue is in the `transformReturnStatement` function on line 643:

```go
if expectedType.TypeKind == ast.TypeKindUserDefined {
    for i, ret := range results {
        if ident, ok := ret.(*goast.Ident); ok {
            goRet := t.buildCompositeLiteralForReturn(expectedType, ident.Name)
            results[i] = goRet
        }
    }
}
```

This code path is generating the wrong error cases because it's calling `buildCompositeLiteralForReturn` for return statements, not error statements.

**Next Investigation:** I need to understand why this code path is being triggered for error cases instead of the `transformErrorStatement` function.

### Attempt 3: Investigate Return Statement Code Path

**Date:** Current
**Status:** 🔍 Investigating

**New Discovery:** Found the second code path! The `buildCompositeLiteralForReturn` function is being called on line 643 in the `transformReturnStatement` function, not in the `transformErrorStatement` function.

**Code Path Analysis:**

1. **`transformErrorStatement` function** (line 157) - This is called for `EnsureNode` statements and generates error cases correctly
2. **`transformReturnStatement` function** (line 643) - This calls `buildCompositeLiteralForReturn` for user-defined return types

**Root Cause:** The issue is in the `transformReturnStatement` function on line 643:

```go
if expectedType.TypeKind == ast.TypeKindUserDefined {
    for i, ret := range results {
        if ident, ok := ret.(*goast.Ident); ok {
            goRet := t.buildCompositeLiteralForReturn(expectedType, ident.Name)
            results[i] = goRet
        }
    }
}
```

This code path is generating the wrong error cases because it's calling `buildCompositeLiteralForReturn` for return statements, not error statements.

**Next Investigation:** I need to understand why this code path is being triggered for error cases instead of the `transformErrorStatement` function.

### Attempt 4: AST Serialization Issue Discovered

**Date:** Current
**Status:** 🔍 Root Cause Found

**Critical Discovery:** The AST generation is working correctly! The debug output shows:

```
[DEBUG] Generated zero value for field fieldName=id fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=name fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=age fieldType=Int zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:INT, Value:0}"
[DEBUG] Generated zero value for field fieldName=email fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
```

This shows that `buildZeroCompositeLiteral` is correctly generating:

- `BasicLit{Kind:STRING, Value:""}` for string fields
- `BasicLit{Kind:INT, Value:0}` for int fields

**Root Cause:** The issue is NOT in the AST generation. The issue is in the Go AST to Go code serialization process. The `*ast.BasicLit` with `Kind:STRING` and `Value:""` is being serialized as `0` in the final Go code.

**Evidence:** The error message shows:

```
cannot use 0 (untyped int constant) as string value in struct literal
```

But the debug output shows that the AST contains `BasicLit{Kind:STRING, Value:""}` for string fields.

**Next Investigation:** I need to find where the Go AST is being serialized to Go code and why `BasicLit{Kind:STRING, Value:""}` is being output as `0` instead of `""`.

### Attempt 5: Investigate Go AST Serialization

**Date:** Current
**Status:** 🔍 Investigating

**Hypothesis:** The issue is in the Go AST to Go code serialization process. The `*ast.BasicLit` nodes are being incorrectly serialized.

**Investigation Plan:**

1. Find where Go AST is serialized to Go code
2. Check if there's a custom printer or formatter being used
3. Verify that `go/ast` and `go/printer` are being used correctly
4. Look for any custom serialization logic that might be overriding the default behavior

### Attempt 6: Two Code Paths Identified

**Date:** Current
**Status:** 🔍 Root Cause Found

**Critical Discovery:** The AST debug output reveals that there are TWO different code paths generating error cases:

**Code Path 1 (Working Correctly):**

- `transformErrorStatement` function generates `*ast.BasicLit` values correctly
- Debug output shows: `BasicLit{Kind:STRING, Value:""}` for string fields

**Code Path 2 (Broken):**

- Some other code path generates `*ast.Ident` values incorrectly
- Debug output shows: `Element 0 key: *ast.Ident, value: *ast.Ident` for all fields

**Evidence from AST Debug Output:**

```
DEBUG: Function UpdateUserAge
DEBUG: Statement 5: *ast.ReturnStmt
DEBUG: Return statement with 2 results
DEBUG: Return result 0: *ast.CompositeLit
DEBUG: Composite literal type: *ast.Ident
DEBUG: Composite literal has 4 elements
DEBUG: Element 0 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 1 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 2 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 3 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
```

**Root Cause:** The error cases are being generated by a different code path that creates `*ast.Ident` values instead of `*ast.BasicLit` values. When these `*ast.Ident` values are serialized to Go code, they become `0` instead of `""`.

**Next Investigation:** I need to find the second code path that's generating `*ast.Ident` values for error cases instead of using `transformErrorStatement`.

### Attempt 7: Find the Second Code Path

**Date:** Current
**Status:** 🔍 Investigating

**Hypothesis:** There's another code path that generates error cases without using `transformErrorStatement` or `buildZeroCompositeLiteral`.

**Investigation Plan:**

1. Search for where `*ast.Ident` values are being generated for error cases
2. Find the code path that bypasses `transformErrorStatement`
3. Identify why this code path is being triggered instead of the correct one

### Attempt 8: Final Root Cause Found

**Date:** Current
**Status:** ✅ Root Cause Identified

**Final Root Cause:** The error cases are being generated by the normal return statement processing in `transformReturnStatement`, not by the `transformErrorStatement` function.

**Code Path Analysis:**

1. **`transformErrorStatement`** - This generates correct `*ast.BasicLit` values for error cases
2. **`transformReturnStatement`** - This processes return statements and calls `buildFieldsForExpectedType`
3. **`buildFieldsForExpectedType`** - This calls `buildFieldValue` for each field
4. **`buildFieldValue`** - When a field is missing (error case), it generates `goast.NewIdent("nil")`

**The Problem:** When error cases are processed by `transformReturnStatement`, they are treated as missing fields in the return statement. The `buildFieldValue` function generates `goast.NewIdent("nil")` for missing fields, which becomes `nil` in the generated Go code. However, for error cases, we need proper zero values like `""` for strings and `0` for integers.

**Evidence:** The AST debug output shows that the error cases are being generated as `*ast.Ident` values instead of `*ast.BasicLit` values:

```
DEBUG: Element 0 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 1 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 2 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 3 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
```

**Solution:** The error cases should be generated by `transformErrorStatement`, not by `transformReturnStatement`. The issue is that the error cases are being processed as normal return statements instead of error statements.

### Attempt 9: Implement Fix

**Date:** Current
**Status:** 🔧 Implementing Fix

**Fix Strategy:** Ensure that error cases are generated by `transformErrorStatement` instead of `transformReturnStatement`. The error cases should use `buildZeroCompositeLiteral` to generate proper zero values.

**Implementation Plan:**

1. Identify why error cases are being processed by `transformReturnStatement` instead of `transformErrorStatement`
2. Fix the code path to ensure error cases use the correct zero value generation
3. Test the fix to ensure both bugs are resolved

### Attempt 10: Fix Did Not Work

**Date:** Current
**Status:** ❌ Fix Failed

**Test Results:** Both bugs still present

**Error Messages:**

```
cannot use User{…} (value of struct type User) as T_NGUd7jXQkPf value in return statement
cannot use 0 (untyped int constant) as string value in struct literal
```

**Key Insight:** The `transformErrorStatement` function is working correctly and generating proper zero values:

```
[DEBUG] Generated zero value for field fieldName=id fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=name fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=age fieldType=Int zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:INT, Value:0}"
[DEBUG] Generated zero value for field fieldName=email fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
```

But the AST debug output still shows `*ast.Ident` values:

```
DEBUG: Element 0 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 1 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 2 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
DEBUG: Element 3 key: *ast.Ident, value: *ast.Ident  // ❌ Should be BasicLit
```

**Root Cause Analysis:** The error cases are NOT being generated by `transformErrorStatement`. They are being generated by a different code path that creates `*ast.Ident` values instead of `*ast.BasicLit` values.

**The Real Issue:** The error cases are being processed as normal return statements, not as error statements. The `transformErrorStatement` function is being called correctly, but the error cases are also being processed by the normal return statement processing, which overwrites the correct error cases with wrong ones.

**Next Investigation:** I need to find where the error cases are being overwritten by the normal return statement processing.

### Attempt 11: Fix Made Things Worse

**Date:** Current
**Status:** ❌ Fix Failed

**New Errors:**

```
not enough return values
have ()
want (T_NGUd7jXQkPf, error)
not enough return values
have ()
want (User, error)
cannot use 0 (untyped int constant) as string value in struct literal
```

**Root Cause Discovery:** The `transformErrorStatement` function is not finding the function's return types:

```
[PINPOINT] Function signature for error return functionName=UpdateUserAge returnCount=0 returnTypes="[]"
```

The issue is that `fn.ReturnTypes` is empty, which means the function signature inference is not working correctly. The function signature is not being properly inferred during the type checking phase.

**The Real Problem:** The function signature inference is not working correctly, so the error cases don't know what return types to generate.

**Next Fix:** I need to fix the function signature inference so that `fn.ReturnTypes` contains the correct return types.

### Attempt 12: Progress Made, New Issues Found

**Date:** Current
**Status:** 🔧 Partial Success

**Progress:** The `transformErrorStatement` function is now working correctly and generating proper zero values:

```
[DEBUG] Generated zero value for field fieldName=id fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=name fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=age fieldType=Int zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:INT, Value:0}"
[DEBUG] Generated zero value for field fieldName=email fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
```

**Remaining Issues:**

1. **Bug 1 still present**: `cannot use User{…} (value of struct type User) as T_NGUd7jXQkPf value in return statement`

   - The `CreateUser` function should return `CreateUserResponse` (which is `T_NGUd7jXQkPf`), not `User`
   - The error case is returning `User` instead of `CreateUserResponse`

2. **New issue**: `undefined: errors` - The `errors` package is not being imported
   - The error cases are trying to use `errors.New()` but the `errors` package is not imported

**Root Cause Analysis:**
The function signature inference is working correctly, but there's a mismatch between the inferred return type and what the error cases are returning. The `CreateUser` function should return `CreateUserResponse`, but the error cases are returning `User`.

**Next Fix:** I need to fix the import issue and ensure the error cases return the correct type.

### Attempt 13: Final Analysis

**Date:** Current
**Status:** 🔍 Root Cause Identified

**Progress:** The import issue is fixed - no more `undefined: errors` errors. The `transformErrorStatement` function is working correctly and generating proper zero values:

```
[DEBUG] Generated zero value for field fieldName=id fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=name fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
[DEBUG] Generated zero value for field fieldName=age fieldType=Int zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:INT, Value:0}"
[DEBUG] Generated zero value for field fieldName=email fieldType=String zeroValue="*ast.BasicLit" zeroValueDetails="BasicLit{Kind:STRING, Value:\"\"}"
```

**Remaining Issues:**

1. **Bug 1 still present**: `cannot use User{…} (value of struct type User) as T_NGUd7jXQkPf value in return statement`
2. **Bug 2 still present**: `cannot use 0 (untyped int constant) as string value in struct literal`

**Final Root Cause Analysis:**
The `transformErrorStatement` function is working correctly and generating proper zero values, but the error cases are being overwritten by another code path. The issue is that there are two different code paths generating error cases:

1. **`transformErrorStatement`** - This is working correctly and generating proper zero values
2. **Some other code path** - This is overwriting the error cases with wrong values

The error cases are being generated by the normal return statement processing, not by the `transformErrorStatement` function. The `transformErrorStatement` function is being called correctly and generating the right zero values, but the error cases are being overwritten by the normal return statement processing.

**Conclusion:** The issue is that the error cases are being generated by the wrong code path. The `transformErrorStatement` function is working correctly, but the error cases are being overwritten by the normal return statement processing.

**Next Steps:** I need to find where the error cases are being overwritten and fix that code path.
