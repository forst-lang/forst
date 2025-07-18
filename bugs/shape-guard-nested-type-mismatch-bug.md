# Shape Guard Nested Type Mismatch Bug

## Summary

When compiling the shape guard example, the Forst compiler generates Go code that uses hash-based types for nested shapes instead of the expected named types. This occurs specifically when transforming nested shape literals that should use named types like `AppContext` and `User`, but instead get hash-based types like `T_LYQafBLM8TQ` and `T_7zywpPhwVhj`.

## Status: ✅ RESOLVED

This bug has been successfully fixed. The transformer now correctly uses named types for nested shapes with assertion types.

## Error Messages

```
cannot use T_LYQafBLM8TQ{...} (value of struct type T_LYQafBLM8TQ) as AppContext value in struct literal
cannot use T_7zywpPhwVhj{...} (value of struct type T_7zywpPhwVhj) as User value in struct literal
```

## Root Cause Analysis

Based on detailed trace logs, the issue occurred in the field types extraction logic within `transformShapeNodeWithExpectedType`. The problem was that when processing nested shapes, the transformer failed to extract the expected field types from the parent type definition.

### Key Evidence from Trace Logs

1. **Outer shape processing worked correctly**:

   ```
   DEBU[0000] Expected type is compatible, using it directly  expectedType=T_488eVThFocF function=findExistingTypeForShape
   DEBU[0000] === findExistingTypeForShape RESULT ===       found=true function=transformShapeNodeWithExpectedType typeIdent=T_488eVThFocF
   ```

2. **Field types extraction failed for nested shapes**:

   ```
   DEBU[0000] Processing field in type definition           field=AppContext fieldName=ctx function=transformShapeNodeWithExpectedType
   DEBU[0000] Final extracted field types                   fieldTypes=map[input:T_azh9nsqmxaF] function=transformShapeNodeWithExpectedType
   ```

   Notice that the `ctx` field (which should be `AppContext`) was missing from the extracted field types.

3. **Nested shape transformation got no expected type**:

   ```
   DEBU[0000] Starting transformShapeNodeWithExpectedType   expectedType= function=transformShapeNodeWithExpectedType shape={sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}}
   ```

   The nested shape transformation received `expectedType=` (empty) instead of the expected `AppContext` type.

4. **Nested shape fell back to structural matching**:
   ```
   DEBU[0000] Expected type not found in typechecker definitions  expectedType= function=findExistingTypeForShape
   DEBU[0000] Found matching type definition                function=findExistingTypeForShape typeIdent=T_LYQafBLM8TQ
   ```
   Without an expected type, it found a structurally matching hash-based type instead of using the named type.

## Technical Details

### Field Types Extraction Logic Issue

The field types extraction logic in `transformShapeNodeWithExpectedType` had a bug where it only processed fields that have explicit `Type` or `Shape` properties, but failed to handle fields that are assertion types (like `AppContext`).

From the trace logs:

```
DEBU[0000] Processing field in type definition           field=AppContext fieldName=ctx function=transformShapeNodeWithExpectedType
```

The field `ctx` had type `AppContext` (an assertion type), but the extraction logic didn't handle assertion types properly, so it was skipped and not added to the `fieldTypes` map.

### Expected vs Actual Behavior

**Expected**: The nested shape `{sessionId: &sessionId, user: {name: "Alice"}}` should use the named type `AppContext` because it matches the expected type structure.

**Actual**: The nested shape used hash-based type `T_LYQafBLM8TQ` because the field types extraction failed to provide the expected type, causing the transformer to fall back to structural matching.

## Resolution

### Fix Applied

1. **Fixed Field Types Extraction Logic**: Added support for assertion types in the field types extraction logic:

