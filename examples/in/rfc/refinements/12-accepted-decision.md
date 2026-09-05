# 12 — Accepted decision: analyzable refinements and the role of `ensure`

**Status:** **Accepted, amended.** Normative except where [the locked technical design](../../../../.plans/analyzable-refinements/TECHNICAL-DESIGN.md) overrides historical alternatives. Type targets are [14](./14-type-targets.md); stability is [13](./13-refinement-stability.md).

> **Amendment:** Forst has no source aliases. `type Name = TypeExpression` always
> declares a named type; `=` is the declaration separator. This work ships named
> homogeneous finite string/integer/boolean unions and enum subsets only. Mixed,
> float, inline, shape, tagged, and general unions are deferred. Contrary examples
> below are historical rationale, not implementable semantics.

**Scope:** `ensure`, type guards, assertion disjunction, failure handling, refinement propagation. Mutation, aliasing, and Go interop are specified in [13](./13-refinement-stability.md), not here.

**Supersedes:** [06](./06-recommendation.md) where this document conflicts with it. Analysis in [00](./00-the-trap.md)–[11](./11-ensure-worth-and-user-types.md) remains background. Do not implement against 06’s DNF-unfold requirement, [09](./09-nominal-proxies.md)’s ban on call-site disjunction, the draft that kept `or` as the ensure **failure** keyword, or the draft of this file that used `|` for assertion alternatives.

**Mutation / aliasing / Go:** [13](./13-refinement-stability.md). This file does not define those rules and does not reopen them.

**Implementation plan:** [`.plans/analyzable-refinements`](../../../../.plans/analyzable-refinements/README.md).

---

## 1. Objective

Forst should make `ensure` a central mechanism for establishing runtime invariants that the type system can remember.

The core developer experience should be:

> Define an invariant once, establish it once with `ensure`, and let the compiler remember it afterward.

This is intended to reduce repeated conditions and dispersed validation logic.

Forst should **not** attempt to encode arbitrary program logic into its type system.

The refinement system must remain:

- statically analyzable;
- predictable;
- easy to explain;
- cheap to compile;
- compatible with Go lowering;
- useful for generated TypeScript contracts where applicable.

No SMT solver, theorem prover, dependent type system, or arbitrary predicate inference is introduced.

---

## 2. Core conceptual model

Forst distinguishes four concepts.

| Concept | Construct |
| --- | --- |
| Which values may exist | Types |
| One of several possible **types** | `\|` |
| One of several possible **assertions** on the same place | `or` |
| A reusable invariant about a value | Type guard |
| RHS of `ensure` / `if … is` | **Refinement target:** type **or** assertion ([14](./14-type-targets.md)) |
| Establishing a target or taking a failure path | `ensure` / `else` |

Example:

```ft
is (ctx AppContext) LoggedIn {
    ensure ctx.session is Present()
    ensure ctx.user is Present()
}

def handler(ctx AppContext) {
    ensure ctx is LoggedIn() else Unauthorized()

    // Compiler knows LoggedIn(ctx).
    // It may additionally know ctx.session and ctx.user are Present.
}
```

---

## 3. Accepted surface syntax

There are three canonical forms of `ensure`. The thing after `is` is a **refinement target**: a **type** or an **assertion** ([14](./14-type-targets.md)).

### 3.1 Target with contextual/default failure

```ft
ensure x is Valid()
```

Meaning:

> `Valid(x)` must hold. If it does not, apply the failure behavior defined by the current context.

The exact available default behavior may differ by context, such as:

- `main`;
- tests;
- guard evaluation;
- other explicitly supported contexts.

Do not infer a new generic failure mechanism merely from this decision.

Today: `main` may omit a typed failure and use a failure block or process exit; type guards omit typed failure because a failed `ensure` makes the guard false; ordinary functions typically require an explicit failure strategy.

### 3.2 Assertion with typed failure

```ft
ensure x is Valid() else InvalidX()
```

This **replaces** the previous:

```ft
ensure x is Valid() or InvalidX()
```

`or` is **not** the failure keyword. Failure is `else`. `or` means assertion alternative (see §5).

`else` means:

