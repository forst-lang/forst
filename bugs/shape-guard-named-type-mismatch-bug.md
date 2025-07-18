# Hash-based and Named Type Shape Guard Compatibility Bug

## Summary

When using shape guards (type guards) in Forst, the typechecker and transformer do not treat hash-based types (auto-generated for anonymous shapes) and named types (user-defined) as compatible, even if their underlying structure is identical. This causes shape guards to fail to match, resulting in errors like:

    failed to transform constraint: no valid transformation found for constraint: Valid

## Steps to Reproduce

1. Define a named type (e.g., `User`) and a function that returns a struct literal matching that shape.
2. Add a shape guard (e.g., `is (user User) Valid`) and use it in an `ensure` statement in the function.
3. Run the test `TestEnsureErrorReturnType_UsesNamedReturnType` in `forst/internal/transformer/go/transformer_test.go`.
4. Observe that the transformer fails to find a compatible type guard, even though the struct literal and the named type are structurally identical.

## Example Debug Output

```
time="..." level=info msg="Checking type guard compatibility" function=isTypeGuardCompatible typeGuard=Valid varType=T_i6MUJeF9PoV
time="..." level=info msg="Expected base type of type guard identified based on variable type" baseType=T_i6MUJeF9PoV function=isTypeGuardCompatible
time="..." level=info msg="Checking parameter type" baseType=T_i6MUJeF9PoV function=isTypeGuardCompatible paramType=User
time="..." level=info msg="Type compatibility check result" baseType=T_i6MUJeF9PoV compatible=false function=isTypeGuardCompatible paramType=User typeGuard=Valid
time="..." level=info msg="No compatible type guard found" baseType=T_i6MUJeF9PoV function=isTypeGuardCompatible typeGuard=Valid
```

## Root Cause Analysis

- The typechecker uses `IsTypeCompatible` to determine if the type of the variable being guarded (e.g., `T_i6MUJeF9PoV`) is compatible with the parameter type of the type guard (e.g., `User`).
- `IsTypeCompatible` currently only checks for direct type name matches or shallow aliasing.
- If the types are not directly equal, but both are shape types with identical fields, the function does not recognize them as compatible.
- This is a problem for shape guards, as the transformer often generates hash-based types for struct literals, while type guards are defined for named types.

## Proposed Solution

- Patch `IsTypeCompatible` to resolve both types to their underlying shape (using the type definition in `tc.Defs`).
- If both are shapes, compare their fields for structural equality (field names and types).
- If the shapes are structurally identical, treat the types as compatible for the purposes of type guards and function return type checks.

## Impact

- This bug prevents valid Forst code using shape guards from compiling or generating correct Go code.
- Developers are forced to work around the issue by avoiding shape guards or manually aligning type names, which defeats the purpose of structural typing.
- Fixing this will improve developer experience and make Forst's type system more robust and ergonomic.

## References

- Failing test: `TestEnsureErrorReturnType_UsesNamedReturnType` in `forst/internal/transformer/go/transformer_test.go`
- Type compatibility logic: `IsTypeCompatible` in `forst/internal/typechecker/go_builtins.go`

---

# Shape Guard Named Type Mismatch Bug

## Summary

When compiling the shape guard example, the Forst compiler generates Go code that uses hash-based types (e.g., `T_BCsiNy7Ed6r`, `T_F5XAayb9XdF`, `T_Y82wrgBRenH`) for struct literals, instead of the expected named types (`AppContext`, `User`, etc.). This leads to invalid Go code and type errors at runtime.

## Error Message

```
cannot use T_BCsiNy7Ed6r{...} as AppContext in argument to ...
cannot use T_7zywpPhwVhj{...} as User in argument to ...
```

## Root Cause

The transformer fails to match shape literals to named type definitions when generating Go code. Instead, it falls back to using hash-based types, even when the shape structure matches a named type. This is due to overly strict or incorrect shape matching logic in the transformer, which compares field values instead of just field names and types.

## Evidence from Debug Logs

- Debug logs show that the transformer finds matching type definition for shape literal, but still emits hash-based types in the generated Go code.
- Example log:
  ```
  Found matching type definition typeIdent=AppContext for shape literal
  [DEBUG] Using structType for Go code generation structType=&ast.Ident{Name:"T_BCsiNy7Ed6r"}
  ```
- The generated Go code uses `T_BCsiNy7Ed6r{...}` instead of `AppContext{...}`.

## Original Forst Code

```forst
type User = {
  name: String,
}
type AppContext = {
  sessionId: *String,
  user: *User,
}

func main() {
  sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
  ctx := {
    sessionId: &sessionId,
    user: {
      name: "Alice",
    },
  }
  // ...
}
```

## Expected Behavior

The transformer should generate:

```go
AppContext{
  sessionId: &sessionId,
  user: &User{name: "Alice"},
}
```

## Actual Behavior

The transformer generates:

```go
T_BCsiNy7Ed6r{
  sessionId: &sessionId,
  user: &T_7zywpPhwVhj{name: "Alice"},
}
```

## Technical Analysis

- The shape matching logic in the transformer is too strict or incorrect, failing to match shape literals to named types based on structure alone.
- As a result, the transformer falls back to using hash-based types for struct literals, even when a named type is available and expected.
- This causes type errors in the generated Go code, as the Go compiler expects named types, not hash-based types.

## Steps to Reproduce

1. Write a Forst function that takes a shape literal matching a named type (e.g., `AppContext`, `User`) as an argument, as in the shape guard integration example.
2. Compile the code and inspect the generated Go code.
3. Observe that the struct literals use hash-based types instead of the expected named types.
4. Attempt to compile the Go code and observe type errors.

## Impact

This bug prevents shape guard and type-safe struct literal usage in Forst, breaking a core language feature and causing runtime type errors in Go.

## Priority

**High** - This is a core functionality bug that prevents shape guards from working, which is a fundamental feature of the Forst language.

## Additional Notes

- The bug is confirmed by both integration and unit tests.
- Fixing the shape matching logic to match on field names and types (not values) should resolve the issue.
- See also: `shape-guard-pointer-dereference-bug.md` for a related pointer field bug.
