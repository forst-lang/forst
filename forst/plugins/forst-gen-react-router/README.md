# forst-gen-react-router

SSR plugin for **React Router / Remix resource routes**. Forst stays data/API; React stays in the user's `app/` tree. This plugin writes only under `out`. Regenerating `out` is safe.

## Output

| File | Owner |
| --- | --- |
| `routes.ts` | plugin — `forstApiRoutes` using `route()` from `@react-router/dev/routes` |
| `handlers/*.ts` | plugin — resource modules (`loader` / `action`, **no** `default` component) |
| `loaders.ts` | plugin — named `load{Type}{Method}` helpers for user page modules to import |
| `registry.ts` | plugin — path → handler metadata |
| `runtime.ts` | plugin — `readBody` (+ invoke client when `opt.invoke` is `"client"`) |
| `meta.json` | plugin |

Do **not** generate `app/root.tsx`, `app/routes.ts`, or any file that still needs a UI component. User pages import loaders; user `routes.ts` spreads `forstApiRoutes`.

## Options

| Field | Default | Description |
| --- | --- | --- |
| `markers` | `["Router"]` | Contract marker names |
| `routesRoot` | `"app/api"` | `.ft` files that become resource routes |
| `paramStyle` | `"$id"` | Filename param convention |
| `invoke` | `"package"` | `"package"` → `$pkg.Fn` from `@forst/gen/<pkg>`; `"client"` → `invokeFunction` |
| `clientImport` | `"@forst/gen"` | Package prefix (`@forst/gen/ordersid`) or client module when `invoke` is `"client"` |
| `routeImport` | `"@react-router/dev/routes"` | Module that exports `route()` |

GET/HEAD/`loader` → `loader`. POST/PUT/PATCH/DELETE/`action` → `action`. Channel returns and `runnable: false` are skipped with diagnostics.

## Config

```jsonc
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
```

## User-owned composition (once)

```ts
// app/routes.ts — USER OWNED
import { route } from "@react-router/dev/routes";
import { forstApiRoutes } from "../generated/rr/routes";

export default [
  ...forstApiRoutes,
  route("/", "routes/_index.tsx"),
];
```

```tsx
// app/routes/_index.tsx — USER OWNED
import { loadRoutesGET } from "../../generated/rr/loaders";

export async function loader() {
  return loadRoutesGET();
}

export default function Index() {
  return <main>{/* user UI */}</main>;
}
```
