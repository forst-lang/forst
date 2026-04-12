# Result value construction: alternatives to `Ok()` / `Err()` (RFC)

**Audience:** Language designers, compiler contributors, and API authors reviewing how **`Result(T, E)`** values should be **produced** in source. **Narrowing** (`is`, `ensure`) is only in scope here where it touches **construction** vs **guards**.

**Status:** **Exploratory** for **construction** only. This document **withdraws** the normative choice (in [11](./11-forst-result-tuple-single-return.md)) that **`Ok(...)`** / **`Err(...)`** are the **dedicated constructors** for `Result` values. **Separately:** **`Ok`** and **`Err`** remain **built-in type guards** on **`Result`** for **`if` / `ensure` narrowing**—that is the **intended direction for now** and is **not** deprecated by this RFC.

**Leading candidate (stated preference):** **No** **constructor** **keywords** for **`Result`** values—**success** via **ordinary** **`return x`** (where **`x: S`**), **failure** via **explicit** **nominal** **error** **constructors** (**`ParseError(...)`**, **`ValidationError { … }`**, factories—see **§5.7** and **§7**). **Requires** type rules when **`S`** and **`F`** overlap.

**Relations:** Builds on [02-result-and-error-types.md](./02-result-and-error-types.md) (`Result` shape, **error-kinded** failure), [11](./11-forst-result-tuple-single-return.md) (single return, `Tuple`, Go lowering), [09](./09-ensure-is-narrowing-and-binary-types.md) (`ensure`, `is`, narrowing), [13 — Prior art](./13-result-constructors-prior-art.md) (other languages, **`ensure`** / factories vs constructors), the **[errors RFC hub](../errors/README.md)** and **[error system architecture](../errors/00-error-system-architecture.md)** (§7), and [PHILOSOPHY.md](../../../../PHILOSOPHY.md) (explicit control flow, traceable errors).

---

## 0. Scope split: guards vs constructors

