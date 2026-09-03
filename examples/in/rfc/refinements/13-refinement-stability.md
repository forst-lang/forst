# 13 — Refinement stability: mutation, aliasing, and Go interop

**Status:** **Accepted.** This is the normative decision for how established refinements survive or die after writes.

**Scope:** Invalidation of flow-sensitive facts after mutation, reassignment, aliasing, function calls, collections, pointers, and Go interop.

**Does not reopen:** [12](./12-accepted-decision.md), [14](./14-type-targets.md). `ensure`, `else`, assertion `or`, type targets, failure blocks, `|` in types, analyzable guards, `must()`, and literal unions stay as 12 and 14 specified them.

**Implementation plan:** [`.plans/analyzable-refinements`](../../../../.plans/analyzable-refinements/README.md), phase 4.

**Source handoff:** [`.plans/analyzable-refinements/refinement_stability_handoff.md`](../../../../.plans/analyzable-refinements/refinement_stability_handoff.md). This file is the normative write-up of that handoff. Spec tests: [`.plans/analyzable-refinements/tests/`](../../../../.plans/analyzable-refinements/tests/README.md).

---

## 1. The problem

`ensure` and guards produce facts the compiler is allowed to remember ([12](./12-accepted-decision.md)). Those facts are about **current storage**, not about a variable name forever.

```ft
ensure user is Adult

user.age = 12

acceptAdult(user)
```

If `Adult` depends on `user.age >= 18`, the compiler must not still treat `user` as `Adult`.

The same hole appears through an alias:

```ft
other = user

ensure user is Adult

other.age = 12

acceptAdult(user)
```

through a Forst call:

```ft
ensure user is Adult

modify(user)

acceptAdult(user)
```

and through imported Go:

```ft
ensure user is Adult

goPackage.Modify(user)

acceptAdult(user)
```

---

## 2. Design thesis

Forst is not an ownership language. This RFC does not add a borrow checker, lifetimes, move checking, or user-facing `mut` / `ref` / `borrow` syntax.

Refinements are **flow-sensitive facts**. A fact stays usable only while the storage it depends on is known not to have changed in a way that could falsify it.

> Relevant mutation **invalidates** a refinement. It does **not** normally make the mutation illegal.

The programmer re-establishes the fact with `ensure`, a guard, or another narrowing operation.

That is the intended source shape:

```ft
ensure invoice is Payable

invoice.amount = discountedAmount

ensure invoice is Payable

pay(invoice)
```

The source says: the invariant held; something that affects it changed; it was checked again.

---

## 3. Decisions from 12 this RFC preserves

Do not reopen these while implementing stability.

| Decision | Rule |
| --- | --- |
| `ensure` establishes type facts | Successful `ensure` is a runtime assertion **and** a compile-time proof ([12 §1–2](./12-accepted-decision.md)) |
| Typed failure is `else` | `ensure user is Adult() else InvalidAge()`. `or` is assertion alternative ([12 §3–5](./12-accepted-decision.md)) |
| Failure blocks | Ordinary functions and `main` only; not inside type/shape guards ([12 §3.3, §13](./12-accepted-decision.md)) |
| `or` in assertions | Same place; constraint chains; compound facts invalidate as a whole ([12 §5](./12-accepted-decision.md)) |
| Shape guards keep their name and role | They produce facts; mutation can drop those facts ([12](./12-accepted-decision.md), [guard RFC](../guard/guard.md)) |
| Type targets | `ensure status is ActiveStatus` (no parens) is a type fact; same drop rules ([14](./14-type-targets.md)) |

Because guard bodies are compiler-analyzable, Forst can determine which values an established refinement depends on. That is the foundation of this RFC.

---

## 4. Programming model

Unrelated writes must not destroy a proof:

```ft
ensure order is Shippable

order.lastViewedAt = now

ship(order)
```

If `lastViewedAt` is not in `Shippable`’s dependency set, `Shippable` remains.

Relevant writes destroy the proof, not the assignment:

```ft
ensure order is Shippable

order.address = newAddress

ship(order)   // error: order is no longer known to be Shippable
```

Re-establish explicitly:

```ft
ensure order is Shippable

order.address = newAddress

ensure order is Shippable

ship(order)
```

The compiler does **not** require re-narrowing immediately. This is fine:

```ft
ensure user is Adult

user.age = inputAge

print(user.name)
```

