# forst-gen-jsonschema

Emits **JSON Schema** (`schema.json`) from Forst type constraint chains. Unknown kinds and type-guard predicates become diagnostics; they never invent `format: email`.

## Output

| File | Description |
| --- | --- |
| `schema.json` | `$defs` for exported package types |
| `meta.json` | Generator metadata |
| `README.txt` | One-line import hint |

## Options (`generate.plugins[].opt`)

| Field | Default | Description |
| --- | --- | --- |
| `draft` | `"2020-12"` | JSON Schema draft (`2019-09`, `2020-12`, `07`) |
| `markers` | unset | If set, only types whose constraint chain includes one of these names |

## Behavior

- Walks `packages[].typeIds`. Named types are `$defs` entries; nested references use `$ref`.
- Method-only `.Router()` shapes are omitted (not JSON data).
- Builtin constraints map to JSON Schema (`Min`/`Max`/`HasPrefix`/`Contains`/`LessThan`/`GreaterThan`/`NotEmpty`/`Value`/…).
- Type-guard predicates (`Email()`, user guards) stay in the snapshot and emit a **warning**.
- `goType`, `channel`, `func`, `unknown` → warning; empty / `not` schema.

## Example

```jsonc
{
  "name": "jsonschema",
  "cmd": "forst-gen-jsonschema",
  "out": "generated/jsonschema",
  "opt": { "draft": "2020-12" }
}
```
