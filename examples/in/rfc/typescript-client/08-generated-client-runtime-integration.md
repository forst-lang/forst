# RFC FR-CLIENT-1: Generated typed client and runtime integration

| **ID** | `FR-CLIENT-1` |
| **Status** | Draft — **decision record** |
| **Audience** | `@forst/sidecar` maintainers, compiler team, adopters building Node backends |
| **Depends on** | [03-forst-generate-contract.md](./03-forst-generate-contract.md), [02-forst-dev-http-contract.md](./02-forst-dev-http-contract.md), [07-typescript-client-layered-architecture.md](./07-typescript-client-layered-architecture.md) |
| **Related** | [../optionals/03-typescript-emission-optionals-and-result.md](../optionals/03-typescript-emission-optionals-and-result.md), [../sidecar/12-forst-http-outcome-pipeline.md](../sidecar/12-forst-http-outcome-pipeline.md) |

---

## 1. Abstract

This RFC **decides** how **generated TypeScript clients** (`*.client.ts`) integrate with **`@forst/sidecar` at runtime**, closing the gap between “types exist on disk” and “developers still pass string package/function names.” It also **decides** the **wire–type fidelity policy**: when generated TS **must** match JSON on the wire, and when it remains descriptive only.

---

## 2. Problem (current state)

Today:

- [`forst generate`](../../../../forst/cmd/forst/generate.go) emits typed stubs in [`client.go`](../../../../forst/internal/transformer/ts/client.go) that call `client.invokeFunction<T>(package, function, args)`.
- [`ForstSidecar`](../../../../packages/sidecar/src/forst-sidecar.ts) exposes untyped `invoke(packageName, functionName, args)`.
- Express handlers typically use **string literals** for package and function names.
- [`03 contract`](./03-forst-generate-contract.md) states generated TS is **descriptive**; runtime JSON may not match `Result<T,E>` unless a codec exists.

This produces a **broken DX loop**: codegen runs in CI, but day-to-day development ignores it.

---

## 3. Decisions

### D1 — Generated client factories are the canonical invoke surface

For each merge-package stem `foo.ft`, generation **must** emit:

```typescript
// generated/foo.client.ts
export const foo = (client: ForstSidecarClient) => ({
  bar: async (x: Input): Promise<Output> => { /* invokeFunction */ },
});
```

**Normative rules:**

1. App code **should** call `foo(sidecar.getClient()).bar(args)` — not raw `invoke("foo", "bar", [...])` — for symbols listed in IR.
2. Raw `invoke` remains **supported** for escape hatches, debugging, and dynamic dispatch not known at compile time.
3. `ForstSidecar` **must** expose `getClient(): ForstSidecarClient` (already exists) and **should** expose a typed aggregate (see D2).

### D2 — Sidecar facade loads generated modules (opt-in path)

`ForstSidecar` **should** support attaching generated clients without manual wiring:

| Approach | Decision |
|----------|----------|
| Auto-import all `generated/*.client.ts` at runtime | **Reject** — requires bundler/tsconfig paths; fragile in monorepos |
| **`createForstClient(sidecar, generated)`** helper in `@forst/client` or generated `client/index.ts` | **Adopt** — explicit import, tree-shakeable |
| **`ForstSidecar.useGenerated(import(...))`** fluent API | **Optional** — convenience wrapper over D2 |

Generated `client/index.ts` **must** export a factory:

```typescript
export function createForstClient(client: ForstSidecarClient) {
  return {
    engine: engine(client),
    server: server(client),
  };
}
```

This replaces the current class-based `ForstClient` default export as the **recommended** pattern ([`generate.go`](../../../../forst/cmd/forst/generate.go) evolution).

### D3 — Dev loop: `watchGenerate` + import from `generated/`

| Setting | Behavior |
|---------|----------|
| `watchGenerate: true` in sidecar config | After debounced `.ft` reload, run `forst generate`; TS server picks up changes |
| `generateTypes()` on demand | Same as CLI; throws on typecheck failure |
| `fetchTypes()` on low-level client | Returns live TS string; **must** include IR hash when [FR-IR-1](./06-forst-intermediate-representation.md) lands |

**Decision:** `ForstSidecar` **should** expose `fetchTypes()` as a delegate (today only on `ForstSidecarClient`). Live types are for **editor iteration**; committed **`generated/`** is for **CI and production builds**.

### D4 — Express integration types

| Surface | Decision |
|---------|----------|
| `createExpressMiddleware` → `req.forst` | **Keep**; add shipped **`@forst/sidecar/express`** augmentation types |
| `createRouteToForstMiddleware` | **Keep**; gateway types **must** import from generated `types.d.ts` when [RFC 05](./05-forst-http-gateway-signature-pipeline-rfc.md) gateway module ships |
| `req.forst.engine.playMove(...)` | **Target DX** via `createForstClient(req.forst.getClient())` attached once in middleware |

Sidecar **must not** duplicate domain types (e.g. `GameState`) long-term; gateway wire types (`ForstRoutedRequest`) **may** remain in sidecar until generated re-exports land ([sidecar/12](../sidecar/12-forst-http-outcome-pipeline.md)).

