# Phase 2 (exploration): running emitted Go without `forst dev`

This note is **not** a commitment to a single workflow. It lists common options after **Layer B** (`forst dev` + sidecar) is in good shape.

## Why this is separate from the sidecar

The **sidecar** runs the **`forst dev`** HTTP server and calls functions through **discovery + executor**. Production services often:

- Deploy a **compiled Go binary** or **container** with no dev HTTP server.
- Expose **REST/gRPC** written in hand-authored Go that **imports** transpiled Forst packages.

## Options (high level)

1. **`go run` / `go build`** on emitted output in CI or locally — standard Go toolchain; Forst does not need to own the command.
2. **Long-running Go service** — load balancer → Go; TS clients use **Layer A** types only, or OpenAPI/gRPC clients (future tooling).
3. **Subprocess** from Node — invoke a **compiled** binary with a defined stdin/stdout protocol (not the same as `forst dev` HTTP); would require a **new** contract if pursued.
4. **Serverless / edge** — often a **separate** Go deployment; shared **types** via `forst generate` remain relevant.

## Relationship to the roadmap

See [ROADMAP.md](../../../../ROADMAP.md) TypeScript interoperability and Go interoperability tables. **Innovation** items (OpenAPI emit, etc.) are tracked separately and are non-binding.
