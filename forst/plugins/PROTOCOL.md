# Semantic plugin protocol (v1)

`forst generate` typechecks your module once, projects a JSON **semantic snapshot**, and pipes it to each configured plugin on **stdin**. The plugin writes **stdout** JSON listing files to create under `generate.plugins[].out`.

This document is the public contract for plugin authors. It matches `protocolVersion: 1` in the compiler.

## Request (stdin)

```jsonc
{
  "protocolVersion": 1,
  "compilerVersion": "0.x.y",
  "plugin": { "name": "my-plugin", "opt": { } },
  "module": { "goModule": "github.com/acme/app", "root": "/abs/path" },
  "packages": [ /* SemanticPackage */ ],
  "types": { /* id → Type */ },
  "functions": { /* id → Function */ }
}
```

- **packages** — module layout: `name`, `dir`, `files[]`, `typeIds[]`, `functionIds[]`.
- **types** — resolved types keyed by id (`catalog.Username`, `structural:…`). Includes constraint chains, shape fields, method bindings.
- **functions** — runnable defs with `input`, `returns`, `errorSet`, and optional `span.file` for convention-based plugins.

There is **no `surfaces` table**. RPC and HTTP plugins discover contract members as **method fields on types** whose constraint chain includes a marker (builtin `Router` by default).

## Response (stdout)

```jsonc
{
  "protocolVersion": 1,
  "files": [{ "path": "schema.json", "content": "..." }],
  "diagnostics": [{
    "severity": "warning",
    "message": "…",
    "typeId": "catalog.Username",
    "span": { "file": "catalog/types.ft", "startLine": 3, "startCol": 1, "endLine": 3, "endCol": 40 }
  }]
}
```

Rules:

- `files[].path` — relative to the plugin's `out`, `/` separators only, no `..`, not absolute.
- Empty `files` is valid (analyzer-only plugins).
- Non-zero exit or invalid JSON → `forst generate` fails.
- Diagnostics are warnings/errors for the user; they do not replace a valid JSON response.

## Host behavior

| Host | Plugin |
| --- | --- |
| Parse, typecheck, build snapshot | Map snapshot → files / diagnostics |
| Spawn `cmd`, 30s timeout, validate paths | Return file contents only |
| Atomic write under `out`, skip identical bytes | Never read Forst source or AST |

Plugins never see tokens, AST, or function bodies. `TypeChecker` internals (`T_*` hashes, LSP maps) stay out of the protocol.

## Writing a plugin

1. `go build` a `main` package that reads stdin, unmarshals `semantic.GenerateRequest`, emits `semantic.GenerateResponse`.
2. Reuse [`internal/genplugin`](../internal/genplugin/) for the runner, router walking, and route-path helpers.
3. Register in `ftconfig.json`:

```jsonc
{
  "generate": {
    "plugins": [
      { "name": "my-plugin", "cmd": "forst-gen-my-plugin", "out": "generated/my-plugin", "opt": { } }
    ]
  }
}
```

4. Debug the snapshot without running plugins:

```bash
forst generate --dump-semantic ./your-app
```

## Snapshot goldens

The compiler maintains fixture snapshots under [`internal/semantic/testdata/`](../internal/semantic/testdata/) (`constraints`, `router`, `layout`). Plugin unit tests should unmarshal these goldens rather than hand-rolling JSON.

## Versioning

Breaking snapshot changes bump **`protocolVersion`** in the compiler. Individual plugin binaries embed their own generator version in `meta.json` / file headers (`0.1.0` today).