Only later code that **requires** `Adult` fails:

```ft
acceptAdult(user)
```

---

## 5. Primary semantic rule

> A refinement remains valid until an operation **may modify** any value on which that refinement depends.

When that happens:

> The affected refinement is **removed** from the current flow environment.

The mutation itself is normally legal.

Soundness beats precision. A false invalidation costs another `ensure`. A false retention is a type-system hole.

---

## 6. Facts have dependencies

Do **not** model

```text
user: Adult
```

as an unconditional replacement of `user`’s ordinary type.

Model something equivalent to:

```text
Fact:         Adult(user)
Dependencies: user.age
```

```ft
is (user User) Adult {
    ensure user.age >= 18
}
```

Successful narrowing establishes `Adult(user)` depending on `user.age`.

```ft
user.name = "Alice"   // disjoint → retain Adult
user.age = 12         // intersects → discard Adult
```

The carrier type of `user` does not change. The **fact** does.

This is the same split [SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) already draws: lattice ops on types stay pure; memory effects live in the fact layer. Invalidation must not be folded into `Meet` / `Join`.

---

## 7. Relational refinements

Guards may relate fields or separate values.

```ft
is (period Period) ValidPeriod {
    ensure period.start < period.end
}
```

```text
Fact:         ValidPeriod(period)
Dependencies: period.start, period.end
```

Either write drops the fact. `period.description = "Vacation"` does not.

Across values:

```ft
ensure withdrawal is AllowedFor(account)
```

If the condition depends on `withdrawal.amount`, `account.balance`, and `account.status`, a write to any of them drops `AllowedFor`.

```ft
ensure withdrawal is AllowedFor(account)

account.balance -= fee

ensure withdrawal is AllowedFor(account)

execute(withdrawal, account)
```

---

## 8. Dependency paths

Dependencies are **value paths** (or an equivalent place abstraction). Observable examples:

```text
user
user.age
user.address
user.address.country
order.items
order.items[*].price
account.balance
```

The compiler representation is an implementation detail. Observable behavior is not:

- a write to a path **equals** a dependency → invalidate
- a write to a **prefix** of a dependency → invalidate (replacing the parent replaces the child)
- a write to a **disjoint sibling** → retain
- a write through an **index/wildcard** that covers a dependency → invalidate

---

## 9. Mutation overlap

Given dependency `user.address.country`:

| Write | Effect |
| --- | --- |
| `user.address.country = …` | invalidate |
| `user.address = newAddress` | invalidate (contains the dependency) |
| `user = anotherUser` | invalidate |
| `user.address.street = "Main Street"` | retain, if `street` cannot modify `country` |

Conceptually:

```text
write(path) overlaps dep
    iff path is a prefix of dep, or dep is a prefix of path, or they are equal
        under the collection-wildcard rules in §11
```

Sibling fields of a struct are disjoint when the compiler knows the write cannot tear through to the sibling. If it cannot know that (opaque Go type, interface, `any`), treat the write as overlapping the whole reachable root.

---

## 10. Reassignment is mutation

There is no philosophical split between “mutating an object” and “rebinding a variable.” What matters is whether the fact can still be true of the storage now named.

```ft
ensure user is Adult

user = anotherUser   // discard Adult(user)
```

```ft
ensure period.start < period.end

period.start = otherStart   // discard the relational fact
```

---

## 11. Compound assertions

The dependency set of a logical fact is the **union** of the dependencies needed to establish it.

```ft
ensure user is Adult() or Admin()
```

depends on every path either alternative reads (here typically `user.age` and `user.role`).

If the assertion succeeded because age was 25 and role was `User`, then `user.age = 10` does **not** let the compiler conclude that the other disjunct now holds. The whole compound fact dies.

More clever proof retention (keeping a disjunct that was independently still known) is **out of scope**. Conservative union of dependencies is the rule.

`must()` export from [12 §20](./12-accepted-decision.md) does not change this. Exported conjuncts are separate facts, each with its own dependency set. A write that hits `user.email` drops `Present(user.email)` and does not have to drop an unrelated `Present(user.session)` on the same root.

---

## 12. Shape guards

Shape guards use the same mechanism. They are not renamed and not special-cased into a second invalidation system.

```ft
ensure response is {
    user: {
        id: UUID
        email: Email
    }
}
```

