# Changelog

## [0.1.4](https://github.com/forst-lang/forst/compare/vscode-forst-v0.1.3...vscode-forst-v0.1.4) (2026-04-06)


### Features

* **sidecar:** send X-Forst-Sidecar-Version on dev server HTTP requests ([47c9698](https://github.com/forst-lang/forst/commit/47c969808b6b8bb68f7ccbd4cc6d8f19e24ebc2d))
* **vscode-forst:** include extension semver in the activation log line. ([47c9698](https://github.com/forst-lang/forst/commit/47c969808b6b8bb68f7ccbd4cc6d8f19e24ebc2d))

## [0.1.3](https://github.com/forst-lang/forst/compare/vscode-forst-v0.1.2...vscode-forst-v0.1.3) (2026-04-06)


### Features

* **cli:** gate binary download and expose release helpers ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **sidecar:** optional compiler download via config and env ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **vscode-forst:** resolve `forst` via @forst/cli and `forst.downloadCompiler` ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))

## [0.1.2](https://github.com/forst-lang/forst/compare/vscode-forst-v0.1.1...vscode-forst-v0.1.2) (2026-04-06)


### Features

* **vscode:** add language icon for .ft tab and explorer ([a4cbd43](https://github.com/forst-lang/forst/commit/a4cbd43ca2ae06650fb1877de621c70f06b19eff))
* **vscode:** rename marketplace icon file for cache bust ([f3e9a4a](https://github.com/forst-lang/forst/commit/f3e9a4a9d02199b55abfe944e2a5d22ef7967b68))

## [0.1.1](https://github.com/forst-lang/forst/compare/vscode-forst-v0.1.0...vscode-forst-v0.1.1) (2026-04-05)


### Features

* add extension, syntax highlighting, and LSP editor support ([#36](https://github.com/forst-lang/forst/issues/36)) ([049133e](https://github.com/forst-lang/forst/commit/049133e28a95a24749a22fd1bc03002924607489))
* **cli:** add @forst/cli package and wire sidecar to shared resolver ([#51](https://github.com/forst-lang/forst/issues/51)) ([e569101](https://github.com/forst-lang/forst/commit/e569101aaa34ef4a6be14c4541559a65e314a2f5))
* **fmt:** add pretty-printer, forst fmt, and LSP formatting ([cf9b098](https://github.com/forst-lang/forst/commit/cf9b098e921435a7e19b82acb1c1fbc225ad570b))
* **lsp:** navigate locals, folding, completion, and formatting ([#40](https://github.com/forst-lang/forst/issues/40)) ([b4c559c](https://github.com/forst-lang/forst/commit/b4c559cfaaac8d6f1d96a092b09e07aea16a6245))
* **lsp:** rename, format code actions, Go-style params, VSIX icon ([6343445](https://github.com/forst-lang/forst/commit/63434455a92f926a108b08b076dd0630bf637a47))
* **vscode-forst:** add Forst editor snippets and snippet JSON test ([e5eec47](https://github.com/forst-lang/forst/commit/e5eec47f29fccea9999e0b3aeef868074ab9a010))
* **vscode:** add extension marketplace icon ([801e9b4](https://github.com/forst-lang/forst/commit/801e9b4476ae1753ae0b93e4626704e81223d67f))


### Bug Fixes

* **lsp:** robust URIs and cache invalidation ([c0b3b65](https://github.com/forst-lang/forst/commit/c0b3b65c697797745f8c88d61587eb8e235d4207))
* **release:** sync VS Code extension version with release-please ([a5e2ff1](https://github.com/forst-lang/forst/commit/a5e2ff1eedfbd53e5b322a2fc1f1ae16ce34b2b8))
