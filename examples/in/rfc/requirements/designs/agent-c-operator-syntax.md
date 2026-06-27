# Function Requirements — Agent C: Type-Indexed Operator Syntax

**Author:** Language Design Agent C  
**Status:** Design proposal (competition entry)  
**Philosophy:** Minimal keywords, maximum traceability — use **`?Type`** to declare slots and **`@require Type`** to resolve them.

**Depends on:** [optionals / Result prior art (Effect `R` channel)](../optionals/13-result-constructors-prior-art.md), [sidecar decisions](../sidecar/10-decisions.md), [errors normative](../errors/02-first-class-errors-normative.md).

**Competes with:** Keyword-heavy designs (`needs`, `using`, `with`) that treat requirements as English prose rather than type-indexed slots.

---

## Executive summary

Forst functions should declare **runtime capabilities** separately from **data parameters**. Agent C proposes two operators:

| Operator | Role | Reads as |
| --- | --- | --- |
| **`?T`** | Requirement **slot** (type position) | “This function may use capability **T**.” |
| **`@require T`** | Requirement **resolution** (expression) | “Give me the **T** implementation from the scope bundle.” |

Requirements are **not** injected through a framework container. They lower to a **plain Go struct field** (`ReqSet`) passed explicitly at boundaries (HTTP handler, test helper, `main`). Every `@require` in source maps 1:1 to a field read in generated Go — no reflection, no magic globals.

---

## 1. Core primitive(s)

### 1.1 `requirement` — the capability contract

A **requirement** is a named interface-like contract: methods are the capability surface; there is no instance data in the declaration itself.

```forst
requirement UserRepo {
  Find(id Int) Result(User, Error)
  Save(user User) Result(User, Error)
}

requirement Clock {
  Now() Time
}
```

**Lowering:** each `requirement R` becomes a Go interface `R` (same as today’s nominal type emission pattern). Implementations are ordinary Forst/Go types that satisfy the method set.

### 1.2 `?T` — requirement slot (type operator)

**`?T`** is a **prefix type operator** meaning “function **requires** capability **T**.” It appears only in:

- an optional **`needs`** clause on a function (see §2), or
- inferred requirement metadata attached to the function type (when `@require` appears in the body and no clause is written).

`?T` is **not** a parameter. It does not appear in call sites as an argument. Callers inherit or satisfy requirements through **composition** (§4).

### 1.3 `@require T` — resolution expression

**`@require T`** is a **prefix expression** that yields a value of type **`T`** (the requirement interface) by reading from the function’s scope **`ReqSet`**.

```forst
repo := @require UserRepo
```

**Lowering:** `req.UserRepo` where `req` is the hidden (or explicit-at-boundary) bundle parameter.

### 1.4 `ReqSet` — the Go bundle (compiler-generated)

For each package (or each entrypoint module), the compiler emits a struct aggregating all requirement slots used by functions in that compilation unit:

```go
// generated — names stable, field-per-requirement-type
type ReqSet struct {
    UserRepo UserRepo
    Clock    Clock
}
```

**Rule (Go compatibility):** if a struct field solves it, **do not** use DI. `ReqSet` **is** the struct field pattern — one field per requirement **type**, wired at the edge.

### 1.5 Boundaries: where `ReqSet` becomes explicit

| Site | Pattern |
| --- | --- |
| **`main` / CLI** | `req := ReqSet{ UserRepo: postgresUserRepo, Clock: systemClock }` then pass `&req` into handlers |
| **HTTP / sidecar entry** | Framework constructs `ReqSet` once per process (or per request if needed), passes pointer into Forst-generated handlers |
| **Tests** | `ReqSet{ UserRepo: &FakeUserRepo{…}, Clock: &FixedClock{t: …} }` — LLMs can generate fakes from the `requirement` block alone |
| **Internal calls** | Hidden — callee inherits caller’s `req` pointer; no parameter noise |

---

## 2. Syntax examples

### 2.1 Basic: infer needs from `@require`

```forst
requirement UserRepo {
  Find(id Int) Result(User, Error)
}

func getUser(id Int) Result(User, Error) {
  repo := @require UserRepo
  return repo.Find(id)
}
// Inferred signature (conceptual):
//   func getUser(id Int) Result(User, Error) needs ?UserRepo
```

### 2.2 Explicit `needs` clause (documentation + early errors)

```forst
func getUser(id Int) Result(User, Error) needs ?UserRepo {
  repo := @require UserRepo
  return repo.Find(id)
}
```

If the body uses `@require Clock` but `needs` omits `?Clock`, that is a **compile error** (explicit clause wins for author intent; body must match).

### 2.3 Multiple requirements — set merge