Possible facts:

```text
response.user exists
response.user.id is UUID
response.user.email is Email
```

| Write | Effect |
| --- | --- |
| `response.metadata = metadata` | retain the user facts |
| `response.user.email = input` | drop at least the email fact |
| `response.user = anotherUser` | drop every fact under `response.user` |

Prefer **fact-level** invalidation over discarding every fact on the root object.

---

## 13. Do not introduce implicit immutability

This is rejected:

```ft
ensure user is Adult

user.age = 12
# compile error because Adult depends on user.age
```

That would treat refinement dependencies as implicitly borrowed immutable. Ordinary business code becomes illegal:

```ft
ensure invoice is Payable

invoice.amount = applyDiscount(invoice.amount)
```

Forst would then owe the programmer answers about when the borrow ends, explicit drops, lexical vs inferred lifetimes, nested borrows, alias borrows, mut vs immut borrows, escaped references, and calls while borrowed.

Those are ownership-language problems. Forst does not need them for refinement soundness.

User-facing docs that currently hint at “scoped immutability for `ensure`” are wrong relative to this decision. Recovery is re-narrowing, not locking.

---

## 14. Re-narrowing is the recovery mechanism

Relevant mutation destroys **knowledge**. It does not forbid the write.

`ensure` is the boundary between “possibly invalid state” and “state whose invariant has been re-established.”

That matches the broader Forst goal: shared conditions live in the type/refinement system instead of being copied through business code.

---

## 15. Direct aliasing

Path overlap is not enough if two names share storage.

```ft
alias = user

ensure user is Adult

alias.age = 12

acceptAdult(user)
```

If `alias` and `user` may refer to the same mutable storage, `Adult(user)` dies.

The compiler uses a **conservative may-alias** model. It does not need to prove precise identity. It answers:

> Could this write affect storage on which an active refinement depends?

If yes, invalidate.

### 15.1 Value copies do not alias

Forst lowers to Go. Go copies structs, arrays, and primitive values on assignment.

```ft
other = user   // User is a struct with no interior references
other.age = 12 // does not modify user
```

If the compiler **proves** the assignment is a pure value copy (no pointers, slices, maps, chans, interfaces, or funcs in the copied type), it must **not** treat `other` as an alias of `user`.

If it cannot prove that, it must treat the names as may-aliases.

### 15.2 Pointers, slices, maps, and interior references do alias

```ft
p = &user
ensure user is Adult
p.age = 12          // invalidates Adult(user)
```

```ft
items = order.items
ensure order is Priced
items[0].price = 0  // invalidates if Priced depends on items[*].price
```

Taking the address of a refined place, or slicing it, creates a may-alias.

### 15.3 No general borrow checker

Do not expose alias analysis as ownership. Forst does not need:

- ownership transfer
- borrow lifetimes
- mutable vs shared borrows as a user concept
- lifetime annotations
- move checking
- borrow regions

The only compiler question is: **may this operation mutate storage supporting an active fact?**

---

## 16. Function calls need write summaries

```ft
ensure user is Adult

normalize(user)

acceptAdult(user)
```

Whether `Adult` survives depends on what `normalize` may write.

```text
normalize(user)
writes: user.name     → Adult(user) retained if it depends only on user.age
writes: user.*        → Adult(user) discarded
```

---

## 17. Forst effect summaries are inferred, not declared

Ordinary Forst functions must not require:

```text
mutates(user.name)
```

Infer from the body:

```ft
def rename(user, name) {
    user.name = name
}
```

```text
rename: writes arg0.name
```

Then:

```ft
ensure user is Adult

rename(user, "Alice")

acceptAdult(user)   // Adult retained
```

Effects propagate through calls. Recursion uses a fixpoint. That is an implementation concern, not a source-language feature.

Unknown, unanalyzable, or effect-inference failure → treat as unknown (§19).

---

## 18. Returned aliases

Writes are not the only effect. A function may return a name into argument storage.

```ft
address = getAddress(user)
address.country = "DE"   // may affect facts about user.address.country
```

```text
getAddress(arg0)
returns alias of: arg0.address
```

Infer this for Forst functions when the return is a field, slice, or address of an argument. If provenance is unclear, conservative **whole-root** aliasing of every mutable argument is allowed.

Correctness beats maximal narrowing retention.

---

