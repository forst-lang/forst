# Go interoperability: lowering `T | Nil` and boundaries

**Topic:** How **optional** / **nilable** Forst types could **lower** to **idiomatic Go**, and **FFI** tradeoffs. Complements [01 §2](./01-single-return-unions-and-go-interop.md) (**import** mapping of **`(T, error)`** → **`Result`**), which focuses on **calling Go** from Forst—not **emitting** **`T?`**.

Go must stay **valid** and **idiomatic** where possible ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)).

---

## Strategies for lowering `T?` / `T | Nil`

1. **Lower to `*T`** (or **`*T`** + convention)  
   - **Pros:** Direct mapping; **`nil`** = absent.  
   - **Cons:** Value types (**`int`**, etc.) need **boxing** or **wrappers**—**cost** and **interop** friction.

2. **Lower to `(T, bool)`** or **`struct { value T; ok bool }`**  
   - **Pros:** No **heap** pointer for **`int`**; clear **presence**.  
   - **Cons:** Differs from Go APIs that use **pointers**; **imported** code still **`(T, error)`**.

3. **Keep public Forst→Go as `(T, error)`**; use **`T?`** only **internally** or in **generated** helpers  
   - **Pros:** **Stable** boundary.  
   - **Cons:** Two **mental models** at **FFI**.

4. **`Error?` as optional `error` field** in a **struct**  
   - Lowers to **`error`** + **`nil`** or **`*error`**—idiomatic for **optional diagnostic**.

**Interop principle:** **Calling** imported Go stays **`(T, error)`** until **`go/types`** supports richer **surface** types. **Emitting** Go from Forst should **document** the chosen **lowering** (pointer vs **`ok` bool**).

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md)
- [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md)
- [02-result-and-error-types.md](./02-result-and-error-types.md)
- [06-nil-safety-pointers-and-optionals.md](./06-nil-safety-pointers-and-optionals.md)

---

## Document status

**Exploratory.** Update when **emit** rules are **fixed**.
