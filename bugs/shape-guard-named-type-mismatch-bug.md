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