## 19. Unknown effects fail closed

If the compiler cannot determine whether a call can mutate relevant storage, **invalidate**.

```ft
ensure user is Adult

unknownOperation(user)

acceptAdult(user)   // Adult is gone unless proven otherwise
```

A false negative here is unsound. A false positive is another `ensure`.

---

## 20. Collections

Mutation can go through elements, headers, or aliases.

```ft
ensure users is AllAdults
```

If the guard depends on `users[*].age`:

```ft
users[0].age = 12     // invalidate
users.append(child)   // invalidate if membership or length is in the fact
users = other         // invalidate
```

Conceptual paths:

```text
users.length
users[*]
users[*].age
```

**v1 may be coarse.** Treating any element write, append, slice header write, or index write as overlapping `coll[*]` and `coll.length` is acceptable. Per-index precision (`users[0]` vs `users[1]`) is not required.

Maps: a write to `m[k]` overlaps `m[*]` and, if the fact cares about keys, `m`.

---

## 21. Local values versus aliased mutable storage

Local primitives have trivial stability. Only reassignment of that binding can drop the fact:

```ft
age = 25
ensure age >= 18
age = 10          // fact gone
```

No alias machinery is required. Ordinary scalar refinements must stay cheap.

Pointers, slices, maps, and interior references are the cases that need may-alias (§15, §26).

### 21.1 Pointer identity vs pointee

`Present` / `Nil` on a pointer field depends on that field’s pointer identity, not on the pointee’s contents, unless the guard also reads the pointee.

```ft
ensure ctx.session is Present()

ctx.session = Nil     // drop Present
*ctx.session = …      // does not by itself drop Present(session)
```

A guard that reads through the pointer (`ensure ctx.session.userId is UUID`) depends on the pointee path. Writing `*ctx.session` or replacing `ctx.session` overlaps it.

Clearing a pointer **must** drop `Present` on that path.

---

## 22. Control-flow merging

A fact is available after a merge only if it remains established on **every** incoming path. Same policy as [SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md). Do not invent a second join lattice.

```ft
ensure user is Adult

if condition {
    user.name = "Alice"
} else {
    user.name = "Bob"
}

acceptAdult(user)   // ok: neither branch touches age
```

```ft
ensure user is Adult

if condition {
    user.age = 12
} else {
    user.name = "Bob"
}

acceptAdult(user)   // error: one path dropped Adult
```

---

## 23. Loops

Same overlap rule. If the loop body cannot write a dependency, the fact may survive the loop:

```ft
ensure user is Adult

while condition {
    user.name = nextName()
}

acceptAdult(user)   // ok
```

If the body may write a dependency, the fact is gone after the loop (and cannot be assumed on a back-edge):

```ft
ensure user is Adult

while condition {
    user.age = nextAge()
}

acceptAdult(user)   // error
```

The compiler does not re-prove the guard on every iteration unless code inside the loop needs the fact. `ensure` inside a loop re-establishes for the rest of that iteration until the next overlapping write.

Do **not** require dropping every fact at loop entry. That is coarser than this RFC.

---

## 24. Closures

Creating a closure does **not** by itself mutate captured storage.

```ft
ensure user is Adult

callback = () => {
    user.age = 12
}

acceptAdult(user)   // still Adult: callback has not run
```

Calling it does:

```ft
callback()
acceptAdult(user)   // Adult gone
```

If the closure **escapes** (passed to unknown code, stored where it may run later, sent through a channel), treat captured mutable places as escaped (§25). Conservative v1: passing or storing a closure that captures a refined mutable place invalidates those facts, or poisons the places for the rest of the function.

Use the same effect/alias model. No separate “closure borrow” feature.

---

## 25. Concurrency

Forst compiles to Go. This RFC does not solve data races. It must not keep a fact whose storage may now change on another goroutine.

```ft
ensure user is Adult

go modify(user)

acceptAdult(user)   // Adult gone: mutable state escaped concurrently
```

Invalidate **immediately** when mutable state supporting a fact escapes to concurrently executing code.

Channel send is the same if it exposes a mutable alias:

```ft
ensure user is Adult

channel.send(user)

acceptAdult(user)   // Adult gone if user is reference-bearing
```

A proven copy of a pure value through a channel need not invalidate the original.