| Topic | Direction |
| --- | --- |
| **`Ok` / `Err` as built-in guards on `Result`** | **Keep** — use in **`is Ok(...)`** / **`is Err(...)`** (and **`ensure`**) to narrow success vs failure payloads. Same role as in the **type guard** / assertion pipeline ([09](./09-ensure-is-narrowing-and-binary-types.md), [generics §10](../generics/00-user-generics-and-type-parameters.md#result-types-generics-and-narrowing-ok-and-err)). |
| **`Ok` / `Err` as the only way to *construct* `Result` values** | **Open** — **not** locked; **§5.7** records the **preferred** direction (**plain** **`return`** + **nominal** **errors**); §5–5.6 remain **alternatives** if that **preference** changes. |

Implementations may still **lower** construction through **`Ok`/`Err` internally** during transition; the **user-facing** **target** is **no** **`Ok`/`Err`** **wrappers** at **returns**—see **§5.7**.

---

## 1. Problem statement

Dedicated **`Ok(...)`** and **`Err(...)`** forms (whether spelled as keywords or magic builtins) add **visual noise** at almost every return in `Result`-returning code. They repeat information already carried by:

- the function’s **declared** return type `Result(S, F)`, and  
- the **static type** of the value being returned.

Many authors would prefer **lighter primitives** that preserve:

- **Unambiguous** success vs failure in the **type system** and in **lowered Go** `(T, error)`.  
- **Teachable** rules—readers should see **where** errors are produced without hunting for magic wrappers.  
- **Stable** interop: imported **`(T, error)`** and emitted Go must remain **idiomatic**.

This RFC records **design goals**, **candidate directions** (§5), and a **creative catalog** (§5.6) so the project can converge on a single story and update [11](./11-forst-result-tuple-single-return.md) when ready.

---

## 2. Non-goals

- **Removing** the **`Result(S, F)`** type constructor or **error-kinded** failure parameter ([02](./02-result-and-error-types.md)) — out of scope.  
- **Replacing** **`Ok`/`Err`** as **built-in guards** on **`Result`** for **`is` / `ensure`** — **not** proposed here; guards stay **for now** (§0).  
- **Guaranteeing** a final **construction** syntax in this document — **exploratory** only until merged into [11](./11-forst-result-tuple-single-return.md) or a normative addendum.

---

## 3. Design goals (must-haves)

| Goal | Rationale |
| --- | --- |
| **Low clutter** | Fewer tokens per return than mandatory `Ok`/`Err` wrappers where the type already fixes the channel. |
| **No silent failures** | Rules must not hide **which** expressions are failures; [PHILOSOPHY.md](../../../../PHILOSOPHY.md) applies. |
| **Decidable checking** | Inference or sugar must not require whole-program analysis that breaks IDE responsiveness. |
| **Go mapping** | Success/failure must lower cleanly to **`(T, error)`** (or agreed wrapper types) without ambiguous heuristics at the FFI boundary. |
| **Narrowing** | Callers refine **`Result`** with **`is`** / **`ensure`** using **built-in `Ok`/`Err` guards** on the **`Result`** subject (**stable intent**); only **how values are created** is under review here. |

---

## 4. Design tensions (tradeoffs)

| Tension | Notes |
| --- | --- |
| **Brevity vs explicitness** | Omitting wrappers favors scanning; explicit **`Ok`/`Err`** favored auditability and RFC 11’s old “unambiguous codegen” story. |
| **Inference vs annotations** | **`ensure`-only** or **body-driven** inference reduces noise but needs **triggers** when there is **no** `ensure`, and rules for **public** APIs. |
| **Uniformity vs Go familiarity** | “**Return `err`** on failure paths” matches Go muscle memory but must not **confuse** success values with failures when types overlap (e.g. **`Error`**-shaped values). |

---

## 5. Candidate directions (not mutually exclusive)

These are **options to combine or reject**; none is **selected** by this RFC.

### 5.1 Typed `return` sugar

Allow **`return expr`** when **`expr`**’s type is provably **only** the **success** type **`S`** or **only** the **failure** type **`F`** of `Result(S, F)`, and **lower** to the appropriate branch without requiring **`Ok`/`Err`**. **Ambiguous** cases require an **annotation**, **`as`**, or a **fallback** primitive (temporary or permanent).

**Pros:** Removes noise when types disambiguate. **Cons:** Edge cases where **`S`** and **`F`** overlap need explicit disambiguation rules.

### 5.2 Block- or statement-scoped outcomes

Introduce **small** constructs (e.g. success/failure **blocks**, or **`return .success(x)`** / **`return .failure(e)`** with **enum-like** labels) that are **shorter** than function calls but **visible** in the AST.

**Pros:** Explicit discriminant without **`Ok`/`Err`** names. **Cons:** New syntax to teach; must not resemble **exceptions**.

### 5.3 `ensure` / control-flow–driven inference

Infer **`Result`**-shaped returns when the function contains at least one **`ensure`** (or similar) and **failure paths** are typed as **`Error`** or **subtypes**—along the lines of “errors flow from **`ensure`** and **`return err`**.”

**Pros:** Aligns **`ensure`** with **`Result`** mentally. **Cons:** Functions **without** `ensure` need another rule; **if** branches that **return** different **error** types imply **widening**, **unions**, or **annotations**.

### 5.4 Keep runtime discriminant; guards vs return syntax can diverge

Lowered Go remains **`nil` vs non-nil `error`**. **`is`** / **`ensure`** can keep using **built-in `Ok`/`Err` guards** on **`Result`** (§0) regardless of which **construction** sugar lands—**guard names** need not match **constructor** spelling.

**Pros:** Minimal change to **narrowing** and IDE affordances. **Cons:** Authors must remember **guards** (`is Ok`) vs **returns** (new sugar) if they differ.

### 5.5 Method or namespace constructors

**`Result.success(x)`** / **`Result.failure(e)`** (illustrative only—not locked) as **ordinary** or **static** forms instead of keywords.

**Pros:** No reserved **`Ok`/`Err`** tokens. **Cons:** Still **call-shaped**; may feel as heavy as today’s constructors.

### 5.6 Creative catalog: alternatives to **`Ok`/`Err` *constructors*** (not guards)

This section is **brainstorm only**. **`is Ok` / `is Err`** on **`Result`** remain the **guard** vocabulary for **narrowing** (§0); the ideas below replace only **how you *write* success/failure values** at **`return`** (or equivalent), and aim for **less visual clutter** than **`return Ok(x)`** / **`return Err(e)`** without looking like **type-guard** predicates.

| Idea | Sketch (illustrative) | Why it might feel lighter | Risks |
| --- | --- | --- | --- |
| **Dual keywords: `give` / `fail`** | `give x` · `fail e` | Two **verbs**: **give** the success value vs **fail** with **`e`**. Not **`is …`**-shaped. | **`fail`** suggests **panic** to some readers; must document as **recoverable** **`error`**. |
| **`pass` / `fail`** | `pass x` · `fail e` | **`pass`** reads as “**pass** the check / deliver success” (not Python’s `pass`). | **`pass`** overloads a familiar word from **Python**. |
| **`return` + channel keyword** | `return success x` · `return failure e` | **English** full words; **no** `()`-shaped guard mimic. | Verbose; **`success`/`failure`** might collide with **identifiers** unless reserved or contextual. |
| **Labeled `return`** | `return .value(x)` · `return .error(e)` | **Labels** read like **enum cases** without importing **`Ok`/`Err`**. | Dot-label noise; close to **Swift** `Result` cases. |
| **Unary channel markers** (sparingly) | `return ^x` · `return ~e` | Extremely **compact**. | **Cryptic**; hurts accessibility and grep-ability. |
| **Arrow flow** | `return -> x` · `return <- e` | **Arrows** suggest **direction**: forward value vs **hand back** error. | Unfamiliar; parsing **`->`** vs function types. |
| **Effect-flavored names** | `return yield x` · `return abort e` | **`yield`** (misleading) / **`abort`** sounds terminal. | **`yield`** wrong unless **generators** exist; **`abort`** too strong. |
| **Outcome namespace** | `outcome.value(x)` · `outcome.error(e)` | One **import**, **clear** roles; not **`is`**-shaped. | Still **call-shaped**; similar to §5.5. |
| **Traffic / status metaphor** | `green(x)` · `red(e)` | **Memorable**, **not** guard syntax. | Silly for some codebases; i18n / colorblind docs. |
| **Sport / game metaphor** | `score x` · `foul e` | Playful **domain** language. | Too informal for **stdlib** tone. |
| **Plain `return` + inference only** | `return x` · `return e` | **Zero** extra tokens when **`S`** and **`F`** are **disjoint** or provable. | **`S`** ∩ **`F`** **non-empty** needs **escape hatch** (annotation or §5.1). |
| **`ensure` as the “failure pump”** | Rarely **`return`** failure; **`ensure cond or e`** / **`ensure x is …`** covers failures; success is **`return x`** | **Validation-first**; fewer **explicit** failure **returns** in happy code. | Not all failures are **predicates** on a **single** value. |
| **Block-scoped only** | `with result { … value x … fault e }` | **Structured**; keywords **local** to block. | New **block** form; teaching cost. |
| **Borrow from Zig (spirit, not syntax)** | Named **error** values + **`return x`** success; **`return err`** where **`err`** is typed **`F`** | **No** **`Err()`** wrapper if **`e: F`** is enough. | Needs **clear** rules when **`x`** could be **`S`** or **`F`**. |
| **Borrow from Effect (spirit)** | `succeed(x)` · `fail(e)` as **stdlib** **aliases** (not guards) | Familiar to **TS** **Effect** readers. | Still **functions**; may feel as heavy as **`Ok`/`Err`**. |
| **Asymmetric: only mark the rare arm** | **`return x`** default success; **only failures** need **`fail e`** (or another **failure-only** keyword, not exceptions) | **Happy path** is **one** token **`return`**. | **Asymmetry** can confuse (“why is **failure** special-cased?”). |
| **Sigil on expression** | `return +x` / `return -e` (unary) | **Tiny**. | **`-e`** might parse as **negation**; **`+`** easy to miss. |
| **String-tag for humans** | `return #ok x` · `return #err e` | **Hash** reads as **“channel tag”** in editors. | **`#`** already used elsewhere? noisy. |
| **“Package” the value** | `wrap x` · `unwrap e` | — | **`unwrap`** implies **panic** in **Rust**; **avoid**. |
| **Honest boring names** | `return good(x)` · `return bad(e)` | Short **adjectives**. | **Childish** in **production** APIs. |
| **Domain-neutral: `value` / `fault`** | `value x` · `fault e` as **keywords** in **`Result`** fn | Neutral **nouns**; not **`Ok`/`Err`**. | **`value`** is overloaded (**expression** **value**). |

**Patterns that recur in the table**

- **Verb-led** exits (**`give`**, **`fail`**, **`pass`**) — read like **imperatives**, not **predicates**.  
- **Single `return` + inference** — minimum clutter; needs **solid** **disambiguation**.  
- **Labeled / enum-like** — **`return .foo`** — **clear** **AST** without **`()`** **guard** **parallels**.  
- **Metaphor** — fun in **docs**; **risky** as **only** **official** **spelling**.

**Recommendation for narrowing this list:** prototype **2–3** **candidates** in **small** **examples** (same **`Result`**, same **lowered Go**) and score **teachability**, **grep**, **collision** with **future** **keywords**, and **diagnostic** **messages**.

### 5.7 Stated preference: no constructor keywords — plain **`return`** + nominal errors

This subsection records a **concrete** **direction** favored by **language** **design**: **drop** **keyword** **wrappers** (**`Ok`**, **`Err`**, **`fail`**, **`give`**, …) for **building** **`Result`** **values** at **`return`** sites.

| Channel | Preferred surface | Rationale |
| --- | --- | --- |
| **Success** | **`return x`** with **`x: S`** for **`Result(S, F)`** | **Normal** **return** matches **happy-path** **reading**; **no** extra **token** when the **value** is **already** **typed** as **success**. |
| **Failure** | **`return` `SomeError(...)`** or other **explicit** **nominal** **constructors** from the **[error hierarchy](../errors/00-error-system-architecture.md)** (**`ValidationError { … }`**, **factories**, …) with **`SomeError <: F`** | **Failure** **mode** is **named** by the **error** **type**, not by a **generic** **`Err`** shell—aligned with **§7**. |

**Guards unchanged:** **`is Ok(...)`** / **`is Err(...)`** on **`Result`** **subjects** remain the **narrowing** vocabulary (**§0**); this **preference** only affects **function** **bodies**, not **`if`/`ensure`** **syntax**.

**Typechecker obligations**

- **Infer** or **check** that **`return expr`** in a **`Result(S, F)`** **function** selects the **success** path when **`expr: S`**, and the **failure** path when **`expr: F`** (and **`F`** is **error-kinded** per [02](./02-result-and-error-types.md)).  
- When **`S`** and **`F`** are **compatible** or **overlap**, require an **annotation**, **`as`**, or a **dedicated** **disambiguation** rule (same **tension** as §5.1).

**Lowering:** **`return x`** → **`return x, nil`** in Go; **`return DomainError(...)`** → **`return zero, err`** (or **wrapper** **implementing** **`error`**) per **existing** **transformer** **strategy**.

---

## 6. Open questions

1. Which **construction** rules (§5–5.6) combine cleanly with **unchanged** **`Ok`/`Err` guards** on **`Result`**?  
2. How do **imported** **`(T, error)`** values surface in **`is`**—unchanged **runtime** representation, new **pattern** forms?  
3. What is the **migration** path for **return sites** that today use **`Ok`/`Err`** as constructors in the compiler tree (guards may stay as-is)?  
4. How do **nominal** **error** **types** and **factories** from the **errors RFC** reduce the need for a **wrapper** at every **failure** **return** (§7)?

---

## 7. Error system RFC — relationship and what can be inferred

The **[errors RFC hub](../errors/README.md)** and **[error system architecture](../errors/00-error-system-architecture.md)** describe a **structured**, **hierarchical** **`Error`** model (base type, **categories** like **`ValidationError`** / **`DatabaseError`**, **fields**, **factories**, **tracing**, **TS/Effect** **tagged** interop). That work is **orthogonal** to **whether** **`Ok`/`Err`** spell **construction**, but it **strongly constrains** what “**good**” **failure** **returns** look like and what the **typechecker** can **infer**.

### 7.1 Alignment with **`Result(S, F)`** ([02](./02-result-and-error-types.md))

- The **failure** parameter **`F`** of **`Result(S, F)`** is **error-kinded**: it must be **`Error`** or a **subtype** / **refinement** in the **same** hierarchy—not an arbitrary **`E`** as in Rust’s default **`Result<T, E>`**.  
- **Subtyping:** **`Result(Int, ParseError)`** usable where **`Result(Int, Error)`** is expected when **`ParseError <: Error`**—same **width** story as in [02 §2](./02-result-and-error-types.md) and the **architecture** doc’s **tree** of **specific** errors under **`Error`**.

**Inference:** Any **construction** rule that allows **`return e`** on the **failure** path should require **`e`** to be **assignable** to **`F`** (and thus, ultimately, compatible with **`Error`** semantics).

### 7.2 What the architecture doc already commits to (product shape)

| Theme | Content relevant to **Result** **construction** |
| --- | --- |
| **Nominal errors** | **`error ValidationError extends Error { … }`**-style types with **fields** — failures are **not** only **`string`**; they carry **structure** for **TS** and **observability**. |
| **Factories** | **Error factory system** (see **§4** in the architecture doc) — **preferred** way to **build** instances with **context**. |
| **Categories** | **Validation** / **Network** / **Business** / **System** — informs **retry**, **logging**, **HTTP** mapping; **not** the same as **`is Err`**, but **overlaps** with **how** **`ensure`** failures are **classified**. |
| **Effect / tagged errors** | **TypeScript/Effect integration** — **discriminated** / **tagged** shapes for **client** code; parallels **Effect**’s **`catchTag`** story ([13 §5.6](./13-result-constructors-prior-art.md)). |

**Inference:** If **failure** values are usually **`ValidationError { … }`** or **`SomeDomainError(…)`** from a **factory**, the **type** of the **expression** already **says** “**failure**” **more** **clearly** than a **generic** **`Err(...)`** wrapper. That **reduces** the **informational** **need** for **`Err`** as a **second** **label** on every **return**.

### 7.3 Interaction with **`ensure`** and **validation**

- The architecture doc emphasizes **validation**-shaped errors (**`ValidationError`** with **field**, **constraint**, …).  
- **`ensure x is …`** ([09](./09-ensure-is-narrowing-and-binary-types.md)) is the **language** hook for **checks** that **fail** with **`error`**—often **mapping** to **validation** or **domain** errors in the **hierarchy**.

**Inference:** A **large** fraction of **failure** **paths** may be **expressible** as **`ensure`** + **narrowing** + **standard** **error** **payloads**, leaving **fewer** **raw** **`return`** **failure** **sites** that need **`Err(...)`** or a **replacement** **keyword** (**`fail`**, etc.).

### 7.4 What you can (and cannot) infer from the errors RFC alone

**Can infer**

- **Failure** **returns** should **compose** with **named** **error** **types** and **factories**—**syntax** should not **fight** **`ParseError(...)`** or **Go** **`fmt.Errorf`**-style **wrapping** at **boundaries**.  
- **Narrowing** on **`F`** at **call** **sites** (e.g. **`if r is Err(e)`** then **`e`** refined to **`ParseError`**) is **compatible** with a **rich** **hierarchy**—**guards** stay on **`Result`**, **payloads** are **hierarchy**-typed.  
- **Widening** **`F`** to **`Error`** at **module** **edges** is **expected**; **construction** rules should **not** **force** **narrow** **wrappers** at **every** **internal** **return**.

**Cannot infer** (needs **this** RFC + **language** **decisions**)

- **Exact** **keywords** for **success**/**failure** **returns**.  
- Whether **`Err(expr)`** remains the **internal** **AST** for **all** **failure** **returns** regardless of **surface** syntax.  
- **Full** **implementation** **status** of the **errors** **RFC** (see [ROADMAP.md](../../../../ROADMAP.md); parts are **planned** vs **delivered**).

### 7.5 Summary

The **error** **hierarchy** RFC implies **failures** are **first-class** **nominal** **values** with **clear** **types**—so **constructor** **clutter** is **redundant** mainly when **`Err(...)`** **duplicates** information already in **`SomeError(...)`**. **Construction** **experiments** (§5–5.6) should be **judged** against **that** **target** **shape**, not only against **`(T, error)`** **lowering**.

**Fit with §5.7:** The **stated** **preference** (**plain** **`return`** + **nominal** **error** **constructors**) is **exactly** the **shape** the **hierarchy** RFC **supports**: **failures** read as **domain** **types**, **success** as **ordinary** **returns**.

---

## 8. Relationship to implementation

The **current** compiler uses **`Ok`/`Err`** both as **expression constructors** and as **assertion/guard** forms for **`Result`**. This RFC targets **replacing mandatory constructor usage** at **return** sites, **not** removing **`Ok`/`Err`** from the **narrowing** / **type guard** pipeline unless a future, separate decision does so.

---

## 9. References

| Doc | Role |
| --- | --- |
| [11](./11-forst-result-tuple-single-return.md) | Single return, `Result`/`Tuple`, Go interop (**constructor section explicitly open**) |
| [02](./02-result-and-error-types.md) | Error-kinded **`Failure`**, hierarchy |
| [09](./09-ensure-is-narrowing-and-binary-types.md) | `ensure`, `is`, narrowing |
| [13](./13-result-constructors-prior-art.md) | **Rust/Go/Zig/TS** patterns; **`ensure`** + error factories vs **`Err()`** clutter |
| [07](./07-background-motivation-and-tradeoffs.md) | Broader pros/cons in the optionals space |
| [errors README](../errors/README.md) | Error system **hub** |
| [error system architecture](../errors/00-error-system-architecture.md) | **Hierarchy**, factories, **TS/Effect**, observability |
| [errors / ensure-only failure propagation](../errors/01-ensure-only-failure-returns.md) | **`ensure … is Ok() or …`** for **`Result`** propagation; **`Ok`/`Err`** as **guards**; **compiler** rule vs **`if`**; **coverage** vs this RFC’s **open** items |

---

## Document history

| Change | Notes |
| --- | --- |
| Initial | Exploratory RFC; **withdraws** locked **`Ok`/`Err`** **constructor** decision from [11](./11-forst-result-tuple-single-return.md); lists goals, tensions, and candidate directions. |
| Clarification | **§0** — **`Ok`/`Err`** stay as **built-in guards** on **`Result`** for **`is`/`ensure`**; only **construction** is open. |
| §5.6 | **Creative catalog** — **`give`/`fail`**, **`pass`/`fail`**, labeled **`return`**, arrows, metaphors, inference-only, **asymmetric** failure-only syntax; **brainstorm**, not a decision. |
| §7 | **Deep dive** — **[errors RFC](../errors/README.md)** / **[architecture](../errors/00-error-system-architecture.md)** vs **`Result`** **construction**; **what** **can** / **cannot** be **inferred**. |
| §5.7 | **Stated preference** — **no** **constructor** **keywords**; **`return x`** **success**, **`return NominalError(...)`** **failure**; **guards** unchanged. |