### D5 — Wire–type fidelity tiers

Not all generated types guarantee JSON shape. This RFC **defines three tiers**:

| Tier | Types | Wire guarantee | Verification |
|------|-------|----------------|--------------|
| **T0 — Descriptive** | Domain structs, generic `Result` | **None** — TS approximates Go/memory shapes | `tsc --noEmit` only |
| **T1 — Invoke envelope** | `InvokeSuccess<T>`, gateway `ForstRoutedResponse` | **Yes** — documented in [02 contract](./02-forst-dev-http-contract.md) | Golden JSON tests in sidecar |
| **T2 — Codec-backed** | `Result<T,E>` on listed invoke targets | **Yes** — when handler uses standard Result JSON codec | Golden tests compiler + sidecar |

**Decisions:**

1. **Default** domain function returns are **T0** until explicit codec or gateway path applies.
2. Gateway handlers (`GatewayHandler`) **must** be **T1** for `result` JSON ([RFC 05 §12](./05-forst-http-gateway-signature-pipeline-rfc.md)).
3. When Forst `Result` types are emitted to TS ([optionals/03](../optionals/03-typescript-emission-optionals-and-result.md)), listed remote functions **should** opt into **T2** via registration metadata or stdlib marshal helpers — not automatic for all functions.

### D6 — `FORST_TS_CONTRACT` versioning

Introduce header in `types.d.ts`:

```typescript
// forstTypesVersion: 1
// forstIrVersion: 1
```

Breaking changes to stub shape, import paths, or fidelity tier for a symbol **must** bump `forstTypesVersion` and appear in release notes ([03 contract](./03-forst-generate-contract.md)).

---

## 4. Output layout (normative target)

Relative to project root (ftconfig output dir):

| Path | Role |
|------|------|
| `generated/types.d.ts` | Merged types + version header |
| `generated/forst.ir.json` | IR ([FR-IR-1](./06-forst-intermediate-representation.md)) |
| `generated/<stem>.client.ts` | Per-file client factory |
| `client/index.ts` | `createForstClient`, re-exports |
| `client/package.json` | `@forst/client` stub depending on `@forst/sidecar` |

Apps **should** depend on `@forst/sidecar` directly for lifecycle; import generated factories from `./generated/` or `./client`.

---

## 5. Error handling (normative)

Generated stubs **must not** catch invoke errors. They **must** propagate:

- `DevServerInvokeRejected` — Forst returned `success: false`
- `DevServerHttpFailure` — non-2xx HTTP
- `DevServerRequestRetriesExhausted` — retry budget exhausted

Typed return types apply only to **successful** `result` payloads; failures remain exceptional ([02 contract](./02-forst-dev-http-contract.md)).

---

## 6. Streaming

When `compiler.generateStreamingClients: true` in `ftconfig.json`:

- Emit `<Fn>Stream(...): AsyncIterable<Row>` delegating to `client.invokeStream` ([`client.go`](../../../../forst/internal/transformer/ts/client.go)).
- Streaming **does not** use gateway `kind` JSON ([RFC 05 §11.1](./05-forst-http-gateway-signature-pipeline-rfc.md)); fidelity tier **T1** applies to NDJSON row types when row type is known.

---

## 7. Non-goals

- Bundling generated code into `@forst/sidecar` npm package
- Runtime schema validation (Zod/io-ts) in generated stubs — apps may add layer 3 packages
- Browser bundle of sidecar without Node — out of scope for this RFC

---

## 8. Conformance tests

Implementations **must** add or extend:

1. **`TestGenerate_typescriptTypechecks_*`** — generated tree passes `tsc` ([`generate_tsc_test.go`](../../../../forst/cmd/forst/generate_tsc_test.go))
2. **Golden invoke JSON** — fixture Forst function → POST `/invoke` → assert TS type narrows correctly (sidecar integration test)
3. **Gateway golden** — `GatewayResponse` encode → `result` JSON ↔ `isAnswer` / `isPass` ([`express-middleware.integration.test.ts`](../../../../packages/sidecar/src/express-middleware.integration.test.ts))
4. **Example manifest** — tictactoe `mustContain` includes `createForstClient` or factory export when implemented

---

## 9. Migration

| Today | Target |
|-------|--------|
| `new ForstClient()` class | `createForstClient(sidecar.getClient())` |
| `sidecar.invoke("pkg", "fn", args)` | `client.pkg.fn(args)` via generated factory |
| Hand-maintained Express casts | Import `@forst/sidecar/express` types |
| `fetchTypes()` only on client | Also on `ForstSidecar` facade |

Existing string invoke **remains valid** indefinitely for dynamic use cases.

---

## 10. Related documents

- [07-typescript-client-layered-architecture.md](./07-typescript-client-layered-architecture.md)
- [06-forst-intermediate-representation.md](./06-forst-intermediate-representation.md)
- [05-forst-http-gateway-signature-pipeline-rfc.md](./05-forst-http-gateway-signature-pipeline-rfc.md)
- [packages/sidecar/README.md](../../../../packages/sidecar/README.md)
