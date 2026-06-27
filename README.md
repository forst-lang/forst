# Forst &emsp; [![CI]][actions] [![Release]][release] [![Go Report Card]][goreport] [![Coverage Status]][coveralls] [![License]][license]

[CI]: https://img.shields.io/github/actions/workflow/status/forst-lang/forst/lint-test-coverage.yml
[actions]: https://github.com/forst-lang/forst/actions
[release]: https://img.shields.io/github/v/release/forst-lang/forst?filter=v*
[Go Report Card]: https://goreportcard.com/badge/github.com/forst-lang/forst
[goreport]: https://goreportcard.com/report/github.com/forst-lang/forst
[Coverage Status]: https://coveralls.io/repos/github/forst-lang/forst/badge.svg?branch=main
[coveralls]: https://coveralls.io/github/forst-lang/forst?branch=main
[License]: https://img.shields.io/github/license/forst-lang/forst

> Move heavy backend logic out of TypeScript without rewriting your app.  
> Keep your TypeScript. Replace the parts that break.

### Why use Forst

On Node, memory issues often show up in prod first: big JSON payloads, heavy 3rd party dependencies and unexpected load patterns. Forst is aimed at that memory story and at failures that are hard to trace when the real system breaks.

Forst compiles selected backend functions to native code (Golang) in a sidecar with its own memory budget, so that work does not inflate the Node heap. Move one function at a time; keep HTTP and routing in TypeScript. Call from TypeScript using generated types and middlewares so callers stay accurate; the sidecar runs locally.

## Who it’s for

Teams on Node, Bun or Deno who hit heap or GC issues in production, or who fight vague errors when debugging.

Teams who want ergonomic, predictable and debuggable backends.

## What Forst is (and isn’t)

| Why not just use …? | Because … |
| --- | --- |
| TypeScript only | High-throughput workloads and complex object graphs easily fill up the V8 heap in the main process, causing out-of-memory errors and high load.<br><br>Forst runs compiled work out of process with its own heap. |
| Go, Rust | A full swap or microservice means a new stack and often two backends until cutover.<br><br>Forst stays beside Node and you offload incrementally. |
| `worker_threads` | Workers add confusing plumbing and stay in the same process.<br><br>Forst enables native hot paths and a heap split without a worker protocol. |
| Zod, tRPC, Effect, Prisma | Share the goal of schema-first ergonomics.<br><br>Forst puts constraints on types and generates TypeScript for callers. |
| Neverthrow, fp-ts `Either`, hand-rolled `Result` | Share the goal of discriminatable errors.<br><br>Forst has first-class, nominal errors in the language, not only library wrappers. |
| “Another language to learn” | Niche languages often lack libraries and have odd syntax.<br><br>Forst follows Go: you can use all Go packages and the core syntax stays familiar. |

## Sidecar first: how you actually use it

You run another process and pay a local HTTP round trip for each `invoke`. In exchange you get heap isolation and native execution on the paths you move.

The integration model is CLI first and friendly for local development:

1. `@forst/cli`: `npx forst` installs the native compiler (no Go toolchain required). See [`packages/cli/README.md`](./packages/cli/README.md).
2. `forst dev`: development server with hot reload when `.ft` sources change (spawned by the sidecar in the common setup).
3. `@forst/sidecar`: starts or attaches to `forst dev`, exposes HTTP invoke from your existing Node/Express app, optional Express middleware (`req.forst`), `createRouteToForstMiddleware` to forward a full incoming request (method, path, headers, body) to a Forst function (uses `POST /invoke/raw` by default), and can run `generateTypes()` with the same roots as the dev server. See [`packages/sidecar/README.md`](./packages/sidecar/README.md).
4. `forst generate`: emits `generated/` TypeScript declarations and client stubs so callers don’t duplicate shapes. Optional `watchGenerate` keeps output in sync when you use spawn mode.

Spawn vs connect: one process spawns `forst dev` (typical on a laptop); other packages or CI jobs connect to an existing URL (`FORST_DEV_URL`) so you don’t run duplicate servers.

