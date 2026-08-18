# Generate plugins example

> **Work in progress.** Official semantic plugins ship with the compiler but have no independent semver yet. Pin the compiler, expect output and `opt` shapes to evolve, and read [Plugins](https://forst-lang.org/docs/workflow/plugins) for known limitations.

Single Forst boundary that exercises every official `forst generate` semantic plugin from real `.ft` sources.

## Layout

| Path | Purpose |
| --- | --- |
| `catalog/api.ft` | RPC catalog (`.Router()` + constraints) for jsonschema + orpc |
| `app/api/ping.ft` | File-route GET handler |
| `app/api/orders/$id.ft` | Parametric route for file-routes + react-router |
| `ftconfig.json` | Registers all official plugin binaries |

## Run

From the repo root (build compiler + plugins once):

```bash
task build
export PATH="$PWD/bin:$PATH"
forst generate ./examples/in/plugins
```

When using `@forst/cli` (`npx forst generate .`), official plugin binaries are downloaded into the same cache directory as the compiler and resolved automatically for bare `cmd` names in `ftconfig.json`.

Outputs land under `examples/in/plugins/generated/` (gitignored). The TypeScript client is written to `.forst/client` inside the example boundary.

## Plugins exercised

| Plugin binary | Output dir |
| --- | --- |
| `forst-gen-jsonschema` | `generated/jsonschema/` |
| `forst-gen-orpc` | `generated/orpc/` |
| `forst-gen-file-routes` | `generated/api/` |
| `forst-gen-react-router` | `generated/rr/` |
| `forst-gen-echo` | `generated/echo/` |

Inspect the semantic snapshot without running plugins:

```bash
forst generate --dump-semantic ./examples/in/plugins
```
