# Changelog

## [0.5.1](https://github.com/forst-lang/forst/compare/runtime-v0.5.0...runtime-v0.5.1) (2026-09-04)


### Bug Fixes

* **ci:** skip npm publish when version already exists ([7b57e48](https://github.com/forst-lang/forst/commit/7b57e481d1c3c4787ed75350230a1af7c2d90c97))
* **ci:** upload only release files and clobber on re-run ([7b57e48](https://github.com/forst-lang/forst/commit/7b57e481d1c3c4787ed75350230a1af7c2d90c97))

## [0.5.0](https://github.com/forst-lang/forst/compare/runtime-v0.4.2...runtime-v0.5.0) (2026-09-03)


### ⚠ BREAKING CHANGES

* **interop:** node.loader is deprecated; use javascript.legacyModules.format. The default runtime mode is compiled JavaScript, not tsx-backed TypeScript source. Set javascript.legacyModules.format to typescript to restore the previous behavior.
* **imports:** Migrate in one step — no compatibility shims remain. 1. Replace `import "./path" node` with `import "./path" js`. 2. Collapse `ftconfig.node` and `ftconfig.javascript` into `bridge` (use    `bridge.legacyModules.format` instead of `node.loader`). 3. `npm install @forst/runtime` and remove `@forst/node-runtime`. 4. Rename deploy env vars `FORST_NODE_*` to `FORST_BRIDGE_*`. 5. Point bootstrap defaults to `node_modules/@forst/runtime/dist/bootstrap.js`. 6. Replace CLI flag `-require-no-node` with `-require-no-bridge`.

### Features

* **imports:** safe local names for node imports with LSP quickfixes ([608fb80](https://github.com/forst-lang/forst/commit/608fb8031d786e17eb85cdb262b63e185ca6ca6f))
* **interop:** support Bun and Deno as JavaScript bridge hosts ([#190](https://github.com/forst-lang/forst/issues/190)) ([608fb80](https://github.com/forst-lang/forst/commit/608fb8031d786e17eb85cdb262b63e185ca6ca6f))

## [0.4.2](https://github.com/forst-lang/forst/compare/runtime-v0.4.1...node-runtime-v0.4.2) (2026-08-16)


### Bug Fixes

* **invokeserver:** route invoke logs through logrus ([#181](https://github.com/forst-lang/forst/issues/181)) ([66f5c42](https://github.com/forst-lang/forst/commit/66f5c42accf4c2ed7e872a8d0133966f56a702e4))

## [0.4.1](https://github.com/forst-lang/forst/compare/runtime-v0.4.0...node-runtime-v0.4.1) (2026-07-20)


### Bug Fixes

* **nodert:** honor `HostReadyTimeout` for bootstrap ready wait ([#149](https://github.com/forst-lang/forst/issues/149)) ([120c4c8](https://github.com/forst-lang/forst/commit/120c4c8e2e470c8d6ec16e881b117ac26f15fda0))

## [0.4.0](https://github.com/forst-lang/forst/compare/runtime-v0.3.2...node-runtime-v0.4.0) (2026-07-20)


### ⚠ BREAKING CHANGES

* **nodert:** move bootstrap Node interop to socket RPC ([#147](https://github.com/forst-lang/forst/issues/147))

### Features

* **nodert:** move bootstrap Node interop to socket RPC ([#147](https://github.com/forst-lang/forst/issues/147)) ([5071ed6](https://github.com/forst-lang/forst/commit/5071ed603f8be1bd305f30195b0620bae654a9d6))

## [0.3.2](https://github.com/forst-lang/forst/compare/runtime-v0.3.1...node-runtime-v0.3.2) (2026-07-13)


### Features

* **dev:** add runtime profile for embedded invoke and host mode ([#142](https://github.com/forst-lang/forst/issues/142)) ([fcfc8c0](https://github.com/forst-lang/forst/commit/fcfc8c08fc72eaba997a7bebb6303542a2684781))

## [0.3.1](https://github.com/forst-lang/forst/compare/runtime-v0.3.0...node-runtime-v0.3.1) (2026-07-12)


### Features

* **lsp:** add go-to-definition for import node bindings ([#138](https://github.com/forst-lang/forst/issues/138)) ([fdcd35d](https://github.com/forst-lang/forst/commit/fdcd35d80147589f42bca8b9cc571065cb1b21cb))


### Bug Fixes

* **nodert:** prevent duplicate host binds, restore piped colors ([#141](https://github.com/forst-lang/forst/issues/141)) ([3ecb2ff](https://github.com/forst-lang/forst/commit/3ecb2ff505b3bd8fa60c8bd51292cd60285b999d))

## [0.3.0](https://github.com/forst-lang/forst/compare/runtime-v0.2.1...node-runtime-v0.3.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **node-runtime:** adopt Effect for RPC, logging, host runtime ([#136](https://github.com/forst-lang/forst/issues/136))

### Features

* **node-runtime:** adopt Effect for RPC, logging, host runtime ([#136](https://github.com/forst-lang/forst/issues/136)) ([75a4032](https://github.com/forst-lang/forst/commit/75a403238038225ffc753b5382c279f659438f0b))

## [0.2.1](https://github.com/forst-lang/forst/compare/runtime-v0.2.0...node-runtime-v0.2.1) (2026-07-11)


### Bug Fixes

* **node-runtime:** drop register.cjs from JSR publish ([d4da94f](https://github.com/forst-lang/forst/commit/d4da94f57d0418bccef2dcf78d39c05b7deb3b9c))

## [0.2.0](https://github.com/forst-lang/forst/compare/runtime-v0.1.0...node-runtime-v0.2.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **bridge-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130))

### Features

* **bridge-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130)) ([85bcdcc](https://github.com/forst-lang/forst/commit/85bcdcc81ed9551eecdd1f38e1bc15378e42ef8c))
* **node-runtime:** add npm and JSR publish pipeline ([1744924](https://github.com/forst-lang/forst/commit/1744924ef613deab88722a3adb97bac147ee3abe))

## [0.1.0](https://github.com/forst-lang/forst/releases/tag/runtime-v0.1.0) (2026-07-11)

### Features

* **node-runtime:** initial publish of bootstrap, host mode, indexer CLI, and RPC runtime
