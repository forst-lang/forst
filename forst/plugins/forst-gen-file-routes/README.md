# forst-gen-file-routes

Emits a **sealed HTTP route registry** from `.Router()` types whose `span.file` lives under `opt.routesRoot`. The URL comes from the Forst file path (`app/` is not part of the URL). Users never edit `out`.

## Output

| File | Description |
| --- | --- |
| `registry.ts` | Path → HTTP methods → `{ package, function, params }` + `dispatch()` |
| `handlers/*.ts` | Per-file invoke wrappers (`GET`/`POST`/…) |
| `routes.ts` | `{ path, module }` array for a user-owned catch-all / `routes.ts` spread |
| `runtime.ts` | `createInvokeClient`, `readBody`, `matchRoute` |
| `meta.json` | Generator metadata |

## Options

| Field | Default | Description |
| --- | --- | --- |
| `markers` | `["Router"]` | Contract marker constraint names |
| `routesRoot` | `"app/api"` | Module-relative directory of route `.ft` files |
| `paramStyle` | `"$id"` | How filenames encode params (`$id` or `[id]`); both parse to `:id` URLs |
| `clientImport` | `"@forst/client"` | Module that exports `createInvokeClient` |

## Convention

| Forst file | URL |
| --- | --- |
| `app/api/orders/place.ft` | `/api/orders/place` |
| `app/api/orders/$id.ft` | `/api/orders/:id` |

Member names are HTTP verbs (`GET`, `POST`, …) or Remix `loader`/`action`. Unknown members → diagnostic. Path params in the filename must match a function parameter or the plugin emits an error.

`dispatch` matches `:param` segments, binds path params then JSON/form/query into positional `invokeFunction` args.

The plugin never writes into `app/`.

## User-owned dispatcher (once)

```ts
// app/api/[[...path]]/route.ts
import { dispatch } from "../../../generated/api/registry";
export const GET = (req: Request) => dispatch("GET", req);
export const POST = (req: Request) => dispatch("POST", req);
```
