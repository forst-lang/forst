# Changelog

## [0.0.30](https://github.com/forst-lang/forst/compare/cli-v0.0.29...cli-v0.0.30) (2026-04-19)


### Bug Fixes

* **release:** add --provenance to npm publish commands ([6e9beac](https://github.com/forst-lang/forst/commit/6e9beac63d5aa6d0dd541fb655ed4a0333fd58bd))

## [0.0.29](https://github.com/forst-lang/forst/compare/cli-v0.0.28...cli-v0.0.29) (2026-04-19)


### Bug Fixes

* **release:** Further adjust release jobs for NPM ([8f9466f](https://github.com/forst-lang/forst/commit/8f9466fe82a44f09112e02e8c2fd58ee19a78d35))

## [0.0.28](https://github.com/forst-lang/forst/compare/cli-v0.0.27...cli-v0.0.28) (2026-04-19)


### Bug Fixes

* **release:** Restore permissions for NPM release ([3c2650b](https://github.com/forst-lang/forst/commit/3c2650b20d10052cfddca9383c697511b1f87da4))

## [0.0.27](https://github.com/forst-lang/forst/compare/cli-v0.0.26...cli-v0.0.27) (2026-04-19)


### Bug Fixes

* **release:** align npm auth with GitHub Node.js publish guide ([ac613b3](https://github.com/forst-lang/forst/commit/ac613b3683dc95cd8bd7bacb2ff493397e3e52fd))

## [0.0.26](https://github.com/forst-lang/forst/compare/cli-v0.0.25...cli-v0.0.26) (2026-04-19)


### Bug Fixes

* **cli:** document Node 18+ in README ([67e7abc](https://github.com/forst-lang/forst/commit/67e7abc1d9db416a11f9d0a5b1429a0c2ad88b6f))
* **release:** npm whoami and --workspaces=false ([67e7abc](https://github.com/forst-lang/forst/commit/67e7abc1d9db416a11f9d0a5b1429a0c2ad88b6f))

## [0.0.25](https://github.com/forst-lang/forst/compare/cli-v0.0.24...cli-v0.0.25) (2026-04-18)


### Bug Fixes

* **release:** run sidecar publish when CLI job is skipped ([594962c](https://github.com/forst-lang/forst/commit/594962c5a15724ba28ffbf55ede92bfe8b259f20))

## [0.0.24](https://github.com/forst-lang/forst/compare/cli-v0.0.23...cli-v0.0.24) (2026-04-06)


### Features

* **cli:** add --forst-go-buildinfo using go version -m ([f3b46a6](https://github.com/forst-lang/forst/commit/f3b46a616c730a61ba6f63c8fa86d3560db24bf4))

## [0.0.23](https://github.com/forst-lang/forst/compare/cli-v0.0.22...cli-v0.0.23) (2026-04-06)


### Features

* **cli:** add --version/-V to npm shim ([490c8e5](https://github.com/forst-lang/forst/commit/490c8e5211b83cb3c7a15b2c3218ff5ace597b0a))
* **sidecar:** add invalidateFunctionCache on ForstSidecarClient ([490c8e5](https://github.com/forst-lang/forst/commit/490c8e5211b83cb3c7a15b2c3218ff5ace597b0a))
* **vscode-forst:** add Copy debug info to clipboard command ([490c8e5](https://github.com/forst-lang/forst/commit/490c8e5211b83cb3c7a15b2c3218ff5ace597b0a))

## [0.0.22](https://github.com/forst-lang/forst/compare/cli-v0.0.21...cli-v0.0.22) (2026-04-06)


### Features

* **cli:** add --forst-cli-info diagnostics ([ee648e5](https://github.com/forst-lang/forst/commit/ee648e5d6c4d8f71e6547c7f51ce54e696aec215))

## [0.0.21](https://github.com/forst-lang/forst/compare/cli-v0.0.20...cli-v0.0.21) (2026-04-06)


### Features

* **cli:** gate binary download and expose release helpers ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **sidecar:** optional compiler download via config and env ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))
* **vscode-forst:** resolve `forst` via @forst/cli and `forst.downloadCompiler` ([81b819f](https://github.com/forst-lang/forst/commit/81b819fa4e4fe494e08d086a789242439f2426c4))

## [0.0.20](https://github.com/forst-lang/forst/compare/cli-v0.0.19...cli-v0.0.20) (2026-04-05)


### Features

* **cli:** add @forst/cli package and wire sidecar to shared resolver ([#51](https://github.com/forst-lang/forst/issues/51)) ([e569101](https://github.com/forst-lang/forst/commit/e569101aaa34ef4a6be14c4541559a65e314a2f5))
