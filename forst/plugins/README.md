# Forst semantic plugins

Official [`forst generate`](https://forst-lang.org/docs/workflow/cli#forst-generate) plugins read the **semantic snapshot** (JSON on stdin) and write sealed artifacts under `generate.plugins[].out`. User code imports those outputs; never edit them by hand.

These plugins ship in the [Forst repository](https://github.com/forst-lang/forst) and are versioned with the compiler. They are **not** embedded in the compiler — configure them in `ftconfig.json`.

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

Each plugin embeds a `0.1.0` generator version in `meta.json` / file headers. Breaking snapshot changes bump `protocolVersion` in the compiler, not individual plugin semver — until these plugins publish on their own release cadence.