## Ten minute migration: before and after

Before: TypeScript + Express + Zod — `express.json()` turns the body into V8 objects; `JSON.parse` and large graphs cost heap in the same Node process as the rest of your app. Zod and your handler run there too, so CPU-heavy or allocation-heavy work shares GC and RAM with routing, sessions, and everything else.

```typescript
// app/routes/order.ts
import express from "express";
import { z } from "zod";

const placeOrderInput = z.object({
  stockKeepingUnit: z.string().min(1).max(64),
  quantity: z.number().int().min(1).max(99),
});

export const orderRouter = express.Router();

orderRouter.post("/orders", express.json({ limit: "1mb" }), async (req, res) => {
  const input = placeOrderInput.parse(req.body);
  const out = await placeOrderInProcess(input); // still: same V8 heap and interpreter for this path
  res.json(out);
});
```

After: TypeScript + Forst sidecar — Node keeps the route and I/O; validation and work move to compiled code in a separate process (`invoke`), so that path does not inflate Node’s heap the same way:

```go
// forst/orders/place.ft: constraints live on the type; invalid input never reaches your logic
package orders

type PlaceOrderInput = {
	stockKeepingUnit: String.Min(1).Max(64),
	quantity:         Int.Min(1).Max(99),
}

func PlaceOrder(in PlaceOrderInput) {
	// Compiled execution; nominal errors and Result flow (see examples/in/)
	return { orderId: "ord_123" }
}
```

```typescript
// app/server.ts — create the sidecar once, then pass `sidecar` into your routes
import { autoStart } from "@forst/sidecar";

const sidecar = await autoStart({ forstDir: "./forst", port: 8080 });
await sidecar.generateTypes(); // same roots as forst dev; commit generated/ in CI if you want
```

```typescript
// app/routes/order.ts — same route shape; the hot path runs in Forst instead of placeOrderInProcess + Zod
orderRouter.post("/orders", express.json({ limit: "1mb" }), async (req, res) => {
  const result = await sidecar.invoke("orders", "PlaceOrder", [req.body]);
  res.json(result);
});
```

| | Before (Express + Zod in Node) | After (Express + Forst sidecar) |
| --- | --- | --- |
| Heap | Body, Zod, and handler all allocate on one V8 heap. | Work runs in a sidecar with its own memory budget. |
| Validation | Zod in JS after the object graph exists. | Constraints on Forst types at compile and decode time in Forst. |
| Hot path | Interpreter + GC on every call. | Compiled `PlaceOrder`; Node does HTTP and `invoke` only. |

Adjust package names and `invoke` arguments to match your tree; full HTTP contract and config are documented in the [sidecar README](./packages/sidecar/README.md) and [`examples/in/rfc/typescript-client/`](examples/in/rfc/typescript-client/).

## Adoption path that wins

1. Install `@forst/cli` and `@forst/sidecar` (and `express@^5` if you use the middleware).
2. Move one hot or sensitive function into `.ft` and call it via `invoke`.
3. Watch for lower memory on that path, clearer structured errors, and no duplicate schema once `forst generate` is in the loop.
4. Move more functions when the upside is obvious.

Requiring a full backend rewrite, a new paradigm in every file, or dropping TypeScript would sink this. The path that works is incremental.

## Language snapshot (optional depth)

Forst is a compiled language with Go compatible syntax extended for structural types, field constraints, nominal errors, and `Result` / `ensure` style control flow. The long Before (Go) / After (Forst) walkthroughs (catalog order, map lookup, narrowing) live under [`examples/in/`](examples/in/). Open those when you’re writing `.ft`; you don’t need them to decide whether the sidecar is worth a try.

### Hello World

For ordinary programs, Forst is a superset of Go: the same `package main` can be built with `go build` or `forst run`. Use `.ft` for the Forst CLI (`forst run`, `forst generate`, …).

```golang
package main

import "fmt"

func main() {
	fmt.Println("Hello World!")
}
```

### Types across the wire (generated TypeScript)

