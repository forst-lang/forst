# 11 — Is `ensure` + refinements a waste? User types, not `String.Foo`

**Status:** Reflection against Go. **Accepted into [12 §17, §23](./12-accepted-decision.md)** (finite sets are types; prefer domain types over `String.Foo`; alias ≠ defined type).

---

## 1. What Go actually does (not folklore)

Go has **no refinement types**. A `string` is every string. `type TaskStatus string` does **not** restrict the value to your constants. Conversion `TaskStatus(raw)` is a **cast**, not a membership test. That is why the ecosystem grows `Valid()` / `UnmarshalText` generators (`string-enumer`, etc.): the type system will not do the check.

Go **does** have the split you are reaching for. Spec, [type declarations](https://go.dev/ref/spec#Type_declarations):

| Form | Spec | Identity | Methods / “guards” |
| --- | --- | --- | --- |
| **Alias** `type T = U` | “binds an identifier to the given type”; identical to `U` | Same type as `U` | **No** own method set. `byte` is `uint8`. |
| **Definition** `type T U` | “creates a new, distinct type with the same underlying type” | Different from `U` | Methods allowed. Receiver base type **must be a defined type declared in the same package** ([Method declarations](https://go.dev/ref/spec#Method_declarations)). |

You **cannot** write `func (s string) Pending()`. Predeclared `string` is not a defined type in your package. That is the spec saying: **do not monkey-patch builtins**. The TimeZone example in the spec is the intended enum-ish pattern: `type TimeZone int` + `const` + `iota` + methods on `TimeZone`.

[Iota wiki](https://go.dev/wiki/Iota): iota is incrementing **constants**, usually on a **defined** numeric type. It is not a closed string set and not JSON `"todo"`. For wire statuses, Go programmers still use `type Status string` + `const StatusTodo Status = "todo"` and then **hand-written or generated** validation. The type is nominal; the set is a social convention plus runtime.

[Effective Go — Errors](https://go.dev/doc/effective_go#errors): successive checks, **success runs down the page**, failures `return`. “If the successful flow of control runs down the page, eliminating error cases as they arise.” That is `ensure` without the word `ensure`.

**Go’s verdict on our feature split:**

- Early return + explicit error values: **core Go**. `ensure … else BadStatus()` is that idiom with a named error ([errors 02](../errors/02-first-class-errors-normative.md)). Not a waste.
- User methods on **defined** types in the **same package**: **core Go**. Type guards on user types are that, with a narrowing story. Not a waste if scoped that way.
- Refinements on `string` / `String.Strong` as if the builtin grew methods: **Go forbids the analogue**. Waste, and it fights Go emit (you cannot put `Pending` on `string`).
- Closed finite sets in the type: **Go does not have this.** Forst adding literal unions is a **TS-facing** feature Go lacks, not a duplicate of `ensure`.

## 2. Is the whole refinement idea a waste?

**If refinements mean “every `String` may grow `Min`, `Pending`, `Strong` as user-defined subtypes of the builtin,” yes, it is a waste** for this language. It is monkey-patching. It duplicates builtin constraints, fights Go’s method rule, pollutes hover on every string, and **contradicts** literal unions: `String.Pending` vs `"todo" | "in_progress"` are two systems for the same job.

**If `ensure` means “Effective Go’s early return, with a nominal error, and after success the *subject’s type* is the type you named,” no.** That is the usability core:

```ft
func parseStatus(raw String): Result(TaskStatus, BadStatus) {
  ensure raw is TaskStatus else BadStatus()
  return raw
}
```

Go: `TaskStatus(raw)` with no check, or a generated `Valid()`. Forst: one line, typed failure, happy path below. That is worth being **central**. It is not Liquid Haskell. It is Go’s `if err != nil` plus “the thing on the left is now the *user* type, not `string`.”

**Refinement types earn their keep only as the *result* of that boundary**, not as an open season on builtins:

| After `ensure` | Worth it? |
| --- | --- |
| `String` → `TaskStatus` (closed set / defined type) | **Yes.** Field types can be `TaskStatus`; later code has no `ensure`. |
| `AppContext` → `AppContext.LoggedIn` (fields `Present`) | **Yes.** The continuation uses `*sessionId`. Brand on **your** shape. |
| `Result` → `Ok` payload | **Yes.** Discriminant of a sum you already have. |
| `String` → `String.Pending` | **No.** Monkey-patch. Use a user type or a subset of a literal union. |
| `String` → `String.Min(3)` via a **user** guard | **No.** Builtin constraint or a defined `Name` type. |

So: **`ensure` + errors + early return is not optional.** **User-guard refinements on builtins are the part that is a waste.** The contradiction with literal unions appears only if you keep both `String.Pending` and `"todo" | "in_progress"`. Drop the first.

## 3. Aliases are the wrong fix

You suggested:

```ft
type TaskStatus = String
is (s TaskStatus) Pending { … }
```

In Go that is an **alias**. `TaskStatus` **is** `String`. A guard on it **is** a guard on `String`. Same identity, same monkey-patch, just a prettier name in source. Callers can still pass any `String`. Forst’s current `type Password = String` + `is (password Password) Strong` ([guard RFC](../guard/guard.md), [`basic_guard.ft`](../guard/basic_guard.ft)) is this mistake.

Go’s fix is a **type definition** (no `=`):

```go
type TaskStatus string
```

Distinct type. `string` does not get `Pending`. Assignment from `string` is a conversion. Methods (and Forst guards) attach here, **same package only**.

Forst today only parses `type Name = …` ([`parseTypeDef`](../../../../forst/internal/parser/typedef.go)). Shipping “guards only on user types” **requires** a defined-type form (or treating a literal-union typedef as a defined type). An alias to `String` does not.

## 4. Task list: two ways, one rule

Statuses: `todo`, `in_progress`, `success`, `failed`, `deferred`. “Pending” = todo ∪ in_progress.

### Way A — closed sets (prefer)

```ft
type TaskStatus =
  | "todo"
  | "in_progress"
  | "success"
  | "failed"
  | "deferred"

type PendingStatus = "todo" | "in_progress"
```

`PendingStatus <: TaskStatus` by **set inclusion** ([10](./10-predicates-are-not-sums.md) §5). Fields: `status: TaskStatus`. Functions that only accept pending work take `PendingStatus`. JSON `String` → `ensure raw is TaskStatus else BadStatus()`.

No type guard. No `String.Pending`. No second system. TS emit is `"todo" | "in_progress"`. Go emit is a defined `string` type plus a membership check at `ensure` (what Go authors generate by hand).

### Way B — guard on the **user** type (only if the rule is not a finite subset)

```ft
type TaskStatus = "todo" | "in_progress" | "success" | "failed" | "deferred"

is (s TaskStatus) Pending {
  ensure s is Value("todo") or Value("in_progress")
}
```

This is **legal** under “guards on types you defined,” and it is **redundant** with Way A. Two spellings for the same subset. **Do not offer both as equal.** Rule:

> If the predicate is a **subset of a closed literal (or enum) set**, it is a **type**. If it is not a finite subset — extra fields, time, “not terminal and assignee present” — it is a **type guard on that user type**.

Example that **is** a guard:

```ft
type Task = {
  status: TaskStatus,
  assignee: *User,
}

is (t Task) Actionable {
  ensure t.status is PendingStatus
  ensure t.assignee is Present()
}
```

`Actionable` is not a literal union. It is a predicate on **your** shape. `ensure t is Actionable() else NotActionable()`. Continuation may use `*assignee`. That is refinement worth keeping.

### Illegal

```ft
is (s String) Pending { … }           // builtin
is (s otherpkg.TaskStatus) Pending    // not your type; Go would reject the method
type TaskStatus = String
is (s TaskStatus) Pending { … }       // alias: still String
```

## 5. Enums (how they fit; they do not fight `ensure`)

| Kind | Role | `ensure`? |
| --- | --- | --- |
| **Literal union** | Closed wire set; subsets are types; TS unions | At the **String → set** boundary only |
| **Defined `type TaskStatus String` + `const`** | Go-faithful nominal enum; set not closed in the type | `ensure s is TaskStatus` cannot mean “one of the consts” unless you also have literals or a generated membership guard **on TaskStatus** |
| **`iota` on `type Kind Int`** | Go numeric enum ([spec TimeZone](https://go.dev/ref/spec#Type_definitions)) | Not JSON strings; keep for Go ports |

For a **task app with TS clients**, literal unions (Way A) are the enum. Defined `string` types without a closed set are Go’s hole; do not copy the hole and then invent `String.Pending` to fill it.

`ensure` sits at the **edge**: untrusted `String` or a wider status → named type. Inside the module, parameters are already `TaskStatus` / `PendingStatus`. That is fewer conditions, not more ([10](./10-predicates-are-not-sums.md) §4).

## 6. Builtin constraints vs user guards

`name: String.Min(1)` on a **field you declared** is not monkey-patching `String`. It is a language atom on a use site ([08](./08-complexity-bounds.md)). User `is (s String) Minish` is the patch.

Keep builtins as **compiler** constraints. Keep user guards as **methods on defined / literal-union types in the defining package** (Go’s receiver rule). Shape guards (`LoggedIn` on `AppContext`) already follow this if `AppContext` is yours.

## 7. Usability claim (what must stay central)

Without refinements-on-`String`, the language still has a tight story:

1. **Happy path down the page** — Effective Go, spelled `ensure`.
2. **Failure is a named error on `or`** — errors RFC, not `if err != nil { return err }` soup.
3. **Domain types are yours** — literal unions / defined types / shapes; TS and Go can see them.
4. **Guards name extra rules on those types** — `LoggedIn`, `Actionable`, not `String.Foo`.
5. **Subsets of enums are types** — `PendingStatus`, not a second predicate dialect.

That does not contradict sum types. Sums are the **data**. Guards are **extra predicates on data you own**. `ensure` is **how you move from untrusted / wide to that data, or fail**.

If we keep `is (s String) Strong` in examples, the contradiction stays and the feature looks like a research refinement system. It isn’t, and it should not be ([00](./00-the-trap.md)).

## 8. Verdict

| Question | Answer |
| --- | --- |
| Waste of time as a **Liquid-style** system on builtins? | **Yes.** Drop it. |
| Waste of time as **`ensure` + nominal errors + narrowing to user types**? | **No.** That is the Go-shaped, TS-emitting product. |
| `type TaskStatus = String` then guards? | **No.** Alias ≡ `String`. Use a **defined** type or a **literal union**. |
| Guards only on types defined in this package? | **Yes.** Same rule as Go methods. |
| Pending vs literal union? | **Pending is a subset type.** Guard only if the rule is more than a subset. |
| Enums vs refinements? | **Enums/unions are the set. `ensure` converts into the set. Guards are other predicates on that set’s type.** |

Language gap: Forst needs a **defined type** declaration (Go `type T U`), or must treat `type T = "a" | "b"` as a distinct type (not an alias of `String`). Until then, `type Password = String` will keep teaching the wrong model.
