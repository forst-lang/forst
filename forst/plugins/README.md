# Forst semantic plugins

> **Work in progress.** Protocol v1, official emitters, and CLI integration are evolving. Pin the compiler version; plugin binaries do not have independent release semver yet.

Official [`forst generate`](https://forst-lang.org/docs/workflow/cli#forst-generate) plugins read the **semantic snapshot** (JSON on stdin) and write sealed artifacts under `generate.plugins[].out`. User code imports those outputs; never edit them by hand.

These plugins ship in the [Forst repository](https://github.com/forst-lang/forst) and are built with the compiler release. They are **not** embedded in the compiler — configure them in `ftconfig.json`.

## Install

From a clone of this repository (module path `forst`):

```bash
cd forst
task build:plugins
# installs to ../bin/
```

Or build individual tools:

```bash
go build -o "$(go env GOPATH)/bin/forst-gen-jsonschema" ./plugins/forst-gen-jsonschema
go build -o "$(go env GOPATH)/bin/forst-gen-orpc" ./plugins/forst-gen-orpc
go build -o "$(go env GOPATH)/bin/forst-gen-file-routes" ./plugins/forst-gen-file-routes
go build -o "$(go env GOPATH)/bin/forst-gen-react-router" ./plugins/forst-gen-react-router
```

Ensure the binary directory is on your `PATH`, or set `generate.plugins[].cmd` to an absolute path.

## Configure

```jsonc
{
  "generate": {
    "plugins": [
      {
        "name": "jsonschema",
        "cmd": "forst-gen-jsonschema",
        "out": "generated/jsonschema",
        "opt": { "draft": "2020-12" }
      },
      {
        "name": "orpc",
        "cmd": "forst-gen-orpc",
        "out": "generated/orpc",
        "opt": {
          "markers": ["Router"],
          "queries": ["catalog.Catalog.GetOrder"],
          "routes": {
            "catalog.Catalog.PlaceOrder": { "method": "POST", "path": "/orders" }
          }
        }
      },
      {
        "name": "file-routes",
        "cmd": "forst-gen-file-routes",
        "out": "generated/api",
        "opt": {
          "markers": ["Router"],
          "routesRoot": "app/api",
          "paramStyle": "$id"
        }
      },
      {
        "name": "rr-ssr",
        "cmd": "forst-gen-react-router",
        "out": "generated/rr",
        "opt": {
          "markers": ["Router"],
          "routesRoot": "app/api",
          "paramStyle": "$id",
          "invoke": "package"
        }
      }
    ]
  }
}
```

Run:

```bash
forst generate ./your-app
forst generate --dump-semantic ./your-app   # inspect snapshot only
```

## Plugins

| Binary | Purpose | Docs |
| --- | --- | --- |
| `forst-gen-jsonschema` | JSON Schema from type constraint chains | [README](./forst-gen-jsonschema/README.md) |
| `forst-gen-orpc` | oRPC/tRPC-shaped procedure contract + invoke helpers | [README](./forst-gen-orpc/README.md) |
| `forst-gen-file-routes` | File-path HTTP registry (Remix / RR / Next catch-all) | [README](./forst-gen-file-routes/README.md) |
| `forst-gen-react-router` | React Router SSR resource routes + page loader helpers | [README](./forst-gen-react-router/README.md) |
| `forst-gen-echo` | Diagnostic manifest (development only) | [README](./forst-gen-echo/README.md) |

## Shared library

Plugin authors can reuse [`internal/genplugin`](../internal/genplugin/) for snapshot walking, route-path conventions, and the stdin/stdout runner.

## Protocol

See [PROTOCOL.md](./PROTOCOL.md) for the v1 stdin/stdout contract. Snapshot goldens live in [`internal/semantic/testdata/`](../internal/semantic/testdata/).

## Versioning

**Not stable yet.**

| What | Today |
| --- | --- |
| Compiler ↔ plugin bundle | Official plugin binaries ship in the same GitHub release tarball as `forst`. `@forst/cli` downloads the whole bundle when `ftconfig.json` lists bare official `cmd` names. |
| Per-plugin semver | None. Each plugin embeds a placeholder generator version (`0.1.0`) in `meta.json` / file headers. `ftconfig.json` has no `version` field per plugin entry. |
| Snapshot contract | `protocolVersion: 1` in the compiler. Breaking snapshot changes bump that number—not individual plugin semver. |
| Custom plugins | You own versioning and deployment (local `cmd` path or your own release process). |

Until plugins publish on their own cadence, treat compiler pin + regenerate + committed outputs as your compatibility strategy.