> If the assertion fails, take this failure path.

Example:

```ft
ensure user is LoggedIn() else Unauthorized()
```

### 3.3 Assertion with custom failure block

```ft
ensure config is Valid() else {
    println("Invalid configuration")
}
```

The block executes **only when the assertion fails**.

It is a failure-handling block, not part of the assertion itself.

Internally, it should be treated and named accordingly, e.g.:

```text
FailureBlock
```

rather than merely "`EnsureBlock`".

Today the transformer already runs `EnsureNode.Block` on the failure path ([`transformEnsureStatement`](../../../../forst/internal/transformer/go/statement_ensure.go)). This RFC names that role and forbids combining it with typed `else`.

---

## 4. `else` is typed failure

Accepted failure syntax:

```ft
ensure user is LoggedIn() else Unauthorized()
```

Do **not** write failure with `or`:

```ft
ensure user is LoggedIn() or Unauthorized()
```

If `Unauthorized` is an error constructor, that line is **not** typed failure. It is assertion `or` (two alternatives). The checker must reject a non-constraint alternative and point at `else`.

Valid: assertion `or` plus typed `else`:

```ft
ensure phone is International() or National() else InvalidPhone()
```

The structure is visually clear:

```text
assertion:
    International OR National

failure:
    InvalidPhone
```

This is **not** the old failure syntax:

```ft
ensure phone is International() or InvalidPhone()
```

That now parses as two assertion alternatives (`International` or `InvalidPhone`). If `InvalidPhone` is an error constructor, not a constraint on `phone`, the checker rejects it and points at `else`.

---

## 5. `or` is assertion disjunction

`or` means **alternative constraints on the same place**.

It is not limited to type-guard declarations. It works wherever `is` takes a constraint: `ensure`, `if … is`, type-guard `ensure`.

Any number of alternatives:

```ft
ensure status is Pending() or Processing() or Retrying() else InvalidStatus()
```

Multiline, with `or` / `else` aligned, is allowed and preferred for long chains:

```ft
ensure value is String.Min(3).Max(32)
    or UUID.V4()
    or Slug.Valid()
    else InvalidValue()
```

Dot-chains on one alternative are **conjunction** (Meet). They bind tighter than `or`:

```text
(String.Min(3) AND Max(32))
OR UUID.V4()
OR Slug.Valid()
```

Same form in `if` and in guards:

```ft
if status is Pending() or Processing() {
    ...
}
```

```ft
is (task Task) Actionable {
    ensure task.status is Pending() or Processing()
}
```

### 5.1 One narrowable subject

All `or` alternatives refine **one** statically trackable place. The subject is written once, after `ensure` / `if`, before `is`.

Allowed places: identifiers and stable field/index paths.

```ft
ensure user is Active()
ensure ctx.user is Present()
```

Rejected: arbitrary expressions, calls, literals, arithmetic.

```ft
ensure getUser() is Active()    // invalid
ensure a + b is Positive()      // invalid
```

Rejected: repeating `is` with a second subject.

```ft
ensure value is VariantA() or value is VariantB()   // invalid
```

Write:

```ft
ensure value is VariantA() or VariantB()
```

### 5.2 Alternatives are constraints, not programs

Each alternative is a **constraint chain**: named guards and built-in constraints in call syntax `(…)`, optionally dotted.

```ft
ensure value is A() or B() else Error()
```

Not:

```ft
ensure value is A() or foo() == true else Error()
```

Not general boolean `or`.

### 5.3 Grammar

```text
ensure <place> is <constraint-chain> (or <constraint-chain>)* [else <error>]
```

The same `<place> is <constraint-chain> (or <constraint-chain>)*` appears in `if` / `else if`.

Every **assertion** in the chain uses call syntax with parentheses (`Pending()`, `Min(3)`, `Valid()`).

A bare type name after `is` is a **type target**, not a missing-paren assertion. Spec: [14](./14-type-targets.md). `or` does not join type names.

Core rule:

> Every successful `ensure` must attach useful narrowing information to one statically trackable value.

---

## 6. `|` is for types, not assertion disjunction

`|` remains the union-type operator:

```ft
type TaskStatus =
    "todo"
    | "in_progress"
    | "success"
    | "failed"
```

```ft
type Result = Success | Failure
```

Do **not** use `|` inside `is` assertions.

```ft
ensure status is Pending() | Processing() else InvalidStatus()   // invalid
```

Use `or`.

`|` is not general boolean OR in ordinary expressions (`foo() | bar()` stays invalid as a boolean). This RFC does not change that.

---

## 7. Conjunction remains sequential

Multiple assertions in a type guard mean all must succeed.

```ft
is (ctx AppContext) LoggedIn {
    ensure ctx.session is Present()
    ensure ctx.user is Present()
}
```

means:

```text
Present(ctx.session)
AND
Present(ctx.user)
```

There is no requirement to introduce an explicit `&` operator in this RFC.

Sequential `ensure` already expresses conjunction clearly.

---

## 8. Restricted `if` remains inside type guards

Type guards may use restricted conditional control flow.

Example:

```ft
is (message Message) Valid {
    if message.kind is Login() {
        ensure message.password is Present()
    } else if message.kind is Token() {
        ensure message.token is Present()
    }
}
```

This describes different valid structural cases.

The current Forst implementation already supports `if` inside type guards, subject to restrictions ([`infer_typeguard_node.go`](../../../../forst/internal/typechecker/infer_typeguard_node.go)).

This capability is retained.

---

## 9. Type-guard `if` remains analyzable

Inside a type guard, `if` conditions must remain assertions the compiler understands.

Allowed:

```ft
if message.kind is Login() {
    ...
}
```

Not accepted by this design:

```ft
if message.kind == "login" {
    ...
}
```

```ft
if canLogin(message) {
    ...
}
```

The type-guard language is deliberately smaller than ordinary Forst.

Allowed guard constructs should remain approximately:

- `ensure`;
- restricted `if ... is`;
- `else if`;
- `else`;
- comments;
- nested guard constructs that obey the same restrictions.

Do not allow arbitrary:

- loops;
- assignments;
- side effects;
- arbitrary boolean expressions;
- arbitrary function predicates;
- `return`.

Enforce these restrictions **recursively**, not only on the top-level statements of the guard body.

---

## 10. Type guards fail closed

A type guard succeeds only through a valid successful path.

Unmatched conditional branches must fail.

Given:

```ft
is (message Message) Valid {
    if message.kind is Login() {
        ensure message.password is Present()
    } else if message.kind is Token() {
        ensure message.token is Present()
    }
}
```

a message that is neither `Login` nor `Token` does **not** satisfy `Valid`.

The current implementation's unconditional successful fall-through must be removed ([`transformTypeGuard`](../../../../forst/internal/transformer/go/typeguard.go) currently emits a final `return true`).

Conceptually:

```go
func Valid(message Message) bool {
    if isLogin(message.Kind) {
        if !isPresent(message.Password) {
            return false
        }

        return true
    }

    if isToken(message.Kind) {
        if !isPresent(message.Token) {
            return false
        }

        return true
    }

    return false
}
```

A conditional guard without a matching successful branch fails.

---

## 11. Keep the optional failure block outside guards

The optional block is **not removed globally**.

It remains useful in ordinary functions and especially entry points such as `main`.

Example:

```ft
def main() {
    ensure config is Valid() else {
        println("Invalid configuration")
    }

    startServer(config)
}
```

The failure block can perform context-specific handling such as:

- printing diagnostics;
- logging;
- cleanup;
- changing exit presentation.

The language/runtime may then perform the context's standard termination behavior if appropriate.

Existing sugar such as `ensure !err else { … }` (implicit `Nil()`) remains a failure block on that assertion.

---

## 12. Failure blocks are forbidden inside type guards

This is invalid:

```ft
is (user User) LoggedIn {
    ensure user.session is Present() else {
        log("missing session")
    }
}
```

A type guard describes an invariant.

It must not contain arbitrary failure-handling programs.

Inside type guards, use only:

```ft
ensure user.session is Present()
```

The compiler must enforce this recursively.