```go
} else if field.Assertion != nil {
    // Handle assertion types by extracting the base type
    if field.Assertion.BaseType != nil {
        fieldTypes[fieldName] = string(*field.Assertion.BaseType)
        t.log.WithFields(map[string]interface{}{
            "fieldName":     fieldName,
            "assertion":     field.Assertion,
            "extractedType": string(*field.Assertion.BaseType),
            "function":      "transformShapeNodeWithExpectedType",
        }).Debug("Extracted assertion field type")
    } else {
        // If no base type, use "Shape" as fallback
        fieldTypes[fieldName] = "Shape"
        t.log.WithFields(map[string]interface{}{
            "fieldName": fieldName,
            "assertion": field.Assertion,
            "function":  "transformShapeNodeWithExpectedType",
        }).Debug("Assertion field has no base type, using fallback")
    }
}
```

2. **Fixed Pointer Field Logic**: Updated the pointer field logic to correctly wrap struct literals in `&` for pointer fields:

```go
} else if compLit, isStruct := fieldValue.(*goast.CompositeLit); isStruct {
    t.log.WithFields(map[string]interface{}{
        "fieldName":    name,
        "case":         "struct-literal-wrap",
        "function":     "transformShapeNodeWithExpectedType",
        "fieldValue":   fmt.Sprintf("%#v", fieldValue),
        "compLit.Type": fmt.Sprintf("%#v", compLit.Type),
    }).Debug("For pointer fields with struct literals, wrap in & (granular)")
    // For struct literals in pointer fields, wrap them in &
    fieldValue = &goast.UnaryExpr{
        Op: token.AND,
        X:  fieldValue,
    }
}
```

### Verification

The fix has been verified by:

1. **Shape Guard Example**: The shape guard example now compiles and runs successfully:

   ```go
   createTask(T_488eVThFocF{ctx: AppContext{sessionId: &sessionId, user: &User{name: "Alice"}}, input: T_azh9nsqmxaF{name: "Fix memory leak in Node.js app"}})
   ```

2. **All Tests Passing**: All transformer tests pass, confirming no regressions:

   ```
   === RUN   TestShapeGuard_NestedShapes_UseHashTypesInsteadOfNamedTypes
   --- PASS: TestShapeGuard_NestedShapes_UseHashTypesInsteadOfNamedTypes (0.00s)
   ```

3. **Correct Output**: The program runs successfully:
   ```
   Creating task, logged in with sessionId: 479569ae-cbf0-471e-b849-38a698e0cb69
   Correctly created task: Fix memory leak in Node.js app
   Correctly avoided creating gym task as user was not logged in
   ```

## Reproduction Steps

1. Compile the shape guard example:

   ```bash
   task example:shape-guard
   ```

2. Observe the compilation errors in the generated Go code:

   ```go
   // Generated code uses hash-based types instead of named types
   createTask(T_488eVThFocF{ctx: T_LYQafBLM8TQ{sessionId: sessionId, user: T_7zywpPhwVhj{name: "Alice"}}, input: T_azh9nsqmxaF{name: "Fix memory leak in Node.js app"}})
   ```

3. The Go compiler fails because it expects `AppContext` but receives `T_LYQafBLM8TQ`.

## Impact

This bug prevented the shape guard example from compiling correctly, breaking a core language feature. The issue affected any Forst code that uses nested shapes with assertion types in function parameters.

## Priority

**High** - This was a core functionality bug that prevented shape guards from working with nested types, which is a fundamental feature of the Forst language.

## Related Issues

- See also: `shape-guard-named-type-mismatch-bug.md` for the general named type mismatch issue
- See also: `shape-guard-pointer-dereference-bug.md` for pointer field issues

## Debug Information

The trace logs showed that the transformer correctly identified the outer shape type (`T_488eVThFocF`) but failed to propagate the expected types to nested shapes. The field types extraction logic was the root cause, as it didn't handle assertion types like `AppContext` properly.

## Resolution Summary

✅ **Fixed**: The transformer now correctly uses named types instead of hash-based types for nested shapes when the expected type is compatible.

✅ **Verified**: The shape guard example compiles and runs successfully.

✅ **Tested**: All transformer tests pass, confirming no regressions.

The bug has been completely resolved and the shape guard functionality now works as expected.
