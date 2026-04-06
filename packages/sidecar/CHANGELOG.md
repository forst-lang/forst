# Changelog

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