```forst
func auditGetUser(id Int) Result(User, Error) needs ?UserRepo, ?Clock, ?AuditLog {
  repo := @require UserRepo
  clock := @require Clock
  log := @require AuditLog

  user := repo.Find(id)
  ensure user is Ok()
  log.Record("getUser", id, clock.Now())
  return user
}
```

**Call-site ergonomics:** callers do **not** pass `?AuditLog`. If `auditGetUser` is invoked from a handler that already has `ReqSet` with all three fields populated, the call is a normal function call. Requirement propagation is **type-level**, not argument-level.

### 2.4 Chained internal calls

```forst
func getUser(id Int) Result(User, Error) {
  repo := @require UserRepo
  return repo.Find(id)
}

func getUserEmail(id Int) Result(String, Error) {
  user := getUser(id)   // inherits ?UserRepo from callee
  ensure user is Ok()
  return user.email
}
// Inferred: getUserEmail needs ?UserRepo
```

The checker **unions** requirement sets along the call graph within a package.

### 2.5 Test stub (minimal boilerplate)

```forst
requirement UserRepo {
  Find(id Int) Result(User, Error)
}

type FakeUserRepo = {
  users: map[Int]User,
}

func (f FakeUserRepo) Find(id Int) Result(User, Error) {
  u, ok := f.users[id]
  ensure ok or NotFound({ id: id })
  return u
}

func Test_getUser_found() {
  req := ReqSet({
    UserRepo: FakeUserRepo({ users: map[Int]User{ 1: User({ name: "Ada" }) } }),
  })
  with req {
    got := getUser(1)
    ensure got is Ok()
    ensure got.name == "Ada"
  }
}
```

**`with req { … }`** (boundary helper — the one keyword block worth having) temporarily installs a `ReqSet` for the dynamic extent of a block. Lowers to Go:

```go
func Test_getUser_found(t *testing.T) {
    req := ReqSet{ UserRepo: FakeUserRepo{…} }
    withReq(&req, func() {
        got := getUser(1)
        …
    })
}
```

Alternative spelling for tests: pass `req` as an extra parameter only in **`test`**-annotated functions — Agent C prefers **`with req`** for production parity (same `@require` paths as prod).

### 2.6 HTTP handler edge (sidecar)

```forst
func handleGetUser(req HttpRequest) Result(HttpResponse, Error) needs ?UserRepo {
  id := parseId(req.path)
  user := getUser(id)   // still ?UserRepo — same bundle
  return jsonResponse(user)
}
```

Sidecar bootstrap (Go host, not Forst):

```go
bundle := forstpkg.ReqSet{
    UserRepo: db.NewUserRepo(pool),
}
http.HandleFunc("/users/", forstpkg.WrapHandler(bundle, handleGetUser))
```

---

## 3. Go transpilation strategy

### 3.1 Function lowering

Every function that **needs** at least one requirement gets a **final parameter**:

```go
func getUser(req *ReqSet, id int) (User, error) {
    repo := req.UserRepo
    return repo.Find(id)
}
```

Functions with **no** requirements omit the parameter entirely — zero overhead for pure helpers.

**Naming:** generated parameter is always `req` (or `_req` if shadowing — compile error in Forst before emit).

### 3.2 `@require T` lowering

| Forst | Go |
| --- | --- |
| `@require UserRepo` | `req.UserRepo` |
| `repo := @require UserRepo` | `repo := req.UserRepo` |

Optional **nil-safe** mode (configurable per build tag): emit a panic or structured error if `req.UserRepo == nil` at first use — catches wiring bugs in tests quickly.

### 3.3 Internal call lowering

Caller has `req *ReqSet`; callee needs subset → pass **same pointer**:

```go
func getUserEmail(req *ReqSet, id int) (string, error) {
    user, err := getUser(req, id)
    …
}
```

No allocation, no merge at runtime — **set inclusion** is verified statically.

### 3.4 `ReqSet` emission

- One **`ReqSet` struct per package** (or per merged package in examples like tictactoe).
- Fields emitted **only** for requirement types referenced by `?T` in that package.
- Field order: **stable sort by requirement name** (deterministic diffs for golden tests).
- Exported for tests and host wiring.

### 3.5 Traceability guarantees

1. **Source grep:** `@require UserRepo` finds all resolution sites.
2. **Type grep:** `?UserRepo` finds all functions that depend on the capability.
3. **Go grep:** `req.UserRepo` matches 1:1 (modulo generated param name).
4. **No registry:** there is no `map[reflect.Type]interface{}` — LLM-generated tests can stub by struct literal without understanding a framework.

### 3.6 Performance

- One pointer pass-through per call frame (same cost as `context.Context` chaining, but **typed**).
- Field access is a direct interface load — inlineable.
- Requirement **union** is compile-time only; no runtime set algebra.

---

## 4. Inference rules