This is important because the current parser allows an `ensure` block to contain a normal Forst block ([`parseEnsureBlock`](../../../../forst/internal/parser/ensure.go)), which can otherwise act as an escape hatch around the type-guard restrictions.

That loophole must be closed.

---

## 13. `else` and failure blocks are mutually exclusive

Do not support:

```ft
ensure x is Valid() else {
    log("invalid")
} else InvalidX()
```

An `ensure` statement chooses one failure mechanism:

### Typed failure

```ft
ensure x is Valid() else InvalidX()
```

or:

### Custom failure block

```ft
ensure x is Valid() else {
    handleInvalidX()
}
```

but not both.

This keeps the AST and semantics simple.

Conceptually:

```text
Ensure
├── assertion
└── failure
    ├── typed failure
    OR
    └── failure block
```

The current parser can attach both `Block` and `Error` ([`parseEnsureStatement`](../../../../forst/internal/parser/ensure.go)). This RFC forbids that.

---

## 14. Suggested AST shape

The AST should encode the semantic distinction directly.

Conceptually:

```go
type EnsureNode struct {
    Subject Node
    Target  RefinementTarget // TypeTarget | AssertionTarget ([14](./14-type-targets.md))

    Error        *Node
    FailureBlock *BlockNode
}
```

Invariant:

```text
Error != nil XOR FailureBlock != nil
```

when an explicit failure strategy exists.

(`ensure x is Valid()` with neither is allowed only where a **contextual default** applies: `main`, tests, type-guard evaluation.)

Avoid modelling the custom block as an arbitrary generic child of `ensure` without recording its semantic role.

Preferred internal terminology:

```text
FailureBlock
```

rather than:

```text
EnsureBlock
```

---

## 15. Assertions should have their own compiler representation

Do not treat assertion syntax merely as ordinary expressions with special cases scattered through the typechecker.

Introduce or converge toward a small assertion representation such as:

```go
type Assertion interface {
    assertion()
}

type Atom struct {
    // Known assertion
}

type Any struct {
    Children []Assertion
}

type All struct {
    Children []Assertion
}
```

For example:

```ft
x is A() or B()
```

becomes conceptually:

```text
Any(
    A,
    B,
)
```

Sequential guards can form:

```text
All(...)
```

without necessarily requiring explicit source syntax for `&`.

This assertion representation should be:

- statically analyzable;
- finite;
- side-effect free;
- directly lowerable to runtime checks.

Reuse it across `ensure`, `if … is`, type guards, and other assertion contexts.

The compiler does not convert arbitrary Forst AST into theorem-prover formulas.

---

## 16. Do not make arbitrary boolean expressions assertions

Do not introduce:

```ft
ensure quantity > 0 else InvalidQuantity()
```

as part of this RFC.

Do not introduce:

```ft
ensure canEdit(user, document) else Forbidden()
```

as a narrowing assertion merely because the function returns `Bool`.

A boolean result does not tell the compiler which structural/type facts become true.

If such behavior is desired later, it requires a separate design for explicitly declared predicate functions.

For now:

> `ensure` narrows through the assertion language, not arbitrary boolean programs.

---

## 17. Closed finite sets should be types

Do not use type guards to model ordinary enums or finite value domains.

Prefer:

```ft
type TaskStatus =
    "todo"
    | "in_progress"
    | "success"
    | "failed"
```

over:

```ft
is (status String) TaskStatus {
    ...
}
```

The rule is:

> Types define what may exist.

> Guards define additional invariants about existing values.

This is especially important for TypeScript generation.

A literal union naturally generates:

```ts
type TaskStatus =
    | "todo"
    | "in_progress"
    | "success"
    | "failed"
```

without requiring consumers to know anything about runtime guards.

---

## 18. Type guards are primarily nominal facts

Given:

```ft
is (password Password) Acceptable {
    ensure password is Strong() or Passkey()
}
```

after:

```ft
ensure password is Acceptable() else InvalidPassword()
```

the caller definitely knows:

```text
password satisfies Acceptable
```

The caller does **not** individually know:

```text
password satisfies Strong
```

or:

```text
password satisfies Passkey
```

because only one is required.

The guard name is an abstraction boundary.

Callers should not require the compiler to fully unfold every guard implementation.

