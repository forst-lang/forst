# Examples

## Compiler inputs and golden outputs

- **`in/`** — Forst `.ft` sources used by Task targets (e.g. `task example:basic`). Paths are mirrored under **`out/`** with expected Go output for integration checks (`in` → `out`).

- **`in/*.ft`** at the root of `in/` (e.g. `basic.ft`, `ensure.ft`) are the small, primary examples referenced by the Taskfile and [testing rules](../.cursor/rules/testing.mdc).

## `in/rfc/`

Design notes (Markdown), sample `.ft` files, and sometimes TypeScript or config files are grouped **by topic** (sidecar, guards, typescript-client, effects, etc.). That folder is **not only** minimal “hello world” examples: it is **RFC-style documentation + runnable snippets** kept together. No separate layout is required as long as this convention is understood.

## `client-integration/`

Standalone demo of generated client usage; not part of the default `task example:*` paths unless documented in the Taskfile.