### 4.1 Collect `@require` sites

Within function body `F`, collect set **`R(F) = { T | @require T appears in F }`**.

### 4.2 Explicit `needs` clause

If `needs ?T1, ?T2, …` is present:

- **`R(F)` must equal** `{T1, T2, …}` (order irrelevant).
- Mismatch → error: “body requires `?Clock` but `needs` clause omits it” / “`needs ?AuditLog` unused in body”.

**Optional lint:** allow unused `needs` entries only if annotated `needs ?AuditLog /* reserved */` — v2.

### 4.3 No explicit clause

If no `needs` clause, **`needs` metadata = `R(F)`** sorted lexicographically, attached to function type for export to TypeScript (`forst generate`) and docs.

### 4.4 Call propagation

When `F` calls `G`:

- **`Needs(F) ⊇ Needs(G)`** after inference (caller must have at least callee’s requirements).
- If `G` is in another package, **`Needs(G)`** is read from its exported signature metadata; caller’s package **`ReqSet`** must include those fields (cross-package requirements merge into importers’ bundle at link/compile time).

### 4.5 Entrypoints

Functions designated as **entrypoints** (`main`, HTTP handlers, discovered sidecar exports) must have their **`ReqSet` populated by the host**. The compiler emits a **wiring checklist** (debug log / `forst doctor`): “`handleGetUser` needs `[UserRepo]`”.

### 4.6 Subtyping

If `PostgresUserRepo` implements `UserRepo`, the field type is **`UserRepo`** (interface), not the concrete type — same as Go.

### 4.7 Relationship to `Result(S, F)`

Requirements are **orthogonal** to error types. A function may be:

```forst
func getUser(id Int) Result(User, Error) needs ?UserRepo
```

Do **not** encode requirements in `Result`’s third type parameter (contrast Effect’s `R`). Keeps Forst’s binary `Result` story intact per [optionals 13](../optionals/13-result-constructors-prior-art.md).

---

## 5. Why operators might be clearer than keywords here

### 5.1 Type-indexed, not prose-indexed

Keywords like `using UserRepo` or `with service UserRepo` read as **instructions to a runtime**. **`?UserRepo`** attaches the dependency to the **type name** itself — the same identifier used in `@require UserRepo` and in the `requirement UserRepo` declaration. Three sites, one token: **`UserRepo`**.

Grep and LSP jump-to-definition work on **`UserRepo`**, not on the word “using”.

### 5.2 Visual distinction: data vs capability

| Kind | Syntax | Call site |
| --- | --- | --- |
| Data parameter | `id Int` | `getUser(42)` |
| Requirement slot | `needs ?UserRepo` | *(invisible)* |
| Resolution | `@require UserRepo` | *(inside body)* |

Parameters stay in the **`(...)`** list; requirements stay in **`needs (...)`** or inference metadata. No mixing “`ctx Context, userRepo UserRepo`” where one field is data and one is a service — the **`?`** marker makes the category explicit in signatures.

### 5.3 `@require` is an action, not a declaration

**`require UserRepo`** (keyword) looks like a top-level import or a second `import` syntax. **`@require UserRepo`** is unambiguously an **expression** — it produces a value. Parser classification is trivial: `@` starts an expression operator (similar to **`&`** for references).

### 5.4 Traceability for tooling and LLMs

- **Compiler:** `@require T` → field `T` on `ReqSet` — fixed rewrite rule.
- **Tests:** read `requirement UserRepo { … }`, implement methods, set **`ReqSet.UserRepo`** — no container registration API to learn.
- **Code review:** every capability pull is **`@require`** — zero hidden globals.

Keyword designs often drift toward **`Provide` / `Inject` / `Resolve`** families with overlapping semantics. One resolution operator reduces decision fatigue.

### 5.5 Minimal reserved words

Only **`requirement`**, **`needs`**, and boundary **`with`** are new keywords. **`?`** and **`@require`** are operators — less lexer pressure than a family of English verbs (`using`, `given`, `depends`, `from`).

### 5.6 Honest tradeoff

**`?T`** may collide with a future **optional type** syntax. Mitigation: requirement operator **`?`** is only valid **immediately after `needs`** or in requirement-type metadata — not in arbitrary type positions. If optionals adopt **`T?`** postfix instead, coexistence is clean.

---

## 6. Edge cases

### 6.1 Duplicate `@require` in one function

```forst
a := @require UserRepo
b := @require UserRepo
```

Allowed. **`a`** and **`b`** are the same interface value; compiler may reuse one temp in Go.

### 6.2 Requirement used conditionally

```forst
if debug {
  log := @require AuditLog
  log.Record(…)
}
```

**`AuditLog`** is still in **`Needs(F)`** — static analysis includes all branches (Go-style). Rationale: wiring must be total at entrypoints; conditional **use** should not hide dependencies from the checklist.