---

## 19. Export only facts true on every successful guard path

Additional structural narrowing may escape from a guard when it is universally valid.

Example:

```ft
is (ctx AppContext) LoggedIn {
    ensure ctx.session is Present()
    ensure ctx.user is Present()
}
```

After:

```ft
ensure ctx is LoggedIn() else Unauthorized()
```

the compiler may know:

```text
LoggedIn(ctx)
Present(ctx.session)
Present(ctx.user)
```

because all three facts hold on every successful path.

For:

```ft
is (password Password) Acceptable {
    ensure password is Strong() or Passkey()
}
```

neither `Strong` nor `Passkey` is universally true.

Only:

```text
Acceptable(password)
```

must escape.

Rule:

> A fact may leave a guard only when it is true on every successful path.

---

## 20. Avoid mandatory DNF expansion

Do not require guard logic to be normalized into Disjunctive Normal Form.

For:

```text
(A | B) & (C | D)
```

retain structured representation:

```text
All
├── Any
│   ├── A
│   └── B
└── Any
    ├── C
    └── D
```

Do not eagerly expand it into:

```text
A&C
A&D
B&C
B&D
```

because combinations can grow exponentially.

Universal facts can instead be calculated structurally.

Conceptually:

```text
must(All(A, B)) =
    must(A) ∪ must(B)

must(Any(A, B)) =
    must(A) ∩ must(B)
```

This is sufficient for the refinement propagation required here.

Full DNF expansion is not part of the language semantics.

---

## 21. Mutation invalidates refinements; it does not forbid mutation

The rule in one sentence:

> A refinement remains valid until an operation may modify storage that fact depends on. The write stays legal. The fact is dropped.

Do **not** treat refined fields as implicitly borrowed immutable. Recovery is another `ensure` / guard, not a lock.

```ft
ensure user is Adult

user.age = 12

acceptAdult(user)   // error: Adult(user) is gone
```

```ft
ensure invoice is Payable

invoice.amount = discountedAmount

ensure invoice is Payable

pay(invoice)
```

Normative detail — dependency paths, overlap, aliases, Forst effect inference, collections, pointers, and untrusted Go — is [13](./13-refinement-stability.md). Do not implement from this section alone.

This problem is orthogonal to disjunction. Do not complicate the assertion algebra to “solve” aliasing.

---

## 22. Runtime-dependent values must not become static type parameters

Do not accidentally introduce dependent typing.

Static assertions using literals may be supported:

```ft
ensure password is Min(12)
```

But if:

```ft
minLength := loadMinimum()
ensure password is Min(minLength)
```

is supported as a runtime check, it must not automatically create a static type equivalent to:

```text
String.Min(minLength)
```

where the type depends on an arbitrary runtime value.

Runtime checks and static refinements do not always need identical expressiveness.

---

## 23. Domain types should own domain guards

Application guards should normally belong to nominal domain types.

Prefer:

```text
Password.Strong
Email.Valid
Task.Actionable
AppContext.LoggedIn
```

rather than accumulating arbitrary application concepts on:

```text
String.Strong
String.Email
String.Slug
String.Phone
String.Postcode
...
```

Built-in primitives may continue to expose language-defined constraints such as:

```text
String.Min
String.Max
Int.Min
```

Application semantics should normally belong to application-defined types.

Forst therefore needs a clear distinction between:

```text
alias
```

and:

```text
new nominal/defined type
```

analogous in purpose to Go's:

```go
type A = string
```

versus:

```go
type A string
```

An alias must not create nominal ownership.

---

## 24. Relationship to arbitrary business logic

Not every shared condition belongs in the type system.

Examples:

```text
user may currently edit document
customer is under today's rate limit
stock is currently available
subscription has been paid
offer has not expired
```

may depend on:

- several values;
- external state;
- databases;
- time;
- authorization context.

These can still have a single canonical implementation as ordinary functions.

Avoiding duplicate logic does not require every rule to become a type.

The refinement system should focus on stable facts that are useful and sound to carry forward.

---

## 25. Implementation order

Implement in approximately this order.

### Phase 1 — Syntax and soundness fixes

