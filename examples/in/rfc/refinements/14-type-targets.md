# 14 — Type targets in `ensure`

**Status:** **Accepted, amended.** Normative for the RHS of `ensure … is` / `if … is`: a **refinement target** is a **type** or an **assertion**. Types and assertions stay distinct. “Place” in source grammar is represented by compiler `AccessPath`; runtime targets also include analyzable nominal scalar constraints. See [the locked technical design](../../../../.plans/analyzable-refinements/TECHNICAL-DESIGN.md).

**Scope:** Type-target syntax (no parentheses), runtime membership for closed enum/literal subset types, analyzable constrained scalar aliases (`String.Min…`), and bare nominal scalar domains, narrowing to that type. Assertion `or`, `else`, guards: [12](./12-accepted-decision.md). Mutation of type-target facts: [13](./13-refinement-stability.md).

**Does not reopen:** [12](./12-accepted-decision.md), [13](./13-refinement-stability.md). Does not make every Forst type runtime-checkable. Does not turn enum variants into assertions.

**Implementation plan:** [`.plans/analyzable-refinements`](../../../../.plans/analyzable-refinements/README.md) — parse in phase 1, keep TypeTarget distinct in phase 2, membership + narrowing in phase 3, invalidation in 4a–4b.

**Spec tests:** [`.plans/analyzable-refinements/tests/`](../../../../.plans/analyzable-refinements/tests/README.md) (`phase-1/26`, `39`, `41`, `44`; `phase-2/07`; `phase-3/11`–`19`; `phase-4a/10`; `phase-4b/27`). Coverage: [`COVERAGE.md`](../../../../.plans/analyzable-refinements/tests/COVERAGE.md).

---

## Summary

`ensure` currently serves as the main mechanism for proving that a value satisfies some condition and then narrowing that value.

This RFC generalizes the right-hand side of:

```ft
ensure <place> is <target>
```

from “assertion only” to a broader **refinement target**.

A refinement target may be:

```text
RefinementTarget
├── Type
└── Assertion
```

Types and assertions remain distinct concepts.

This allows `ensure` to narrow enum values to named subset types without turning enum members or enum subsets into assertions.

---

## Motivation

Forst should support this:

```ft
type ActiveStatus =
    Status.Pending
    | Status.Processing
    | Status.Retrying

ensure status is ActiveStatus else InvalidStatus()
```

After success:

```text
status: ActiveStatus
```

The alternative would be to make enum members behave like assertions:

```ft
ensure status is Pending() or Processing() or Retrying()
```

This is rejected.

Enum variants are values/types within a closed domain. They should not become part of the assertion system merely to make `ensure` ergonomic.

At the same time, requiring ordinary branching would lose the main benefit of `ensure`:

```ft
if status ... {
    ...
}
```

does not express the intent as directly and risks duplicating the same subset definition across the program.

A named subset type gives the condition one canonical definition while `ensure` provides the narrowing operation.

---

## Proposal

The grammar should conceptually treat the RHS of `is` as a refinement target:

```text
Ensure :=
    "ensure" Place "is" RefinementTarget Failure?

RefinementTarget :=
    TypeTarget
    | AssertionTarget
```

The two target kinds have different semantics.

---

## Type targets

A type target performs compiler-defined runtime membership checking and narrows the subject to that type.

Example:

```ft
type ActiveStatus =
    Status.Pending
    | Status.Processing
    | Status.Retrying

ensure status is ActiveStatus else InvalidStatus()
```

`ActiveStatus` is a type, therefore it does not use parentheses.

No implicit assertion named `ActiveStatus()` is created.

No assertions named `Pending()`, `Processing()`, or `Retrying()` are created.

The enum/type system remains independent from the assertion system.

---

## Assertion targets

Existing guards and constraints remain assertion targets.

All assertions continue to use parentheses:

```ft
ensure password is Strong() else WeakPassword()
```

Chained constraints remain supported:

```ft
ensure username is String.Min(3).Max(32)
    or GeneratedUsername.Valid()
    else InvalidUsername()
```

Assertion disjunction remains restricted to assertions over the same narrowable subject.

---

## Syntactic distinction

The difference between types and assertions is intentionally visible:

