# @forst/errors

Shared tagged error classes for Forst generated TypeScript clients.

## Exports

- **`@forst/errors`** Promise mode. Uses a small `tagged()` helper. No runtime dependencies.
- **`@forst/errors/effect`** Effect mode. Uses `Data.TaggedError`. Requires the `effect` peer.

## What's included

Invoke transport failures (`InvokeRejected`, `InvokeHttpFailure`, …), test harness failures (`ForstTestServerFailed`), and the domain catch-all (`ForstUnknownFailure`).

Built-in `_tag` values are namespaced under `@forst/errors/` (for example `@forst/errors/InvokeRejected`).

Shared invoke, harness, and unknown-failure classes live in this package. Import them directly (`@forst/errors` in Promise mode, `@forst/errors/effect` in Effect mode). Generated clients expose optional domain error namespaces from `@forst/gen/$errors`.

## Usage

Prefer matching on `_tag` in Effect code (`Effect.catchTag`). In Promise code, `instanceof` works because every generated client shares the same class definitions from this package.

```typescript
import { isInvokeFailure, InvokeRejected } from "@forst/errors";
import { $CellTaken } from "@forst/tictactoe/main/errors";

try {
  await client.main.Move({ row: 0, col: 0 });
} catch (error) {
  if (error instanceof InvokeRejected) {
    // transport failure
  }
  if (error instanceof $CellTaken) {
    // domain error
  }
}
```

Domain errors (`error CellTaken { … }`) live in `@forst/gen/<forstPackage>/errors` for each project (also under `@forst/gen/$errors` as package namespaces).

## Development

From the monorepo:

```bash
cd packages/errors
bun run build
bun test
```

## Publishing

Release Please tags `errors-v*` bump `package.json` and `jsr.json`. CI publishes to npm and JSR via [.github/workflows/publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml).

Manual publish from `packages/errors`:

```bash
bun run build
npm publish --access public --workspaces=false
npx jsr publish
```

Dry run:

```bash
bun run pack:dry
npx jsr publish --dry-run
```

### npm trusted publishing (one-time)

Before CI can publish, add a **trusted publisher** for `@forst/errors` on [npmjs.com](https://www.npmjs.com) (same settings as `@forst/cli`):

- Repository: `forst-lang/forst`
- Workflow: `publish-packages.yml`
- Environment: (none, unless you use one for other packages)

## License

MIT. See [LICENSE](./LICENSE).
