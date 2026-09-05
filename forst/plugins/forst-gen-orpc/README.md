# forst-gen-orpc

Emits a **procedure contract** from types marked with `.Router()` (or `opt.markers`). Default style is oRPC (`oc` + Zod). `opt.style: "trpc"` emits an `initTRPC` router that invokes Forst positionally.

## Output

| File | Style | Description |
| --- | --- | --- |
| `zod.ts` | both | Zod schemas for inputs, outputs, nominal errors |
| `invoke.ts` | both | `invokePositional` / `invokeStream` via `createInvokeClient().invokeFunction` |
| `contract.ts` | `orpc` | Nested `oc.input().output().errors()` plus `*Implement` adapters |
| `router.ts` | `trpc` | `publicProcedure.input().mutation/query/subscription` |
| `meta.json` | both | Generator metadata |

## Options

| Field | Default | Description |
| --- | --- | --- |
| `markers` | `["Router"]` | Constraint names that mark a contract type |
| `queries` | `[]` | `{typeId}.{field}` keys treated as queries |
| `routes` | `{}` | Optional HTTP metadata per `{typeId}.{field}` (`method`, `path`) |
| `style` | `"orpc"` | `"orpc"` or `"trpc"` |
| `clientImport` | `"@forst/client"` | Module that exports `createInvokeClient` |

Query vs mutation: `Query` / `Mutation` on the member type, else `opt.queries`, else mutation. `returns[0]` kind `channel` → subscription (`eventIterator` / `.subscription`). Helpers that are not fields on the marked type are excluded.

Generated code imports `zod` and either `@orpc/contract` or `@trpc/server`. Install those in the app; the plugin does not vendor them.

## Example

```jsonc
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
}
```