```ft
ensure status is ActiveStatus else InvalidStatus()
//               ^ type

ensure password is Strong() else WeakPassword()
//                 ^ assertion
```

This preserves the existing decision that every assertion ends in parentheses.

It also avoids pretending that a type is executable predicate syntax.

---

## Initial scope

Type-target narrowing is required for:

1. Closed enum subset / literal union types
2. Analyzable constrained scalar aliases (`type Sku = String.Min(1).Max(64)`)
3. Bare nominal scalar domains (`type Password = String`)

Example (literal union):

```ft
enum Status {
    Pending
    Processing
    Retrying
    Done
    Failed
}

type ActiveStatus =
    Status.Pending
    | Status.Processing
    | Status.Retrying
```

The compiler can generate a finite runtime membership check for `ActiveStatus`.

Example (constrained scalar alias):

```ft
type Sku = String.Min(1).Max(64)

ensure sku is Sku else BadSku()
```

The compiler expands the typedef’s Meet chain for the runtime check and narrows the subject to `Sku`.

This RFC does not define general runtime membership for every Forst type.

In particular, it does not automatically imply support for:

```ft
ensure value is ArbitraryStructuralType
```

Runtime membership semantics for additional type categories must be defined separately before those types become valid `ensure` targets.

---

## Narrowing semantics

A successful type-target `ensure` narrows the tracked place to the target type.

```ft
ensure status is ActiveStatus else InvalidStatus()

consumeActiveStatus(status)
```

The compiler may treat `status` as `ActiveStatus` after the `ensure`.

The same mutation and invalidation rules that apply to other refinements also apply here.

If the tracked place is mutated incompatibly, the refinement is lost.

---

## Relationship to assertion disjunction

Type unions and assertion alternatives remain separate mechanisms.

For enum subsets, prefer a named type:

```ft
type ActiveStatus =
    Status.Pending
    | Status.Processing
    | Status.Retrying

ensure status is ActiveStatus else InvalidStatus()
```

Do not introduce:

```ft
ensure status is Status.Pending or Status.Processing or Status.Retrying
```

as an assertion feature.

`or` remains assertion disjunction:

```ft
ensure value is A() or B() or C() else Error()
```

This preserves a clean boundary:

```text
type union
    → defines a reusable set of values

assertion `or`
    → defines alternative runtime refinements of one value
```

---

## Compiler model

The compiler should represent the distinction explicitly.

Conceptually:

```text
Ensure
├── Place
├── RefinementTarget
│   ├── TypeTarget
│   └── AssertionTarget
└── Failure
```

### `TypeTarget`

The compiler:

1. verifies that the type supports runtime membership testing;
2. emits the appropriate runtime membership check;
3. narrows the place to the target type on success.

### `AssertionTarget`

The compiler:

1. evaluates the existing assertion/guard representation;
2. emits its runtime checks;
3. records the resulting assertion and shape facts.

The two paths may converge in flow narrowing afterward, but they are not the same semantic mechanism.

---

## Design rule

The meaning of:

```ft
ensure value is Target
```

is:

> Prove that the tracked value belongs to `Target`; otherwise take the failure path.

How membership is proven depends on the target:

- a type uses compiler-defined type membership;
- an assertion uses assertion/guard semantics.

`ensure` is therefore the common narrowing operation.

Types and assertions remain separate sources of refinement.

---

## Non-goals

This RFC does not:

- turn enum variants into assertions;
- synthesize guards for enum values;
- make all Forst types runtime-checkable;
- allow arbitrary expressions as refinement targets;
- merge the type system with the assertion system;
- change assertion parenthesis rules;
- replace named enum subset types with repeated `or` expressions.

---

## Decision

Adopt type targets as part of the current refinement redesign.

Initial implementation should support named closed enum subset / union types such as:

```ft
type ActiveStatus =
    Status.Pending
    | Status.Processing
    | Status.Retrying

ensure status is ActiveStatus else InvalidStatus()
```

while preserving ordinary assertion narrowing:

```ft
ensure username is String.Min(3).Max(32)
    or GeneratedUsername.Valid()
    else InvalidUsername()
```

The central abstraction is:

> `ensure` performs narrowing against a refinement target; a refinement target may be a type or an assertion, but types and assertions remain distinct language concepts.
