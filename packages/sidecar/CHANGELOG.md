# Changelog

## [0.1.17](https://github.com/forst-lang/forst/compare/sidecar-v0.1.16...sidecar-v0.1.17) (2026-04-19)


### Bug Fixes

* **release:** add --provenance to npm publish commands ([6e9beac](https://github.com/forst-lang/forst/commit/6e9beac63d5aa6d0dd541fb655ed4a0333fd58bd))

## [0.1.16](https://github.com/forst-lang/forst/compare/sidecar-v0.1.15...sidecar-v0.1.16) (2026-04-19)


### Bug Fixes

* **release:** Further adjust release jobs for NPM ([8f9466f](https://github.com/forst-lang/forst/commit/8f9466fe82a44f09112e02e8c2fd58ee19a78d35))

## [0.1.15](https://github.com/forst-lang/forst/compare/sidecar-v0.1.14...sidecar-v0.1.15) (2026-04-19)


### Bug Fixes

* **release:** Restore permissions for NPM release ([3c2650b](https://github.com/forst-lang/forst/commit/3c2650b20d10052cfddca9383c697511b1f87da4))

## [0.1.14](https://github.com/forst-lang/forst/compare/sidecar-v0.1.13...sidecar-v0.1.14) (2026-04-19)


### Bug Fixes

* **release:** align npm auth with GitHub Node.js publish guide ([ac613b3](https://github.com/forst-lang/forst/commit/ac613b3683dc95cd8bd7bacb2ff493397e3e52fd))

## [0.1.13](https://github.com/forst-lang/forst/compare/sidecar-v0.1.12...sidecar-v0.1.13) (2026-04-19)


### Bug Fixes

* **cli:** document Node 18+ in README ([67e7abc](https://github.com/forst-lang/forst/commit/67e7abc1d9db416a11f9d0a5b1429a0c2ad88b6f))
* **release:** npm whoami and --workspaces=false ([67e7abc](https://github.com/forst-lang/forst/commit/67e7abc1d9db416a11f9d0a5b1429a0c2ad88b6f))

## [0.1.12](https://github.com/forst-lang/forst/compare/sidecar-v0.1.11...sidecar-v0.1.12) (2026-04-18)


### Bug Fixes

* **release:** run sidecar publish when CLI job is skipped ([594962c](https://github.com/forst-lang/forst/commit/594962c5a15724ba28ffbf55ede92bfe8b259f20))

## [0.1.11](https://github.com/forst-lang/forst/compare/sidecar-v0.1.10...sidecar-v0.1.11) (2026-04-18)


### Bug Fixes

* **sidecar:** keep registry @forst/cli off workspace:* ([31763bf](https://github.com/forst-lang/forst/commit/31763bf55e1a26049aa9c3b5c051ba0e63cdc822))

## [0.1.10](https://github.com/forst-lang/forst/compare/sidecar-v0.1.9...sidecar-v0.1.10) (2026-04-06)


### Bug Fixes

* **sidecar:** harden prepublish (local bun install, build CLI before tsc) ([614a697](https://github.com/forst-lang/forst/commit/614a697ea47e07003bd9351b322d08c9d4bf3055))
* **vscode-forst:** copy hoisted @forst/cli before vsce package ([614a697](https://github.com/forst-lang/forst/commit/614a697ea47e07003bd9351b322d08c9d4bf3055))

## [0.1.9](https://github.com/forst-lang/forst/compare/sidecar-v0.1.8...sidecar-v0.1.9) (2026-04-06)


### Features

* **cli:** add --version/-V to npm shim ([490c8e5](https://github.com/forst-lang/forst/commit/490c8e5211b83cb3c7a15b2c3218ff5ace597b0a))
* **sidecar:** add invalidateFunctionCache on ForstSidecarClient ([490c8e5](https://github.com/forst-lang/forst/commit/490c8e5211b83cb3c7a15b2c3218ff5ace597b0a))
* **vscode-forst:** add Copy debug info to clipboard command ([490c8e5](https://github.com/forst-lang/forst/commit/490c8e5211b83cb3c7a15b2c3218ff5ace597b0a))


### Bug Fixes

* **ci:** pass --allow-dirty to jsr publish after rewrite + bun install ([d08da05](https://github.com/forst-lang/forst/commit/d08da051f0e4080a0ceedbc7ec9cfa27559e86ee))
* **sidecar:** read package version without import.meta for CJS build ([d08da05](https://github.com/forst-lang/forst/commit/d08da051f0e4080a0ceedbc7ec9cfa27559e86ee))

## [0.1.8](https://github.com/forst-lang/forst/compare/sidecar-v0.1.7...sidecar-v0.1.8) (2026-04-06)


### Features

* **sidecar:** send X-Forst-Sidecar-Version on dev server HTTP requests ([47c9698](https://github.com/forst-lang/forst/commit/47c969808b6b8bb68f7ccbd4cc6d8f19e24ebc2d))
* **vscode-forst:** include extension semver in the activation log line. ([47c9698](https://github.com/forst-lang/forst/commit/47c969808b6b8bb68f7ccbd4cc6d8f19e24ebc2d))

## [0.1.7](https://github.com/forst-lang/forst/compare/sidecar-v0.1.6...sidecar-v0.1.7) (2026-04-06)


### Features

* **cli:** gate binary download and expose release helpers ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **sidecar:** optional compiler download via config and env ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **vscode-forst:** resolve `forst` via @forst/cli and `forst.downloadCompiler` ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))

## [0.1.6](https://github.com/forst-lang/forst/compare/sidecar-v0.1.5...sidecar-v0.1.6) (2026-04-06)


### Features

* **compiler:** add exportable struct fields for JSON and TS wire alignment ([a7cce2b](https://github.com/forst-lang/forst/commit/a7cce2b17da45c13862954721d3bcf421c107afd))


### Bug Fixes

* **sidecar:** avoid unhandled rejections in version check and shutdown ([a7cce2b](https://github.com/forst-lang/forst/commit/a7cce2b17da45c13862954721d3bcf421c107afd))

## [0.1.5](https://github.com/forst-lang/forst/compare/sidecar-v0.1.4...sidecar-v0.1.5) (2026-04-06)


### Features

* **compiler:** variable slice indexing and indexed assignment ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **compiler:** variable slice indexing and indexed assignment ([#57](https://github.com/forst-lang/forst/issues/57)) ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **examples:** add tic-tac-toe multi-file sample ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **examples:** tic-tac-toe uses ensure constraints and `(MoveResponse, Error)` ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **generate:** merged-package typecheck before emitting TypeScript ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **lsp:** hover dotted fields, receiver methods, and '(' after recv.method ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **typechecker:** collect defs before functions; builtin Go doc for hovers ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))


### Bug Fixes

* **parser:** unary minus, negative literals, and Error built-in type ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **transformer:** correct Go types for slices and named returns ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **transformer:** do not pad `return g()` with `, nil` for multi-value calls ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **typechecker:** infer full tuple from `return f()` when arity matches ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))

## [0.1.4](https://github.com/forst-lang/forst/compare/sidecar-v0.1.3...sidecar-v0.1.4) (2026-04-05)


### Bug Fixes

* **sidecar:** depend on @forst/cli via workspace protocol ([0762ce6](https://github.com/forst-lang/forst/commit/0762ce6996e59f5e5e4fd715384478e29cd0aca5))

## [0.1.3](https://github.com/forst-lang/forst/compare/sidecar-v0.1.2...sidecar-v0.1.3) (2026-04-05)


### Features

* **cli:** add @forst/cli package and wire sidecar to shared resolver ([#51](https://github.com/forst-lang/forst/issues/51)) ([e569101](https://github.com/forst-lang/forst/commit/e569101aaa34ef4a6be14c4541559a65e314a2f5))

## [0.1.2](https://github.com/forst-lang/forst/compare/sidecar-v0.1.1...sidecar-v0.1.2) (2026-04-05)


### Features

* **sidecar:** add GET /version, version check, and generateTypes hook ([ab5516f](https://github.com/forst-lang/forst/commit/ab5516f4aef36b568f071293bf4741dc25816af4))
* **sidecar:** ftconfig-aware generate and stronger dev HTTP client ([caa20af](https://github.com/forst-lang/forst/commit/caa20af409f4fb0098500856bc01287c3bcf71aa))
* **sidecar:** monorepo-friendly connect mode, ftconfig path, and chokidar watch ([ab776d4](https://github.com/forst-lang/forst/commit/ab776d46809b7b71cbc38c4dff7464266353eb32))

## [0.1.1](https://github.com/forst-lang/forst/compare/sidecar-v0.1.0...sidecar-v0.1.1) (2026-04-05)


### Features

* add extension, syntax highlighting, and LSP editor support ([#36](https://github.com/forst-lang/forst/issues/36)) ([049133e](https://github.com/forst-lang/forst/commit/049133e28a95a24749a22fd1bc03002924607489))
* Basic Type Narrowing ([#42](https://github.com/forst-lang/forst/issues/42)) ([fc4977c](https://github.com/forst-lang/forst/commit/fc4977cfbd06ad35ae63ed8767847d1fbae68e7a))
* Generate TypeScript types ([#30](https://github.com/forst-lang/forst/issues/30)) ([2d8349d](https://github.com/forst-lang/forst/commit/2d8349d5f2c0491053994f6867d6d6faba6743e5))
* JS/TS integration ([#25](https://github.com/forst-lang/forst/issues/25)) ([bbe49bc](https://github.com/forst-lang/forst/commit/bbe49bccb192d3dd7b8de3d89a9cd6296f355d44))
* **sidecar:** ForstError hierarchy, dev HTTP contract tests, RFC doc renames ([a53e41d](https://github.com/forst-lang/forst/commit/a53e41d0b03628710f59e79e64081d40382e4717))
