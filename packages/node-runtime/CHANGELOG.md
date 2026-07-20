# Changelog

## [0.4.1](https://github.com/forst-lang/forst/compare/node-runtime-v0.4.0...node-runtime-v0.4.1) (2026-07-20)


### Bug Fixes

* **nodert:** honor `HostReadyTimeout` for bootstrap ready wait ([#149](https://github.com/forst-lang/forst/issues/149)) ([120c4c8](https://github.com/forst-lang/forst/commit/120c4c8e2e470c8d6ec16e881b117ac26f15fda0))

## [0.4.0](https://github.com/forst-lang/forst/compare/node-runtime-v0.3.2...node-runtime-v0.4.0) (2026-07-20)


### ⚠ BREAKING CHANGES

* **nodert:** move bootstrap Node interop to socket RPC ([#147](https://github.com/forst-lang/forst/issues/147))

### Features

* **nodert:** move bootstrap Node interop to socket RPC ([#147](https://github.com/forst-lang/forst/issues/147)) ([5071ed6](https://github.com/forst-lang/forst/commit/5071ed603f8be1bd305f30195b0620bae654a9d6))

## [0.3.2](https://github.com/forst-lang/forst/compare/node-runtime-v0.3.1...node-runtime-v0.3.2) (2026-07-13)


### Features

* **dev:** add runtime profile for embedded invoke and host mode ([#142](https://github.com/forst-lang/forst/issues/142)) ([fcfc8c0](https://github.com/forst-lang/forst/commit/fcfc8c08fc72eaba997a7bebb6303542a2684781))

## [0.3.1](https://github.com/forst-lang/forst/compare/node-runtime-v0.3.0...node-runtime-v0.3.1) (2026-07-12)


### Features

* **lsp:** add go-to-definition for import node bindings ([#138](https://github.com/forst-lang/forst/issues/138)) ([fdcd35d](https://github.com/forst-lang/forst/commit/fdcd35d80147589f42bca8b9cc571065cb1b21cb))


### Bug Fixes

* **nodert:** prevent duplicate host binds, restore piped colors ([#141](https://github.com/forst-lang/forst/issues/141)) ([3ecb2ff](https://github.com/forst-lang/forst/commit/3ecb2ff505b3bd8fa60c8bd51292cd60285b999d))

## [0.3.0](https://github.com/forst-lang/forst/compare/node-runtime-v0.2.1...node-runtime-v0.3.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **node-runtime:** adopt Effect for RPC, logging, host runtime ([#136](https://github.com/forst-lang/forst/issues/136))

### Features

* **node-runtime:** adopt Effect for RPC, logging, host runtime ([#136](https://github.com/forst-lang/forst/issues/136)) ([75a4032](https://github.com/forst-lang/forst/commit/75a403238038225ffc753b5382c279f659438f0b))

## [0.2.1](https://github.com/forst-lang/forst/compare/node-runtime-v0.2.0...node-runtime-v0.2.1) (2026-07-11)


### Bug Fixes

* **node-runtime:** drop register.cjs from JSR publish ([d4da94f](https://github.com/forst-lang/forst/commit/d4da94f57d0418bccef2dcf78d39c05b7deb3b9c))

## [0.2.0](https://github.com/forst-lang/forst/compare/node-runtime-v0.1.0...node-runtime-v0.2.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **node-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130))

### Features

* **node-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130)) ([85bcdcc](https://github.com/forst-lang/forst/commit/85bcdcc81ed9551eecdd1f38e1bc15378e42ef8c))
* **node-runtime:** add npm and JSR publish pipeline ([1744924](https://github.com/forst-lang/forst/commit/1744924ef613deab88722a3adb97bac147ee3abe))

## [0.1.0](https://github.com/forst-lang/forst/releases/tag/node-runtime-v0.1.0) (2026-07-11)

### Features

* **node-runtime:** initial publish of bootstrap, host mode, indexer CLI, and RPC runtime