1. Rename failure `ensure ... or ...` to `ensure ... else ...`.
2. Parse `or` as assertion alternatives on one place (`ensure`, `if … is`, type guards). Parse a bare type name as a **type target** ([14](./14-type-targets.md)). Reject `|` in `is` assertions. Reject non-place subjects and non-constraint alternatives. `or` does not join type names.
3. Make guard fall-through fail closed.
4. Forbid failure blocks inside type guards.
5. Enforce guard restrictions recursively.
6. Make typed failure and custom failure blocks mutually exclusive.

### Phase 2 — Assertion representation

7. Introduce/clean up a dedicated assertion IR (`Any` / `All` / `Atom`).
8. Keep **TypeTarget** distinct from assertion IR ([14](./14-type-targets.md)). Do not encode `ensure x is ActiveStatus` as `Atom(ActiveStatus())`.
9. Represent conjunction structurally where useful as `All`.
10. Reuse assertion IR across:
    - `ensure` assertion targets;
    - `if ... is` assertion targets;
    - type guards;
    - other assertion contexts.

### Phase 3 — Type system

11. Add literal union types.
12. Add/refine nominal defined types if necessary.
13. Record named guard refinements.
14. Compute universally true exported facts.
15. Add correct narrowing for assertion `or` (`Any`) and for type `|`.
16. Type-target membership and narrowing for closed enum/literal subset types ([14](./14-type-targets.md)). No implicit `Type()` assertions. No `or` of type names. No general structural membership.

### Phase 4 — Mutation safety

Specified in [13 §33](./13-refinement-stability.md). Plan: [phase 4 hub](../../../../.plans/analyzable-refinements/phase-4-mutation-safety.md). Tests: [`.plans/analyzable-refinements/tests/`](../../../../.plans/analyzable-refinements/tests/README.md).

17. Facts with dependency paths (4a), including type-target facts ([14](./14-type-targets.md)).
18. Direct local mutation, merge, loops (4b). Type-target facts drop like other facts.
19. Infer Forst write summaries (4c).
20. Basic alias tracking (4d).
21. Collections, coarse (4e).
22. Untrusted Go (4f).
23. Escapes and concurrency (4g).
24. Precision later only after 17–23 are sound.

---

## 26. Required tests

At minimum, add tests for:

### Typed failure

```ft
ensure x is A() else Error()
```

### Assertion disjunction

```ft
ensure x is A() or B() else Error()
```

### Three alternatives, multiline

```ft
ensure value is String.Min(3).Max(32)
    or UUID.V4()
    or Slug.Valid()
    else InvalidValue()
```

### Assertion disjunction in `if`

```ft
if x is A() or B() {
    ...
}
```

### Assertion disjunction inside guards

```ft
is (x X) Valid {
    ensure x is A() or B()
}
```

### Type target ([14](./14-type-targets.md))

```ft
ensure status is ActiveStatus else InvalidStatus()
```

Bare type name. No parens. Catalog: `phase-3/11`–`19`, `phase-1/44`.

### Type target is not assertion `or`

`ensure status is Status.Pending or Status.Processing` fails compilation. `ensure status is Pending()` does not synthesize an enum assertion.

### Place subject only

`ensure getUser() is Active()` and `ensure a + b is Positive()` fail compilation.

### `|` is not assertion `or`

`ensure x is A() | B()` is a parse/type error pointing at `or`.

### Old `or` failure is not failure

`ensure x is A() or Error()` is disjunction, not typed failure. If `Error` is not a constraint on `x`, reject and suggest `else`.

### Sequential conjunction

```ft
is (x X) Valid {
    ensure x is A()
    ensure x is B()
}
```

### Restricted conditional guards

```ft
is (x X) Valid {
    if x.kind is A() {
        ...
    } else if x.kind is B() {
        ...
    }
}
```

### Unmatched conditional failure

Ensure an unmatched guard branch returns false.

### Failure block in ordinary function

```ft
ensure x is A() else {
    ...
}
```

### Failure block in `main`

Verify current intended entry-point behavior remains supported.

### Failure block rejected inside guard

