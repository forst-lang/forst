# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1](https://github.com/forst-lang/forst/compare/v0.2.0...v0.2.1) (2026-04-16)


### Bug Fixes

* **lsp:** repair for-loop navigation hashing and else-if completion tokens ([11f9018](https://github.com/forst-lang/forst/commit/11f90183dd4e54db10dd63493e8b77a5bf85282c))

## [0.2.0](https://github.com/forst-lang/forst/compare/v0.1.0...v0.2.0) (2026-04-12)


### ⚠ BREAKING CHANGES

* **typechecker:** add union/intersection IR; typedef semantics ([#74](https://github.com/forst-lang/forst/issues/74))

### Features

* **lsp:** qualify guard hovers using the inferred subject type ([3d47e0f](https://github.com/forst-lang/forst/commit/3d47e0fad60ecf1c462c498938dbad41bd37aa28))
* **typechecker:** add union/intersection IR; typedef semantics ([#74](https://github.com/forst-lang/forst/issues/74)) ([6087988](https://github.com/forst-lang/forst/commit/6087988b3bc1198f3654df916a3e9a6c81994ce3))
* **vscode-forst:** highlight nominal `error Name` and core builtin types ([3d47e0f](https://github.com/forst-lang/forst/commit/3d47e0fad60ecf1c462c498938dbad41bd37aa28))


### Bug Fixes

* **printer:** format `ensure` error alternatives and nested statement indentation ([3d47e0f](https://github.com/forst-lang/forst/commit/3d47e0fad60ecf1c462c498938dbad41bd37aa28))
* **transformer:** map narrowed Result locals to split Go names ([3d47e0f](https://github.com/forst-lang/forst/commit/3d47e0fad60ecf1c462c498938dbad41bd37aa28))

## [0.1.0](https://github.com/forst-lang/forst/compare/v0.0.30...v0.1.0) (2026-04-12)


### ⚠ BREAKING CHANGES

* **compiler:** nominal error types (`error X { … }`) ([#70](https://github.com/forst-lang/forst/issues/70))

### Features

* **compiler:** nominal error types (`error X { … }`) ([#70](https://github.com/forst-lang/forst/issues/70)) ([58842e7](https://github.com/forst-lang/forst/commit/58842e7c6d65478b89b69629a3afd6e1a866264f))
* **lsp:** add hover dictionaries for built-in types and guards ([#72](https://github.com/forst-lang/forst/issues/72)) ([23a07ea](https://github.com/forst-lang/forst/commit/23a07ea1f2f01d1b464fb4bb6294c2dedf188def))


### Bug Fixes

* **typechecker:** match hash shape literals to missing named param types ([30b5ee2](https://github.com/forst-lang/forst/commit/30b5ee2eaf560da7e8764eebae355ebcc8edf439))

## [0.0.30](https://github.com/forst-lang/forst/compare/v0.0.29...v0.0.30) (2026-04-12)


### Features

* **tictactoe:** colocate wire types and finish ApplyMove rules ([50bba0d](https://github.com/forst-lang/forst/commit/50bba0d0317fa3e10dcbdd03872e64e5ff6ba2c1))
* **transformer:** extend ensure builtins coverage (ensure_builtins + tests) ([50bba0d](https://github.com/forst-lang/forst/commit/50bba0d0317fa3e10dcbdd03872e64e5ff6ba2c1))


### Bug Fixes

* **generate:** align client index expectation with `export type * from './types.d.ts'` ([50bba0d](https://github.com/forst-lang/forst/commit/50bba0d0317fa3e10dcbdd03872e64e5ff6ba2c1))
* **generate:** emit type-only re-export for merged client index ([0e11e55](https://github.com/forst-lang/forst/commit/0e11e55a92b8f865e4e1f2a5c323cdec212f40dd))
* **lsp:** pass dotted span into FieldHoverMarkdown for narrowing display ([50bba0d](https://github.com/forst-lang/forst/commit/50bba0d0317fa3e10dcbdd03872e64e5ff6ba2c1))
* **typechecker:** compound ensure subjects and field-path hover narrowing ([50bba0d](https://github.com/forst-lang/forst/commit/50bba0d0317fa3e10dcbdd03872e64e5ff6ba2c1))

## [0.0.29](https://github.com/forst-lang/forst/compare/v0.0.28...v0.0.29) (2026-04-06)


### Bug Fixes

* **vscode-forst:** import debugInfo from compiled `out/` in node tests ([fb726fe](https://github.com/forst-lang/forst/commit/fb726fec60e0b3b5a1d137498d3d7cd131c4a8b4))

## [0.0.28](https://github.com/forst-lang/forst/compare/v0.0.27...v0.0.28) (2026-04-06)


### Bug Fixes

* **vscode-forst:** package VSIX from isolated stage for vsce ([3b2c4f6](https://github.com/forst-lang/forst/commit/3b2c4f661494eff78b816dbd8b73f54c2a88ce02))

## [0.0.27](https://github.com/forst-lang/forst/compare/v0.0.26...v0.0.27) (2026-04-06)


### Bug Fixes

* **sidecar:** harden prepublish (local bun install, build CLI before tsc) ([614a697](https://github.com/forst-lang/forst/commit/614a697ea47e07003bd9351b322d08c9d4bf3055))
* **vscode-forst:** copy hoisted @forst/cli before vsce package ([614a697](https://github.com/forst-lang/forst/commit/614a697ea47e07003bd9351b322d08c9d4bf3055))

## [0.0.26](https://github.com/forst-lang/forst/compare/v0.0.25...v0.0.26) (2026-04-06)


### Bug Fixes

* **ci:** pass --allow-dirty to jsr publish after rewrite + bun install ([d08da05](https://github.com/forst-lang/forst/commit/d08da051f0e4080a0ceedbc7ec9cfa27559e86ee))
* **sidecar:** read package version without import.meta for CJS build ([d08da05](https://github.com/forst-lang/forst/commit/d08da051f0e4080a0ceedbc7ec9cfa27559e86ee))

## [0.0.25](https://github.com/forst-lang/forst/compare/v0.0.24...v0.0.25) (2026-04-06)


### Features

* **sidecar:** send X-Forst-Sidecar-Version on dev server HTTP requests ([47c9698](https://github.com/forst-lang/forst/commit/47c969808b6b8bb68f7ccbd4cc6d8f19e24ebc2d))
* **vscode-forst:** include extension semver in the activation log line. ([47c9698](https://github.com/forst-lang/forst/commit/47c969808b6b8bb68f7ccbd4cc6d8f19e24ebc2d))

## [0.0.24](https://github.com/forst-lang/forst/compare/v0.0.23...v0.0.24) (2026-04-06)


### Features

* **cli:** gate binary download and expose release helpers ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **sidecar:** optional compiler download via config and env ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **vscode-forst:** resolve `forst` via @forst/cli and `forst.downloadCompiler` ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))


### Bug Fixes

* **printer:** print slice types as []T instead of [T] ([6627c72](https://github.com/forst-lang/forst/commit/6627c72a32e56a2adeca104cca882465852b629c))

## [0.0.23](https://github.com/forst-lang/forst/compare/v0.0.22...v0.0.23) (2026-04-06)


### Features

* **compiler:** add exportable struct fields for JSON and TS wire alignment ([a7cce2b](https://github.com/forst-lang/forst/commit/a7cce2b17da45c13862954721d3bcf421c107afd))
* **executor:** rewrite merged package main to importable forstexec in temp modules ([23ba6ae](https://github.com/forst-lang/forst/commit/23ba6ae5b2369dcb4d7b62a10c646776507dd50a))


### Bug Fixes

* **sidecar:** avoid unhandled rejections in version check and shutdown ([a7cce2b](https://github.com/forst-lang/forst/commit/a7cce2b17da45c13862954721d3bcf421c107afd))
* **tictactoe:** resolve repo-root `bin/forst` in forst-run smoke test ([23ba6ae](https://github.com/forst-lang/forst/commit/23ba6ae5b2369dcb4d7b62a10c646776507dd50a))

## [0.0.22](https://github.com/forst-lang/forst/compare/v0.0.21...v0.0.22) (2026-04-06)


### Features

* **compiler:** add defer and go statements with Go-compatible rules ([780b61b](https://github.com/forst-lang/forst/commit/780b61b3b555933dfea3d000e046287377f7ac9e))
* **compiler:** Go builtins, map literals, and slice ]T parse fix ([#56](https://github.com/forst-lang/forst/issues/56)) ([30deb88](https://github.com/forst-lang/forst/commit/30deb88b9b8a13f2fa3b37654f153c0cb7fb3b9d))
* **compiler:** variable slice indexing and indexed assignment ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **compiler:** variable slice indexing and indexed assignment ([#57](https://github.com/forst-lang/forst/issues/57)) ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **examples:** add tic-tac-toe multi-file sample ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **examples:** tic-tac-toe uses ensure constraints and `(MoveResponse, Error)` ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **generate:** merged-package typecheck before emitting TypeScript ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **lsp:** hover dotted fields, receiver methods, and '(' after recv.method ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **typechecker:** collect defs before functions; builtin Go doc for hovers ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **typechecker:** flow merge, typeops, and predicate-chain hover ([#55](https://github.com/forst-lang/forst/issues/55)) ([f2024df](https://github.com/forst-lang/forst/commit/f2024df5cddf3b349567c4ee5c93fccc19bbacc2))


### Bug Fixes

* **ci:** fix Release workflow validation and allow manual runs ([e262b3a](https://github.com/forst-lang/forst/commit/e262b3a43cdfb0e6b09cc7ec1a89a1f086bedef6))
* **ci:** run Release Please PR step before tagging ([af2ab1e](https://github.com/forst-lang/forst/commit/af2ab1e9962f66404a1f2d75896f4605ae2caeb6))
* **parser:** unary minus, negative literals, and Error built-in type ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **transformer:** correct Go types for slices and named returns ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **transformer:** do not pad `return g()` with `, nil` for multi-value calls ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))
* **typechecker:** infer full tuple from `return f()` when arity matches ([803caf0](https://github.com/forst-lang/forst/commit/803caf0cd28e1e6e7e292f4dfef23de6bba74b49))

## [0.0.21](https://github.com/forst-lang/forst/compare/v0.0.20...v0.0.21) (2026-04-05)


### Bug Fixes

* **sidecar:** depend on @forst/cli via workspace protocol ([0762ce6](https://github.com/forst-lang/forst/commit/0762ce6996e59f5e5e4fd715384478e29cd0aca5))

## [0.0.20](https://github.com/forst-lang/forst/compare/v0.0.19...v0.0.20) (2026-04-05)


### Features

* **cli:** add @forst/cli package and wire sidecar to shared resolver ([#51](https://github.com/forst-lang/forst/issues/51)) ([e569101](https://github.com/forst-lang/forst/commit/e569101aaa34ef4a6be14c4541559a65e314a2f5))

## [0.0.19](https://github.com/forst-lang/forst/compare/v0.0.18...v0.0.19) (2026-04-05)


### Features

* **sidecar:** add GET /version, version check, and generateTypes hook ([ab5516f](https://github.com/forst-lang/forst/commit/ab5516f4aef36b568f071293bf4741dc25816af4))
* **sidecar:** ftconfig-aware generate and stronger dev HTTP client ([caa20af](https://github.com/forst-lang/forst/commit/caa20af409f4fb0098500856bc01287c3bcf71aa))
* **sidecar:** monorepo-friendly connect mode, ftconfig path, and chokidar watch ([ab776d4](https://github.com/forst-lang/forst/commit/ab776d46809b7b71cbc38c4dff7464266353eb32))

## [0.0.18](https://github.com/forst-lang/forst/compare/v0.0.17...v0.0.18) (2026-04-05)


### Features

* **lsp:** rename, format code actions, Go-style params, VSIX icon ([6343445](https://github.com/forst-lang/forst/commit/63434455a92f926a108b08b076dd0630bf637a47))

## [0.0.17](https://github.com/forst-lang/forst/compare/v0.0.16...v0.0.17) (2026-04-05)


### Features

* **lsp:** refined hovers with type guard docs and same-package merge ([3637950](https://github.com/forst-lang/forst/commit/3637950b16707626afe609d9390b84819e5de2de))

## [0.0.16](https://github.com/forst-lang/forst/compare/v0.0.15...v0.0.16) (2026-04-05)


### Bug Fixes

* **release:** sync VS Code extension version with release-please ([a5e2ff1](https://github.com/forst-lang/forst/commit/a5e2ff1eedfbd53e5b322a2fc1f1ae16ce34b2b8))

## [0.0.15](https://github.com/forst-lang/forst/compare/v0.0.14...v0.0.15) (2026-04-05)


### Features

* **fmt:** add pretty-printer, forst fmt, and LSP formatting ([cf9b098](https://github.com/forst-lang/forst/commit/cf9b098e921435a7e19b82acb1c1fbc225ad570b))

## [0.0.14](https://github.com/forst-lang/forst/compare/v0.0.13...v0.0.14) (2026-04-05)


### Features

* **vscode-forst:** add Forst editor snippets and snippet JSON test ([e5eec47](https://github.com/forst-lang/forst/commit/e5eec47f29fccea9999e0b3aeef868074ab9a010))

## [0.0.13](https://github.com/forst-lang/forst/compare/v0.0.12...v0.0.13) (2026-04-05)


### Features

* Basic Type Narrowing ([#42](https://github.com/forst-lang/forst/issues/42)) ([fc4977c](https://github.com/forst-lang/forst/commit/fc4977cfbd06ad35ae63ed8767847d1fbae68e7a))
* **sidecar:** ForstError hierarchy, dev HTTP contract tests, RFC doc renames ([a53e41d](https://github.com/forst-lang/forst/commit/a53e41d0b03628710f59e79e64081d40382e4717))

## [0.0.12](https://github.com/forst-lang/forst/compare/v0.0.11...v0.0.12) (2026-04-05)


### Features

* **examples:** add merged-package imports test and imports example ([0f3e2dd](https://github.com/forst-lang/forst/commit/0f3e2ddcfbc50197deab8355496bd8dc89cece0d))
* **lsp:** add package-merge analysis and cross-file features ([36b9064](https://github.com/forst-lang/forst/commit/36b9064e4d31e1c48623a8a185b61087263a93f8))


### Bug Fixes

* **lsp:** robust URIs and cache invalidation ([c0b3b65](https://github.com/forst-lang/forst/commit/c0b3b65c697797745f8c88d61587eb8e235d4207))

## [0.0.11](https://github.com/forst-lang/forst/compare/v0.0.10...v0.0.11) (2026-04-05)


### Features

* add Go-style for loops, range, and break/continue ([802a163](https://github.com/forst-lang/forst/commit/802a1635c2dc3b31e37b9ccec4f13b238f13f094))
* go/packages interop, qualified-call diagnostics, and LSP hover for Go imports ([a255670](https://github.com/forst-lang/forst/commit/a255670974f297cd2265e84411d5fe121a68368d))
* **lsp:** add document symbols, references, and workspace symbol search ([#38](https://github.com/forst-lang/forst/issues/38)) ([5501834](https://github.com/forst-lang/forst/commit/55018341f47549b7647e563b0482181d54022f1d))
* **lsp:** add semantic completion with zones, locals, and member access ([98b02bb](https://github.com/forst-lang/forst/commit/98b02bb7b3098cd7b1487bc8c985cfce4a937590))
* **lsp:** navigate locals, folding, completion, and formatting ([#40](https://github.com/forst-lang/forst/issues/40)) ([b4c559c](https://github.com/forst-lang/forst/commit/b4c559cfaaac8d6f1d96a092b09e07aea16a6245))


### Bug Fixes

* Add missing example ([4d78e3f](https://github.com/forst-lang/forst/commit/4d78e3fce3a3e9e0d1a7279c7ac7bd0e794b1767))

## [0.0.10](https://github.com/forst-lang/forst/compare/v0.0.9...v0.0.10) (2026-04-03)


### Features

* **lsp:** implement same-file go-to-definition for top-level symbols ([1eb03fd](https://github.com/forst-lang/forst/commit/1eb03fd60a122d0fe4075d87578870ae399644f1))

## [0.0.9](https://github.com/forst-lang/forst/compare/v0.0.8...v0.0.9) (2026-04-03)


### Features

* add extension, syntax highlighting, and LSP editor support ([#36](https://github.com/forst-lang/forst/issues/36)) ([049133e](https://github.com/forst-lang/forst/commit/049133e28a95a24749a22fd1bc03002924607489))
* **ts:** improve TS emission and fail on unknown types; fix generate exit codes ([cc5f33c](https://github.com/forst-lang/forst/commit/cc5f33c33fd9165990e5e1a10fa2a0f6b734ba10))

## [0.0.8](https://github.com/forst-lang/forst/compare/v0.0.7...v0.0.8) (2026-03-25)


### Features

* Generate TypeScript types ([#30](https://github.com/forst-lang/forst/issues/30)) ([2d8349d](https://github.com/forst-lang/forst/commit/2d8349d5f2c0491053994f6867d6d6faba6743e5))
* **ts:** emit richer types from checker shapes, assertions, and Go idents ([71de993](https://github.com/forst-lang/forst/commit/71de993d6c208ff4598e04ac1a9894426dbef344))

## [0.0.7](https://github.com/forst-lang/forst/compare/v0.0.6...v0.0.7) (2025-07-20)


### Features

* Add `forst dump` and `forst lsp` for debugging ([#31](https://github.com/forst-lang/forst/issues/31)) ([7e09ba7](https://github.com/forst-lang/forst/commit/7e09ba79b39fc40be8232dd35057370090d9feb7))


### Bug Fixes

* **ci:** add authentication to arduino/setup-task to avoid rate limits ([2660bf7](https://github.com/forst-lang/forst/commit/2660bf72b91bd4507d8b4bf983b8566c0cb73db5))

## [0.0.6](https://github.com/forst-lang/forst/compare/v0.0.5...v0.0.6) (2025-07-11)


### Features

* JS/TS integration ([#25](https://github.com/forst-lang/forst/issues/25)) ([bbe49bc](https://github.com/forst-lang/forst/commit/bbe49bccb192d3dd7b8de3d89a9cd6296f355d44))

## [0.0.5](https://github.com/forst-lang/forst/compare/v0.0.4...v0.0.5) (2025-07-06)


### Bug Fixes

* Enhanced Field Merging and Constraint Processing ([#26](https://github.com/forst-lang/forst/issues/26)) ([2bf7c65](https://github.com/forst-lang/forst/commit/2bf7c65fae267dbe614586c3d264e95492ce6755))

## [0.0.4](https://github.com/forst-lang/forst/compare/v0.0.3...v0.0.4) (2025-07-06)


### Bug Fixes

* Provide GH_TOKEN variable to enable binary upload ([63e3dda](https://github.com/forst-lang/forst/commit/63e3ddaf1d0c5cc42d3c0fd0d68b7b0f2024028e))

## [0.0.3](https://github.com/forst-lang/forst/compare/v0.0.2...v0.0.3) (2025-07-06)


### Bug Fixes

* Build binaries and upload them to release ([6e99002](https://github.com/forst-lang/forst/commit/6e9900251d975d7f5f27961f7ced49d0013fd065))

## [0.0.2](https://github.com/forst-lang/forst/compare/v0.0.1...v0.0.2) (2025-07-06)


### Bug Fixes

* Upload binaries to GitHub release ([aba5c3b](https://github.com/forst-lang/forst/commit/aba5c3bd51067b996f2b1c77a6f285f874334bf2))

## [0.0.1](https://github.com/forst-lang/forst/compare/v0.0.0...v0.0.1) (2025-07-06)


### Features

* Add control flow, references, and map literals ([#13](https://github.com/forst-lang/forst/issues/13)) ([89c7283](https://github.com/forst-lang/forst/commit/89c72833708430fe13cd12ab2dceed62757c2d4b))
* Add type guards and improve type handling ([#9](https://github.com/forst-lang/forst/issues/9)) ([10d99ee](https://github.com/forst-lang/forst/commit/10d99ee72c4e7d5c8f68823bcb052db944da799e))
* **lexer:** improve token handling and standardization ([#15](https://github.com/forst-lang/forst/issues/15)) ([925cb55](https://github.com/forst-lang/forst/commit/925cb557a2299cd4a5267ebeb5f2512422b84709))
* Shape guards ([#17](https://github.com/forst-lang/forst/issues/17)) ([afe5b12](https://github.com/forst-lang/forst/commit/afe5b126ffb7d75af079e285e3de346a3f2d1135))
* **transformer:** improve assertion handling and error messages ([#12](https://github.com/forst-lang/forst/issues/12)) ([c7f5f4f](https://github.com/forst-lang/forst/commit/c7f5f4fe33f28a7f0580f882b3f701c012823bd2))


### Bug Fixes

* add proper release-please manifest file ([b91597a](https://github.com/forst-lang/forst/commit/b91597aa7da0a45256d3ea0e426584459c30b6de))
* Mapping of packages to versions ([9e34699](https://github.com/forst-lang/forst/commit/9e34699a540615e540e7c4d9ed78bffc559c332b))
* Mapping of packages to versions ([3036a10](https://github.com/forst-lang/forst/commit/3036a10b220100d7e0c73737475bae15b55b3ffc))
* Use correct working directory ([65b2bcd](https://github.com/forst-lang/forst/commit/65b2bcd611773cc682443875e5702911def06527))

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security