**Escape hatch (v2):** `needs ?AuditLog optional` — omitted from entrypoint wiring if nil allowed; `@require` lowers to nil-checked call.

### 6.3 Same requirement type, different semantics

Two different **`UserRepo`** contracts in one program should use **distinct requirement names**:

```forst
requirement ReadUserRepo { … }
requirement WriteUserRepo { … }
```

Do not overload one interface for unrelated capabilities — field slots are **type-indexed by name**, not by structural typing.

### 6.4 Recursive / circular requirements

```forst
requirement A { B() }
requirement B { A() }
```

Forbidden at **`requirement`** declaration time (cycle in requirement graph). Implementations may reference each other via methods, but not via `@require` during **`requirement` method bodies** (requirements cannot `@require` in interface declarations).

### 6.5 `@require` in `requirement` impl vs business func

Implementation methods on concrete types use **normal fields**, not `@require`. **`@require`** is only valid in **`func`** bodies that receive an scope **`ReqSet`**.

### 6.6 Cross-package re-exports

Package **`api`** calls package **`domain`**’s `getUser` needing **`?UserRepo`**. **`api`’s** inferred `handleGetUser` needs **`?UserRepo`** too. **`api.ReqSet`** includes **`UserRepo`** field; **`main`** wires once.

### 6.7 Go interop: host-provided implementations

Go types can implement Forst `requirement` interfaces via generated Go interfaces. Wiring:

```go
bundle.UserRepo = myGoRepo // satisfies forst.UserRepo
```

No Forst syntax required in Go host beyond struct literal.

### 6.8 Generics (future)

When user generics exist, **`?Cache[T]`** may index requirements by type arguments. Defer until [generics RFC](../generics/00-user-generics-and-type-parameters.md) lands; v1 is monomorphic requirement names only.

### 6.9 Conflict with pointer/type syntax

Forst already has **`*T`** pointers. **`?T`** is distinct (prefix on requirement slots only). Parser context: after **`needs`**, `?` starts requirement list, not a ternary.

### 6.10 Discovery / sidecar metadata

Extend discovery JSON with **`requirements: ["UserRepo", "Clock"]`** per exported function — derived from `needs` / inference. TypeScript client can document capabilities without reading Go.

### 6.11 Failure modes

| Situation | Behavior |
| --- | --- |
| Missing field on `ReqSet` at runtime | Panic or `MissingRequirement` error (build-tag policy) |
| `@require T` but `T` not in `needs` / inference | Compile error |
| Entrypoint with no host wiring | `forst doctor` warning + link to wiring example |
| Test without `with req` | Compile error: “no scope ReqSet for `@require UserRepo`” |

---

## 7. Comparison snapshot (why Agent C)

| Criterion | Agent C (`?` / `@require`) | Typical keyword design |
| --- | --- | --- |
| Traceability to Go | Field read on `ReqSet` | Often interface{} registry or ctor injection |
| Grepability | `?UserRepo`, `@require UserRepo` | “using”, “depends on”, varied |
| Call-site noise | None (bundle at edge) | Sometimes extra parameters |
| Test boilerplate | Struct literal + `with req` | Mock frameworks, DI setup |
| New keywords | 2–3 | 5+ |
| LLM test generation | Read `requirement` block → fake struct | Learn container API |

---

## 8. Implementation sketch (compiler phases)

1. **Parse:** `needs ?T, ?U`; `@require T` as prefix expression.
2. **Resolve:** map `T` to `requirement` symbol.
3. **Infer:** compute `Needs(F)`; propagate across calls.
4. **Check:** `@require T` implies `T ∈ Needs(F)`; entrypoint wiring metadata.
5. **Emit:** `ReqSet` struct; append `req *ReqSet` parameter; rewrite `@require`.
6. **Test:** golden `examples/in/rfc/requirements/operator_basic.ft` → Go with struct literal test.

---

## 9. Open questions

1. **`with req` vs explicit test-only parameter** — which golden tests prefer?
2. **Per-request vs process-wide `ReqSet`** — host policy; Forst only types the struct.
3. **Optional requirements** — v1 strict (total wiring) or v1 `optional` annotation?
4. **Operator spelling** — `@require` vs `@` alone (`@UserRepo`)? Agent C prefers **`@require`** for grep clarity; **`@UserRepo`** is shorter if `@` is reserved exclusively for requirements.

---

## 10. Recommended next step

Add **`examples/in/rfc/requirements/operator_basic.ft`** with golden Go output and a **`TestRequirements_operatorBasic`** integration test — same pattern as `nominal_error.ft` and tictactoe merged package. Validates the full pipeline before committing to lexer tokens **`?`** and **`@`**.
