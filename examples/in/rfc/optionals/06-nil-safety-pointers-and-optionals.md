# Nil safety, pointers, and whether `T | Nil` is a real win

This note answers: **How far does Forst already prevent nil pointer dereferences?** and **Would Crystal-style optionals (`T | Nil`) add unnecessary complexity, or a meaningful safety / clarity win?**

It is grounded in the current compiler ([`infer_expression.go`](../../../../forst/internal/typechecker/infer_expression.go), [`infer_assignment.go`](../../../../forst/internal/typechecker/infer_assignment.go), [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)). **Hub:** [00](./00-crystal-inspired-optionals.md); **motivation / tradeoffs:** [07](./07-background-motivation-and-tradeoffs.md); **lowering:** [08](./08-go-interop-lowering.md).

**Status:** engineering assessment for the [optionals RFC](./README.md). Not a product commitment.

---

## 1. What Forst already enforces (today)

### 1.1 Dereference is type-correct only

For `*x` (**`DereferenceNode`**), the typechecker requires the operand’s type to be a **pointer** (`TypePointer`). Dereferencing a **non-pointer** is a **type error**.

```270:284:forst/internal/typechecker/infer_expression.go
	case ast.DereferenceNode:
		valueType, err := tc.inferExpressionType(e.Value)
		// ...
		if valueType[0].Ident != ast.TypePointer {
			return nil, fmt.Errorf("dereference is only valid on pointer types, got %s", valueType[0].Ident)
		}
		// ...
		return valueType[0].TypeParams, nil
```

This is **structural** correctness (you cannot `*` a `String`); it is **not** a guarantee that the pointer is **non-nil** at runtime.

### 1.2 `nil` assignment rules (limited)

For **`var` declarations** with an explicit **`nil`** literal, assignment is only allowed when the explicit type is one of **pointer**, **map**, **slice**, **interface** (`object`), or (TODO) **function**—not arbitrary value types.

```84:96:forst/internal/typechecker/infer_assignment.go
		// For var declarations, check nil assignment is only allowed for pointer, interface, map, slice, channel, or function types
		if isVarDeclaration && len(assign.RValues) == 1 {
			if _, isNil := assign.RValues[0].(ast.NilLiteralNode); isNil {
				explicitType := variableNode.ExplicitType
				isPointer := explicitType.Ident == ast.TypePointer
				// ...
				if !(isPointer || isInterface || isMap || isArray || isFunc) {
					return fmt.Errorf("cannot assign nil to variable of type '%s'", explicitType.Ident)
				}
			}
		}
```

So Forst already **blocks** obviously wrong cases like `var x String = nil` (when spelled as explicit `nil`). This does **not** cover:

- Short declarations where **`nil`** is inferred from context elsewhere.
- Proving that a **`*T`** value is non-nil **before** `*p` is evaluated.

### 1.3 Narrowing: refinements and `ensure`, not pointer liveness

[SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) states **non-goals** for the current design:

- **Whole-program pointer analysis**, **full alias tracking**, **SMT**, arbitrary loop joins.
- **Compound `ensure` subjects** (e.g. field paths) deferred.

There is **no** documented **flow-sensitive** type such as “non-nil `*T`” that is proven after `if p != nil` or similar for **arbitrary** pointers. **`ensure`** / **`is`** narrowing targets **assertions** and **refinements** ([guard RFC](../guard/guard.md)), not Rust-style **pointer provenance**.

**Bottom line today:** Forst prevents **bogus dereferences** (wrong type). It does **not** generally prevent **nil pointer dereference** of a well-typed `*T`.

---

## 2. What `T | Nil` (optionals) would add

### 2.1 Different object than “safe `*T`”

- **`T | Nil`** models **absence of a `T` value** without necessarily using a **pointer** ([00](./00-crystal-inspired-optionals.md), [03](./03-typescript-emission-optionals-and-result.md)).
- Go **interop** often still lowers optionals to **`*T`** or **`(T, bool)`** ([08](./08-go-interop-lowering.md)). If the lowering is **pointer-based**, **runtime** nil risk can **remain** unless **every** use goes through **narrowing** and the **emitter** never emits raw `*` on a possibly-nil pointer without a check.

So **optional types alone** are **not** a silver bullet for nil deref **unless** paired with:

- **Strict** lowering rules, and/or  
- **Non-nullable** pointer types / **flow-sensitive** non-nil **`*T`** (major type-system work—explicitly out of scope for v1 per [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)).

### 2.2 Real wins (even without full pointer proof)

| Win | Why it matters |
| --- | ---------------- |
| **Clearer absence vs failure** | **`T | Nil`** vs **`Result(T, Error)`** vs **`(T, error)`** reduces **sentinel** and **zero-value** ambiguity ([00](./00-crystal-inspired-optionals.md)). |
| **Narrowing on `Nil`** | After **`is`** / control flow, type becomes **`T`**—same **machinery** as other unions; **fewer** “did I check nil?” **mistakes** for **optional-shaped** APIs **if** the **source** type is **`T \| Nil`** and the **checker** enforces use-after-narrow. |
| **Less reliance on `*T` for “maybe value”** | Fewer **pointers** used **only** to mean “optional,” which reduces **one** source of nil deref (not all). |

### 2.3 Complexity cost

| Cost | Detail |
| ---- | ------ |
| **Parallel concepts** | **`T \| Nil`**, **`*T`**, **`(T, error)`**, **`Result`**—**docs** and **style** must stay strict ([00](./00-crystal-inspired-optionals.md)). |
| **Implementation** | Parser, checker, narrowing, **two** backends ([04](./04-tooling-migration-lsp-and-testing.md)). |
| **Expectation management** | Authors may **assume** “optional = no nil bugs”; **without** non-nil **`*T`**, **that** claim is **false** for **pointer**-lowered code. |

---

## 3. Verdict

- **Today:** Forst **does not** broadly **prevent nil pointer dereferences**; it prevents **ill-typed** dereferences and **some** bad **`nil`** assignments, and documents that **deep** nil safety is **not** a current soundness goal ([SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)).
- **`T | Nil` / `T?`:** A **real** win mainly for **modeling absence**, **narrowing**, and **API clarity**, and **indirectly** for safety where **optional** replaces **pointer-for-optionality**. It is **not** automatically a **large** win on **nil deref** **unless** paired with **stricter** pointer / lowering rules **or** future **non-nil** / **flow** analysis—otherwise it risks **adding** complexity **without** matching **user** expectations on **memory** safety.

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md)
- [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)
- [infer_expression.go](../../../../forst/internal/typechecker/infer_expression.go) — **`DereferenceNode`**
- [infer_assignment.go](../../../../forst/internal/typechecker/infer_assignment.go) — **`nil`** on **`var`**

---

## Document status

**Living analysis.** Update if **pointer** / **non-nil** typing or **optional** lowering **lands**.
