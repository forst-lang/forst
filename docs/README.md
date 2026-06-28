# Forst documentation

Public documentation for [Forst](https://github.com/forst-lang/forst), built with [Mintlify](https://mintlify.com).

## Preview locally

Install the [Mintlify CLI](https://www.npmjs.com/package/mint):

```bash
npm i -g mint
```

From this directory:

```bash
cd docs
mint dev
```

Open [http://localhost:3000](http://localhost:3000). The preview updates as you edit MDX files.

## Validate links

```bash
cd docs
mint broken-links
```

## Publishing

Changes deploy via the Mintlify GitHub app when pushed to the default branch (if configured in your Mintlify dashboard).

## Contributing

- Source of truth for feature status: [`ROADMAP.md`](../ROADMAP.md)
- Copy and examples: [`README.md`](../README.md) and [`examples/in/`](../examples/in/)
- Keep experimental features labeled honestly. Match roadmap status.

When adding pages, update [`docs.json`](./docs.json) navigation.

## LLM and agent docs

Mintlify auto-hosts these on deploy (see [For agents](./resources/llms.mdx)):

- `/llms.txt` — auto-generated page index
- `/llms-full.txt` — full corpus (Mintlify-generated; large)
- `/resources/llms` — agent usage cheat sheet (`/resources/llms.md`)

Any doc URL also has a `.md` variant (example: `/quickstart.md`). See [Mintlify llms.txt docs](https://www.mintlify.com/docs/ai/llmstxt).

## Code samples

- Use ` ```forst` for all Forst source, including `.ft` examples on workflow pages.
- Use ` ```go` only for literal Go comparison code (not Forst).
- Use ` ```text` for directory trees and other non-code layout.
- Shell, JSON, and TypeScript keep their native fence tags.

Reuse repeated Forst examples via snippet files in [`snippets/`](./snippets/). Each snippet is a fenced `forst` block. Import with an absolute path:

```mdx
import CatalogOrder from "/snippets/catalog-order.mdx";

<CatalogOrder />
```

Syntax highlighting uses a TextMate grammar at [`languages/forst.tmLanguage.json`](./languages/forst.tmLanguage.json). When Forst keywords change, sync from [`packages/vscode-forst/syntaxes/forst.tmLanguage.json`](../packages/vscode-forst/syntaxes/forst.tmLanguage.json) and keep `"name": "forst"` (lowercase for Mintlify/Shiki fence tags).

Brand icons for cards live in [`icons/`](./icons/). TypeScript logo from [typescriptlang.org](https://www.typescriptlang.org/branding/) (Wikimedia). Go blue wordmark from [go.dev/brand](https://go.dev/brand) (Wikimedia).

The navbar logo and favicon match the VS Code extension **FT** mark in [`packages/vscode-forst/icons/`](../packages/vscode-forst/icons/). When those icons change, sync [`logo/light.svg`](./logo/light.svg), [`logo/dark.svg`](./logo/dark.svg), and [`favicon.svg`](./favicon.svg).

## Internal docs

Adoption and planning notes live in [`adoption/`](./adoption/) (not linked from public nav).
