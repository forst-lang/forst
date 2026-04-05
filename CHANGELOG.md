# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