```ft
is (x X) Valid {
    ensure x is A() else {
        ...
    }
}
```

must fail compilation.

### `else` plus block rejected

```ft
ensure x is A() else {
    ...
} else Error()
```

must fail compilation.

### Branch fact leakage

For:

```ft
is (x X) Valid {
    ensure x is A() or B()
}
```

the caller must not individually assume `A` or `B`.

### Universal fact propagation

For:

```ft
is (x X) Valid {
    ensure x.foo is Present()
    ensure x.bar is Present()
}
```

both facts should be available after ensuring `Valid`, if structural fact propagation is implemented.

### Mutation invalidation

See [13 §34](./13-refinement-stability.md) and the TDD catalog [`.plans/analyzable-refinements/tests/`](../../../../.plans/analyzable-refinements/tests/README.md).

---

## 27. Non-goals

This decision explicitly does not introduce:

- arbitrary boolean `ensure`;
- arbitrary boolean type guards;
- theorem proving;
- SMT solving;
- general predicate implication;
- dependent types;
- arbitrary runtime values as type parameters;
- unrestricted side effects inside guards;
- loops inside guards;
- assignments inside guards;
- `return` inside guards;
- failure-handling blocks inside guards;
- DNF normalization as a semantic requirement;
- guards as a replacement for literal unions;
- type-level encoding of all business logic;
- `|` as general boolean OR in ordinary expressions;
- `or` as general boolean OR in ordinary expressions;
- `|` as assertion disjunction;
- combining typed `else` with a failure block on one `ensure`;
- implicit immutability / refinement borrows / a user-facing borrow checker ([13](./13-refinement-stability.md));
- required `mutates(...)` annotations in source.

If a future feature requires one of these, it requires a separate language-design decision rather than being considered a natural extension of `ensure`.

---

## 28. Final language rule

The intended mental model is:

> **Types define valid data.**

> **Type guards name reusable invariants.**

> **`or` expresses alternative constraints on one place.**

> **`|` expresses alternative types (including literal unions).**

> **`ensure` establishes a refinement against a type or an assertion** ([14](./14-type-targets.md)).

> **A type target uses compiler membership and has no parentheses.**

> **An assertion target uses `()` and may use `or`.**

> **`else` specifies typed failure.**

> **An `ensure … else { ... }` block performs custom failure handling outside type guards.**

> **After a successful `ensure`, the compiler remembers only facts it can establish soundly.**

> **Those facts die if their storage may have changed** ([13](./13-refinement-stability.md)). Mutation stays legal.

This is the accepted direction for the RFC.

---

## 29. Decision

Adopt the analyzable-refinement direction with the following modifications to the original RFC recommendation ([06](./06-recommendation.md)) and to the first draft of this file:

1. Keep restricted `if is` inside type guards.
2. Make unmatched guard branches fail.
3. Add `or` to the assertion algebra, including call-site `ensure` and `if … is`. Same place; constraint chains only.
4. Replace `ensure … or <error>` with `ensure … else <error>`. `or` is assertion alternative, not failure.
5. Keep failure blocks for ordinary functions and `main`; they run **on failure** only.
6. Forbid failure blocks inside type guards; enforce guard restrictions recursively.
7. Typed `else` and a failure block are mutually exclusive.
8. Add literal unions for closed finite sets.
9. Prefer nominal domain types over user guards on primitives.
10. Treat a guard name as the primary refinement visible to callers.
11. Export additional structural facts only when they hold on every successful guard path.
12. Do not require DNF normalization.
13. Do not allow runtime-dependent guard parameters to create dependent static types.
14. Refinement stability under mutation, aliasing, and Go interop is [13](./13-refinement-stability.md): relevant mutation **discards facts**; it does not forbid the mutation. No ownership model.
15. Keep contextual business predicates as ordinary functions when they cannot produce stable, analyzable refinement facts.
16. Type targets ([14](./14-type-targets.md)): `ensure place is Type` (no parens) for closed enum/literal subsets. Enum variants are not assertions. `or` does not join types.

This preserves the original goal—eliminating repeated conditions by giving shared invariants one definition—without making Forst's typechecker responsible for understanding arbitrary programs.