```typescript
import type { PlaceOrderInput } from "./generated/types";

async function submitOrder(input: PlaceOrderInput) {
  // Shapes come from Forst; runtime validation lives in the compiled function instead of a second Zod schema here.
}
```

See [TypeScript client output](#typescript-client-output) for `forst generate` and the `generated/` layout.

## Features

See [ROADMAP.md](./ROADMAP.md) for planned work and feature parity.

- Import Go code to leverage the entire Golang ecosystem.
- Gradual adoption: move one function at a time; keep your existing backend.
- Generated TypeScript: `forst generate` emits types and stubs from `.ft` so callers (including frontend and backend) share one definition.
- Native compilation: `.ft` code runs compiled in the sidecar, not on the Node interpreter.
- Static typing and inference makes your code safer and easier to work with.
- Constrained types: compile-time checks on invalid call sites; generated checks on decoded input at runtime.
- Separate memory budget: heavy work can run in the sidecar instead of the Node/V8 heap.
- Nominal errors and control flow (`ensure`, `Result`): named error variants you can narrow on; optional TypeScript mirroring.

## Design Philosophy

See [PHILOSOPHY.md](./PHILOSOPHY.md) for what guides and motivates the project.

## Development

Install Task using the [official instructions](https://taskfile.dev/installation/).

Run tests:

```bash
task test                  # Go tests: forst/internal/... and forst/cmd/forst/...
task ci:test               # Full CI suite (Go tests + VS Code extension build + @forst/cli & @forst/sidecar + examples)
task test:unit             # Run compiler unit tests (internal only)
task test:unit:parser      # Run parser tests
task test:unit:typechecker # Run typechecker tests
```

Run examples:

```bash
task test:integration                    # Run compilation examples / integration tests
task example -- ../examples/in/basic.ft  # Run specific example
task example:function                    # Run function example
```

## VS Code

The workspace includes an optional extension in [`packages/vscode-forst`](./packages/vscode-forst): it registers `.ft` and talks to the compiler’s HTTP LSP (`forst lsp`) for diagnostics. Its release cadence is separate from compiler `v*` tags (see `vscode-forst-v*` in [`.github/workflows/publish-vscode-extension.yml`](./.github/workflows/publish-vscode-extension.yml)). After `bun install` at the repo root, run `task build:vscode` to compile it (or rely on the F5 preLaunchTask in [`.vscode/launch.json`](./.vscode/launch.json)). CI runs the same compile as the first step of `task ci:test`. See [`packages/vscode-forst/README.md`](./packages/vscode-forst/README.md) for F5 and troubleshooting.

## npm

- [`@forst/cli`](./packages/cli/README.md): `npm i -D @forst/cli`, then `npx forst` / `node_modules/.bin/forst` (downloads the matching native binary from GitHub Releases).
- [`@forst/sidecar`](./packages/sidecar/README.md): dev server lifecycle, HTTP client, Express helpers (`req.forst`, route-to-Forst middleware, `autoStart`), `generateTypes()`, hot reload integration.
- [`@forst/client`](./packages/client/README.md): optional higher level client patterns on top of generated output; see package README.

## TypeScript client output

Run `forst generate` with a `.ft` file or a directory of `.ft` files; it writes a `generated/` tree (declarations plus helpers) and client stubs you wire to your app. The `forst dev` server can also expose types over HTTP while you iterate. See [integration profiles](examples/in/rfc/typescript-client/01-integration-profiles.md).

## Inspirations

- TypeScript: structural typing and gradual adoption across a large existing surface.
- Zod: constraints and shape guards as composable runtime checks on nested data.
- tRPC: one source of truth for API shapes, with TypeScript types and a small client generated from Forst (`forst generate`, `examples/client-integration/`).
- Go and Rust: errors as values and explicit control flow (`ensure` … `or …`) instead of exceptions.
- Ruby and Crystal: readability and terseness in syntax.

Think of Forst as a way to lift the sharp edges off your TypeScript backend while you stay in TypeScript.
