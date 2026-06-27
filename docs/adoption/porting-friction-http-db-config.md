# Porting friction: representative slice (HTTP + DB + config)

One common **service-shaped** slice is: **HTTP handlers**, **database access**, and **configuration**. Below: what blocks **incremental** porting to Forst today vs **language parity** gaps.

## HTTP

- **Interop:** Handlers often use **`context.Context`**, **`http.ResponseWriter` / `*http.Request`**, **middleware types** — many of these are **not** in the narrow `goTypeToForstType` mapping; calls may be **unsupported** until types map or shims exist.
- **Parity:** **`switch` on status / routes**, **`select` on channels** with **`http.Server`** patterns — **not** end-to-end in Forst yet (roadmap: `switch` / `select` planned).

## Database (e.g. `database/sql`)

- **Interop:** `*sql.DB`, `*sql.Rows`, **`sql.Tx`**, **context**, **interfaces** — largely **outside** current Forst↔Go mapping; expect **unsupported** diagnostics unless wrapped in hand-written Go and called through a thin surface.
- **Parity:** **Generics** in newer stdlib APIs — **user generics** not shipped (roadmap).

## Config (env, flags, struct loaders)

- **Interop:** Parsing often returns **structs** and **maps** — same mapping limits; **`encoding/json`** with **exported fields** is aligned with `exportStructFields` for **Forst-owned** shapes, not necessarily for **arbitrary Go struct types** in imports.
- **Parity:** **`const` / `iota`** — experimental; large config tables may feel awkward.

## Summary

| Layer        | Main blockers today                                      |
| ------------ | -------------------------------------------------------- |
| Interop      | Struct/map/interface/channel types on Go boundary; `unsafe`; loader flakiness |
| Go parity    | `switch`, `select`/channels, user generics               |
| Full-stack TS| `forst generate` + dev server path is separate from Go-only ports |

Use [go-interop-audit.md](./go-interop-audit.md) for the concrete `go_interop.go` / `goload` behavior.
