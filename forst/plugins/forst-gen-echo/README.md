# forst-gen-echo

**Development-only** plugin that writes a manifest of all type and function ids in the snapshot. Use it to verify `generate.plugins[]` wiring — not for production artifacts.

## Output

- `manifest.txt` — plugin name, protocol version, sorted ids

## Example

```jsonc
{
  "name": "echo",
  "cmd": "forst-gen-echo",
  "out": "generated/echo"
}
```