Effect summaries may record `spawns_with` / `escapes` (§30). Sequential `go` and channel send are **stage 6** ([§33](#33-implementation-stages)).

---

## 26. Go interoperability

Go has no const references, purity annotations, effect system, or ownership. Forst cannot assume a Go call preserves active refinements.

Unknown Go mutation **invalidates affected facts**. It does **not** reject the program.

Go degradation:

```text
known non-mutating behavior     → retain
known unrelated mutation        → retain
known relevant mutation         → invalidate
unknown mutation capability     → invalidate
```

### 26.1 Primitives and strings

`int`, `int64`, `float64`, `bool`, and other scalars passed by value cannot mutate the caller’s original binding. Facts that depend only on that copied scalar survive the call.

Go strings have immutable contents. A Go function receiving a `string` cannot mutate the logical string the caller still holds. Facts on that string value may survive.

### 26.2 Pointers

Passing `&user` or a `*T` that reaches a dependency, without a trustworthy summary, drops overlapping facts.

Pointer-receiver methods (`func (u *User) Normalize()`) may mutate the receiver until a summary proves otherwise.

### 26.3 Structs passed by value

A Go struct by value copies top-level fields. The callee cannot replace the caller’s top-level fields.

The copy still shares **reference-bearing** fields (pointers, slices, maps, chans, interfaces, funcs). The callee may mutate that reachable storage.

Forst reasons about **reachable mutable storage**, not only whether the parameter is `*T`.

```go
type User struct {
    Name     string
    Metadata map[string]string
    Friends  []*User
}
```

Passing `User` by value: `Name` in the caller is stable; `Metadata` and `Friends` contents are not.

### 26.4 Slices

A slice is a view of a backing array. Passing a slice by value still lets the callee write elements. `append` may share a backing array with other aliases.

v1: a Go call receiving a slice that supports an active fact may write `coll[*]` / `coll.length`. Coarse whole-slice is acceptable.

### 26.5 Maps

Maps are reference-like. A Go call receiving a map that supports an active fact may change its contents. Map aliases share storage.

### 26.6 Interfaces

`any`, `io.Writer`, `http.Handler`, and user interfaces hide the concrete value. Passing refinement-supporting state through an interface is conservative: assume reachable mutable state may change. Dynamic dispatch is a precision boundary.

### 26.7 Methods

Distinguish value vs pointer receivers. Value receiver: top-level copy; pointer receiver: may mutate the original. Value receivers can still mutate reference-bearing fields inside the copy. Receiver form helps; it does not fully determine effects.

### 26.8 Returned aliases

Go may return pointers, slices, maps, or interfaces that alias arguments.

```ft
address = goPkg.AddressOf(user)
ensure user is ValidUser
address.country = "XX"   // may drop ValidUser
```

v1: if the compiler cannot prove otherwise, a returned mutable reference **may-alias** mutable inputs.

### 26.9 Channels

Sending reference-bearing state is an escape boundary (§25). A Go API that stores a supplied pointer for later use is the same.

### 26.10 `reflect`, `unsafe`, cgo

These are unknown-effect boundaries. If refinement-supporting mutable state is reachable through them, drop affected facts. Do not attempt deep `unsafe` analysis. Degrade: forget the fact, do not reject the call.

### 26.11 Same-package handwritten Go

Sibling `.go` in a mixed package is untrusted Go until a later pass analyzes it.

### 26.12 Deriving Go write summaries

Go modules usually have source. A **later** precision pass may walk Go AST and infer `writes arg0.Name` like Forst bodies. That is not required to ship soundness. Missing summary ⇒ invalidate, never assume pure.

### 26.13 Optional interop metadata

A later RFC may add trusted summaries (compiler metadata, Forst bindings, declaration files, stdlib definitions) **without** patching upstream Go. This RFC only says such summaries **may** participate. No `mutates` syntax is required for soundness.

### 26.14 SEMANTICS_NARROWING Level B

Lattice `Meet` / `Join` stay free of memory effects. Invalidation is the fact-layer response to untrusted calls.

---

## 27. Forst-to-Go lowering

Refinement tracking is **compile-time**. There is no runtime borrow machinery and no hidden version field on objects.

Emitted Go for a program that only retains facts looks like the program without stability analysis. `ensure` still emits its usual runtime checks. Re-`ensure` after a write emits a second check. Nothing else.

Do **not** implement:

```text
user.version++
assert refinement.version == user.version
```

That adds runtime cost, hidden state, worse Go interop, and object-identity mess. Static flow analysis is the solution.

---

## 28. Diagnostics

When code **uses** a dropped fact, say the **fact is gone**, not that the **write was illegal**.

The write itself must **not** be an error:

```ft
ensure user is Adult
user.age = 10    // legal; Adult forgotten
```

An IDE may show that a fact was lost. That is not a language error.

A useful error on the later use includes:

1. which refinement was established (`Adult(user)`)
2. where it was established (`ensure user is Adult`)
3. which operation invalidated it
4. why (dependency overlap, or conservative interop boundary)
5. how to recover (`ensure user is Adult`)

Distinguish **known write** vs **conservative Go/unknown boundary**:

```text
The call to Go function:
  legacy.Normalize(&user)
may mutate state reachable through its first argument.
Forst cannot prove that user.age remains unchanged.
```

Do not suggest making the field immutable. Do not mention borrows.

---

## 29. Compiler representation

Conceptual core (names are internal):

```text
FlowEnvironment {
  facts: [
    { proposition: Adult(user),      dependencies: { user.age } }
    { proposition: ValidPeriod(period), dependencies: { period.start, period.end } }
  ]
}

for fact in activeFacts:
  if overlaps(fact.dependencies, operation.writeEffects):
    remove fact
```

Do not model `user: Adult` as an unconditional type replacement. Do not fold this into `Meet` / `Join`.

Provenance for diagnostics: store the `ensure` / guard span that established the fact, and the operation that dropped it.

---

## 30. Effect representation

Summaries are parameter-relative. Room for:

```text
writes
returns_alias_of
escapes
spawns_with
```

```text
rename:        writes arg0.name
getAddress:    returns_alias_of arg0.address
enqueue:       escapes arg0
go modify:     spawns_with arg0
```

v1 must implement `writes` for Forst bodies. The others may start coarse (`escapes` whole root, `returns_alias_of` all mutable args). The representation must not paint the compiler into “writes only.”

Recursive / mutually recursive Forst functions: fixpoint. Widen to `argK.*` if the cycle will not stabilize precisely.

---

## 31. Complexity budget

This feature needs flow-sensitive facts, dependency extraction, write effects, and conservative may-alias.

It does **not** initially need ownership, move semantics, lifetime/borrow syntax, borrow-conflict diagnostics, affine/linear types, or region inference.

That distinction is the point of Forst here: safety for this problem without a new user-facing conceptual system.

A later “stability region” (nothing may mutate these deps in this block) would resemble borrowing. Introduce it **only** after evidence that re-`ensure` is not enough. It is not part of this RFC.

---

## 32. Answers the handoff asked this RFC to settle

**Dependency extraction.** Walk assertion IR and analyzable guard bodies. Comparisons and `is` on a path depend on that path. Named guards: union of body facts (cached). Shape literals: mentioned field paths. `Any`/`All`: union of children. Relational: every mentioned place. Unanalyzable atom: whole subject root. Nested guards: fixpoint / visited set.

**Storage identity.** Paths: root ident plus `Field` / `IndexWildcard` / `Deref`. Overlap: prefix, equal, or collection-wildcard (§9). Internal encoding is free as long as overlap matches this.

**Aliasing, first implementation.** Precise: proven pure value copies do not alias; `&`, pointer assign, slice/map assign do. Widen to whole-root when the type has unknown interior mutability or the operation is untrusted.

**Forst effects.** Parameter-relative write paths; substitute at the call site; fixpoint on recursion.

**Go effects.** Capable of exposing mutable storage: pointers, slices, maps, chans, interfaces, funcs, and structs containing those. Scalars and strings (logical value) are not. Go AST summaries are a later precision pass. Unknown-effect boundary: no Forst body, interface dispatch, `reflect`/`unsafe`/cgo, unanalyzed code.

**Returned references.** Known alias when the Forst (later: Go) body returns a path into an argument. Fallback: may-alias all mutable args.

**Escapes.** At minimum: `go` launch, channel send, callback/closure passed to unknown code, unknown Go calls that receive reachable mutable state, storing a pointer where Go may keep it.

**Diagnostics.** Facts carry establishment span and invalidating operation. Errors on **use**, not on write.

---

## 33. Implementation stages

Mutation/aliasing/Go only. Sits on [12 §25](./12-accepted-decision.md) phases 1–3. Spec tests: [`.plans/analyzable-refinements/tests/`](../../../../.plans/analyzable-refinements/tests/README.md). **TDD:** each stage ports its directory to real tests first (red), then implements until green.

0. **4a — Fact + dependency IR.** Store facts with paths. No invalidation yet.
1. **4b — Direct local mutation.** Assignments, reassignment, field writes, `Present`/`Nil`, control-flow merge, loops. No alias analysis.
2. **4c — Forst function effects.** Infer `writes`; transitive; fixpoint.
3. **4d — Basic alias tracking.** `other = user`, `&user`, nested field aliases. Proven copies do not alias.
4. **4e — Collections.** Coarse `[*]` / length / append / maps / element aliases.
5. **4f — Go interop.** Conservative invalidate from reachable mutable args; sibling `.go`; optional later Go AST summaries.
6. **4g — Escapes and concurrency.** `go`, channels, escaping closures, stored references.
7. **Later — Precision.** Element-level collections, interface dispatch, return provenance, stdlib/Go summaries, optional metadata. Not a ship gate. No new syntax unless inference is proven inadequate.

Do not implement 13 against sticky `NarrowingTypeGuards []string` without dependency sets.

---

## 34. Required tests

The catalog is [`.plans/analyzable-refinements/tests/`](../../../../.plans/analyzable-refinements/tests/README.md). That directory is the gate, not this section’s examples.

Minimum situations (each is a file there):

- unrelated write retains (`lastViewedAt`, `user.name`)
- overlapping write drops; the write itself typechecks
- use that does not need the fact remains ok (`print(user.name)`)
- re-`ensure` restores
- reassignment drops
- relational start/end vs description
- compound `or` dies as a whole
- shape fact-level drop
- local scalar only reassignment invalidates
- merge: both branches name-only retains; one branch age write drops
- loop name-only retains; loop age write drops
- closure create retains; call drops; escape conservative
- `rename` retains; `setAge` drops; transitive `updateProfile`
- value copy does not alias; pointer/`&` does
- collections index/append
- Go pointer / slice / map / interface / sibling `.go`
- `go f(user)` and channel send drop
- `reflect`/`unsafe` drop
- diagnostic names fact, establishment, invalidator; write is not the error

---

## 35. Non-goals

- Rust ownership, lifetime/borrow syntax, affine/linear types
- making narrowed values automatically immutable
- preventing mutation because a refinement exists
- perfect alias analysis or perfect Go effect inference
- proving arbitrary concurrent mutation race-free
- deep static reasoning through `unsafe`
- runtime object versioning
- user-facing effect syntax unless later evidence requires it
- SMT / predicate implication ([12](./12-accepted-decision.md))
- keeping a `|` disjunct after the other side’s storage changes
- folding invalidation into `Meet` / `Join`

---

## 36. Normative rules

1. A successful guard or `ensure` may establish refinement facts in the current flow environment.
2. Each fact has a set of storage locations on whose current values it depends.
3. A refinement remains established only while those dependencies are known unchanged.
4. An operation that may modify a dependency invalidates the affected refinement.
5. Mutation is not prohibited merely because it invalidates a refinement.
6. A discarded refinement may be established again through `ensure`, a guard, or other narrowing.
7. Mutation through an alias has the same effect as direct mutation.
8. Function calls invalidate according to possible write effects.
9. Forst infers mutation effects for Forst code where possible.
10. When mutation or alias effects cannot be determined safely, conservatively invalidate.
11. Go interop follows the same rules; unknown Go mutation is conservative.
12. Preserve unaffected facts; do not discard all narrowing on an object after every mutation.
13. Stability is compile-time; no runtime borrow or mutation-tracking machinery.
14. Mutable state exposed to concurrently executing code cannot support retained refinements unless Forst can prove the dependencies remain stable.

---

## 37. Decision

> **Refinements are proofs over mutable program state, not permanent type conversions.**

A proof has dependencies. While they are stable, the compiler may use it. When something may modify them, the compiler forgets it. The programmer re-establishes with `ensure` when they need it again.

No borrow checker. No implicit immutability. No hidden runtime versioning.

Implement via [`.plans/analyzable-refinements`](../../../../.plans/analyzable-refinements/README.md) phase 4, tests first.

