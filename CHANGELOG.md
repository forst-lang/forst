# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0](https://github.com/forst-lang/forst/compare/v0.8.3...v0.9.0) (2026-07-20)


### ⚠ BREAKING CHANGES

* **nodert:** move bootstrap Node interop to socket RPC ([#147](https://github.com/forst-lang/forst/issues/147))

### Features

* **nodert:** move bootstrap Node interop to socket RPC ([#147](https://github.com/forst-lang/forst/issues/147)) ([5071ed6](https://github.com/forst-lang/forst/commit/5071ed603f8be1bd305f30195b0620bae654a9d6))

## [0.8.3](https://github.com/forst-lang/forst/compare/v0.8.2...v0.8.3) (2026-07-18)


### Bug Fixes

* **compiler:** emit extra invoke packages on `forst build -o` ([#144](https://github.com/forst-lang/forst/issues/144)) ([0650212](https://github.com/forst-lang/forst/commit/0650212632fa1d9271e28023ccdaaa5cac6a47f5))

## [0.8.2](https://github.com/forst-lang/forst/compare/v0.8.1...v0.8.2) (2026-07-13)


### Features

* **dev:** add runtime profile for embedded invoke and host mode ([#142](https://github.com/forst-lang/forst/issues/142)) ([fcfc8c0](https://github.com/forst-lang/forst/commit/fcfc8c08fc72eaba997a7bebb6303542a2684781))

## [0.8.1](https://github.com/forst-lang/forst/compare/v0.8.0...v0.8.1) (2026-07-12)


### Features

* **lsp:** add go-to-definition for Go import bindings ([#140](https://github.com/forst-lang/forst/issues/140)) ([099418a](https://github.com/forst-lang/forst/commit/099418a7791adf41e395534a6c2da8a3395b8e29))
* **lsp:** add go-to-definition for import node bindings ([#138](https://github.com/forst-lang/forst/issues/138)) ([fdcd35d](https://github.com/forst-lang/forst/commit/fdcd35d80147589f42bca8b9cc571065cb1b21cb))


### Bug Fixes

* **nodert:** prevent duplicate host binds, restore piped colors ([#141](https://github.com/forst-lang/forst/issues/141)) ([3ecb2ff](https://github.com/forst-lang/forst/commit/3ecb2ff505b3bd8fa60c8bd51292cd60285b999d))

## [0.8.0](https://github.com/forst-lang/forst/compare/v0.7.0...v0.8.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **node-runtime:** adopt Effect for RPC, logging, host runtime ([#136](https://github.com/forst-lang/forst/issues/136))

### Features

* **node-runtime:** adopt Effect for RPC, logging, host runtime ([#136](https://github.com/forst-lang/forst/issues/136)) ([75a4032](https://github.com/forst-lang/forst/commit/75a403238038225ffc753b5382c279f659438f0b))
* **release:** ship amd64 and arm64 deb packages on compiler release ([7c77310](https://github.com/forst-lang/forst/commit/7c773102619cd14455f15981d3015ad2b4a6fe4e))

## [0.7.0](https://github.com/forst-lang/forst/compare/v0.6.1...v0.7.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **node-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130))
* add slice spread fields and sibling Go FFI ([#129](https://github.com/forst-lang/forst/issues/129))

### Features

* add slice spread fields and sibling Go FFI ([#129](https://github.com/forst-lang/forst/issues/129)) ([322b03f](https://github.com/forst-lang/forst/commit/322b03f4d2942e04c4770776de097ddbe5c5a86b))
* **node-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130)) ([85bcdcc](https://github.com/forst-lang/forst/commit/85bcdcc81ed9551eecdd1f38e1bc15378e42ef8c))
* **node-runtime:** add npm and JSR publish pipeline ([1744924](https://github.com/forst-lang/forst/commit/1744924ef613deab88722a3adb97bac147ee3abe))


### Bug Fixes

* **testrunner:** support Forst `Test*` functions with `testing.T` ([#127](https://github.com/forst-lang/forst/issues/127)) ([e43155d](https://github.com/forst-lang/forst/commit/e43155d0f3bdb47646eb2b7ac68d89c91d41e67b))

## [0.6.1](https://github.com/forst-lang/forst/compare/v0.6.0...v0.6.1) (2026-07-06)


### Features

* **transformer:** add ensure diagnostics in tests ([#123](https://github.com/forst-lang/forst/issues/123)) ([62803b6](https://github.com/forst-lang/forst/commit/62803b6094489b49a193b37d2c18dae6994db8b6))
* **transformer:** lower ensure failures in Test* functions to t.Fatalf ([152f60e](https://github.com/forst-lang/forst/commit/152f60e6ecd68e7568194866b95651b01768b6e3))


### Bug Fixes

* **parser:** support qualified names in pointer composite literals ([152f60e](https://github.com/forst-lang/forst/commit/152f60e6ecd68e7568194866b95651b01768b6e3))
* **printer:** preserve typed slice literals in forst fmt ([9e047f9](https://github.com/forst-lang/forst/commit/9e047f92be14b1b1c8a576051da4240e5a161767))
* **transformer:** correct pointer composite and shape zero-value emission ([152f60e](https://github.com/forst-lang/forst/commit/152f60e6ecd68e7568194866b95651b01768b6e3))
* **typechecker:** improve empty slice inference and append widening ([152f60e](https://github.com/forst-lang/forst/commit/152f60e6ecd68e7568194866b95651b01768b6e3))


### Performance Improvements

* **typechecker:** add scope-key hashing, compat memo ([#125](https://github.com/forst-lang/forst/issues/125)) ([872405c](https://github.com/forst-lang/forst/commit/872405c2605636302c64516dbf73b6b5d2ff90f1))
* **typechecker:** cache scope hashes, memoize HashNode ([#124](https://github.com/forst-lang/forst/issues/124)) ([923d6c9](https://github.com/forst-lang/forst/commit/923d6c9070351b7c9e68f3a0f77d53ef9ff7a49e))

## [0.6.0](https://github.com/forst-lang/forst/compare/v0.5.9...v0.6.0) (2026-07-06)


### ⚠ BREAKING CHANGES

* Manual AST builders and walkers must use []ExpressionNode for array literal elements instead of []LiteralNode.

### Features

* **lsp:** support new syntax and module-wide analysis in the editor ([e21974b](https://github.com/forst-lang/forst/commit/e21974b3a90dd7d0e4b26cf03279969c9509cf29))
* **parser:** add postfix calls, package vars, typed literals, and Int conversions ([e21974b](https://github.com/forst-lang/forst/commit/e21974b3a90dd7d0e4b26cf03279969c9509cf29))
* **testrunner:** emit test-only Go when z_forst_gen.go exists ([1b2dd3e](https://github.com/forst-lang/forst/commit/1b2dd3e3467daf91644eaf49fa46a3e82b46a2bb))
* **typechecker:** split collect/infer and resolve sibling Forst package types ([e21974b](https://github.com/forst-lang/forst/commit/e21974b3a90dd7d0e4b26cf03279969c9509cf29))
* **typechecker:** typecheck unqualified Go calls in mixed .go+.ft packages ([65f1b48](https://github.com/forst-lang/forst/commit/65f1b48ceae2e24bd9cbd002cf6c2bee227c6ca9))


### Bug Fixes

* **compiler:** exclude *_test.ft from -root package merge ([1b2dd3e](https://github.com/forst-lang/forst/commit/1b2dd3e3467daf91644eaf49fa46a3e82b46a2bb))
* infer nested inline shapes and emit typed array literals ([03b9eaf](https://github.com/forst-lang/forst/commit/03b9eafc25cb4971b8a42cb78f71e0bf52e13fce))
* **parser:** disambiguate qualified params from shape-guard assertions ([61701f3](https://github.com/forst-lang/forst/commit/61701f3736e70f1872545006a854fe0a5c996e05))
* **transformer:** lower string(int) to strconv.Itoa ([9ba951f](https://github.com/forst-lang/forst/commit/9ba951f2d4c8d56b999af6c5913cdb72f9213fc3))
* **transformer:** omit json tags on unexported struct fields ([1439a70](https://github.com/forst-lang/forst/commit/1439a7086545d1bcac81915b7580cf8d6304f1c8))
* **transformer:** resolve sibling package types in struct and array literals ([fa8c72f](https://github.com/forst-lang/forst/commit/fa8c72f36c924f1bce957e78acfaa8091b6a9e3e))
* **typechecker:** improve Go interop for tests and builtins ([61701f3](https://github.com/forst-lang/forst/commit/61701f3736e70f1872545006a854fe0a5c996e05))
* **typechecker:** resolve Go imported package selectors ([e7846f4](https://github.com/forst-lang/forst/commit/e7846f4c856d8fcb62b41f93e4f7f26000c40667))
* **typechecker:** resolve sibling imports across Forst packages ([61701f3](https://github.com/forst-lang/forst/commit/61701f3736e70f1872545006a854fe0a5c996e05))

## [0.5.9](https://github.com/forst-lang/forst/compare/v0.5.8...v0.5.9) (2026-07-05)


### Bug Fixes

* wire ensure-or fallback and raise Go coverage ([#118](https://github.com/forst-lang/forst/issues/118)) ([228ff54](https://github.com/forst-lang/forst/commit/228ff54073052688173981e2bd2438be0fe39dc7))

## [0.5.8](https://github.com/forst-lang/forst/compare/v0.5.7...v0.5.8) (2026-06-28)


### Features

* **compiler:** improve constraints, guards, and codegen ([#116](https://github.com/forst-lang/forst/issues/116)) ([0d0da7b](https://github.com/forst-lang/forst/commit/0d0da7b16eb5e434ab431c43d5721288a66921bc))

## [0.5.7](https://github.com/forst-lang/forst/compare/v0.5.6...v0.5.7) (2026-06-28)


### Bug Fixes

* **vscode-forst:** align VS Code engine and types to 1.74 ([589bca5](https://github.com/forst-lang/forst/commit/589bca50d82444c423aa464385d612b3d8ec18a3))
* **vscode-forst:** bump engines.vscode to match @types/vscode ([d8700ba](https://github.com/forst-lang/forst/commit/d8700bae5a194663faeb6f9e929191288ab3a2fb))

## [0.5.6](https://github.com/forst-lang/forst/compare/v0.5.5...v0.5.6) (2026-06-28)


### Features

* **vscode-forst:** publish extension to VS Code Marketplace ([#113](https://github.com/forst-lang/forst/issues/113)) ([06abe0e](https://github.com/forst-lang/forst/commit/06abe0e5b9c1750766e6a1fccfe933293787250b))

## [0.5.5](https://github.com/forst-lang/forst/compare/v0.5.4...v0.5.5) (2026-06-28)


### Features

* **ci:** publish multi-arch Forst compiler images on GHCR ([#112](https://github.com/forst-lang/forst/issues/112)) ([6010d86](https://github.com/forst-lang/forst/commit/6010d869e03269a91603f610940b72b89ab2d670))
* **lsp:** show hover for Go predeclared builtins ([#110](https://github.com/forst-lang/forst/issues/110)) ([555e4b3](https://github.com/forst-lang/forst/commit/555e4b36bc8cc78d0388936c944324e2a6f04bbd))

## [0.5.4](https://github.com/forst-lang/forst/compare/v0.5.3...v0.5.4) (2026-06-28)


### Features

* **compiler:** add compound assignment and standalone inc/dec ([#108](https://github.com/forst-lang/forst/issues/108)) ([1c0dc46](https://github.com/forst-lang/forst/commit/1c0dc465c883b7fb98933fabc52ab0a40abcfdbf))

## [0.5.3](https://github.com/forst-lang/forst/compare/v0.5.2...v0.5.3) (2026-06-28)


### Bug Fixes

* **typechecker:** complete trailing implicit return and provider method codegen ([d94a1de](https://github.com/forst-lang/forst/commit/d94a1de52839341d6d40b4b9eee86e5968807988))

## [0.5.2](https://github.com/forst-lang/forst/compare/v0.5.1...v0.5.2) (2026-06-28)


### Features

* **security:** cap HTTP bodies and confine FS reads with os.Root ([#101](https://github.com/forst-lang/forst/issues/101)) ([055a2ee](https://github.com/forst-lang/forst/commit/055a2ee5c061356aeb49bf4c455263a908d294c4))


### Performance Improvements

* **release:** add PGO profile and use it in `build:release` ([#104](https://github.com/forst-lang/forst/issues/104)) ([985f36d](https://github.com/forst-lang/forst/commit/985f36d4f4084428df4766b77863dcc2f322a7da))

## [0.5.1](https://github.com/forst-lang/forst/compare/v0.5.0...v0.5.1) (2026-06-28)


### Bug Fixes

* **cli:** pin compiler release and verify digests when available ([f43e22a](https://github.com/forst-lang/forst/commit/f43e22a38737dbaa27e93e823ced36367b64b874))
* **vscode-forst:** fail closed when compiler download is enabled ([f43e22a](https://github.com/forst-lang/forst/commit/f43e22a38737dbaa27e93e823ced36367b64b874))

## [0.5.0](https://github.com/forst-lang/forst/compare/v0.4.0...v0.5.0) (2026-06-28)


### ⚠ BREAKING CHANGES

* Providers (`use` / `with`) ([#97](https://github.com/forst-lang/forst/issues/97))

### Features

* Providers (`use` / `with`) ([#97](https://github.com/forst-lang/forst/issues/97)) ([82a1a97](https://github.com/forst-lang/forst/commit/82a1a971f80736e250534f9eb2663838f77e0b0d))


### Bug Fixes

* **docs:** register Forst grammar for Mintlify prod highlighting ([51d4dfc](https://github.com/forst-lang/forst/commit/51d4dfc3ed1bfb12d3d7af61d908b9f7f773ddc8))
* **lsp:** isolate providers analyze test in temp module ([79aa1e0](https://github.com/forst-lang/forst/commit/79aa1e01e7895d48258b3dd72821b20d6333594d))

## [0.4.0](https://github.com/forst-lang/forst/compare/v0.3.0...v0.4.0) (2026-04-21)


### ⚠ BREAKING CHANGES

* map reads are Result(V, Error); emit comma-ok via Go IIFE ([#94](https://github.com/forst-lang/forst/issues/94))

### Features

* map reads are Result(V, Error); emit comma-ok via Go IIFE ([#94](https://github.com/forst-lang/forst/issues/94)) ([0cebdf1](https://github.com/forst-lang/forst/commit/0cebdf15dc86c74d62eaaa4eaedd50aca5fc83a1))

## [0.3.0](https://github.com/forst-lang/forst/compare/v0.2.17...v0.3.0) (2026-04-20)


### ⚠ BREAKING CHANGES

* **sidecar:** ForstSidecarClient.invokeStreaming and invokeStreamingIterable are removed; use invokeStream(package, fn, args) for async iteration or invokeStream(package, fn, args, onResult) for callback consumption.

### Features

* **sidecar:** unify streaming as invokeStream with generator or callback ([fe9d8df](https://github.com/forst-lang/forst/commit/fe9d8df1fb444034b5a4d0838c997c91c21a4a26))

## [0.2.17](https://github.com/forst-lang/forst/compare/v0.2.16...v0.2.17) (2026-04-19)


### Features

* **generate:** opt-in streaming TS clients, chan T IR, and sidecar async iterable ([8dcc68c](https://github.com/forst-lang/forst/commit/8dcc68cbe19ff9ea5db3039da08fac507fe0843e))

## [0.2.16](https://github.com/forst-lang/forst/compare/v0.2.15...v0.2.16) (2026-04-19)


### Features

* **typechecker:** resolve Go-bound fields and hovers via go/types ([b12e3e1](https://github.com/forst-lang/forst/commit/b12e3e19819a4655a188c2435514caed2dbad85c))

## [0.2.15](https://github.com/forst-lang/forst/compare/v0.2.14...v0.2.15) (2026-04-19)


### Features

* **cmd/forst:** add test hooks and return errors from dump instead of exiting ([68ec01a](https://github.com/forst-lang/forst/commit/68ec01a4ef8bd4aae8f501f05140c2ee7b752796))
* **typechecker:** track Go call results and check methods with go/types ([68ec01a](https://github.com/forst-lang/forst/commit/68ec01a4ef8bd4aae8f501f05140c2ee7b752796))


### Bug Fixes

* **compiler:** parse run/build argv without touching os.Args ([66460c1](https://github.com/forst-lang/forst/commit/66460c190a1552cd3a60b98d3a4f369916f19275))

## [0.2.14](https://github.com/forst-lang/forst/compare/v0.2.13...v0.2.14) (2026-04-19)


### Bug Fixes

* **vscode-forst:** install CLI prod deps when materializing for VSIX ([e90aaba](https://github.com/forst-lang/forst/commit/e90aabafe3923aa7e7976bae993025a0ee94c2cb))

## [0.2.13](https://github.com/forst-lang/forst/compare/v0.2.12...v0.2.13) (2026-04-19)


### Features

* **cli:** add preferLatestRelease for GitHub latest vs bundled semver ([4e2eff9](https://github.com/forst-lang/forst/commit/4e2eff97cb0fa3626325145bd0feb4b360237fe3))

## [0.2.12](https://github.com/forst-lang/forst/compare/v0.2.11...v0.2.12) (2026-04-19)


### Bug Fixes

* **typechecker:** lazy-load Go imports and improve opaque FFI typing ([c0bd7d2](https://github.com/forst-lang/forst/commit/c0bd7d28c99f2ddd02ebaad9fc318df3fbbe9927))
* **typechecker:** unify `nil == nil` to Bool ([6e98448](https://github.com/forst-lang/forst/commit/6e98448c7f99b38bf8a0f43b53f1f246b17e3ec1))

## [0.2.11](https://github.com/forst-lang/forst/compare/v0.2.10...v0.2.11) (2026-04-19)


### Bug Fixes

* **release:** Clear token based auth as much as possible ([6240d2f](https://github.com/forst-lang/forst/commit/6240d2f175f86a7f1a647cfdd4b01005f973f26c))

## [0.2.10](https://github.com/forst-lang/forst/compare/v0.2.9...v0.2.10) (2026-04-19)


### Bug Fixes

* **release:** unset NODE_AUTH_TOKEN for release ([006984d](https://github.com/forst-lang/forst/commit/006984dc747f353966eaa9f7e0523a596cd4c4e6))

## [0.2.9](https://github.com/forst-lang/forst/compare/v0.2.8...v0.2.9) (2026-04-19)


### Bug Fixes

* **release:** update GitHub action and update npm to latest ([2139b3c](https://github.com/forst-lang/forst/commit/2139b3c63075de44fc0a2815eb3821a7eea74467))

## [0.2.8](https://github.com/forst-lang/forst/compare/v0.2.7...v0.2.8) (2026-04-19)


### Bug Fixes

* **release:** add --provenance to npm publish commands ([6e9beac](https://github.com/forst-lang/forst/commit/6e9beac63d5aa6d0dd541fb655ed4a0333fd58bd))

## [0.2.7](https://github.com/forst-lang/forst/compare/v0.2.6...v0.2.7) (2026-04-19)


### Bug Fixes

* **release:** Further adjust release jobs for NPM ([8f9466f](https://github.com/forst-lang/forst/commit/8f9466fe82a44f09112e02e8c2fd58ee19a78d35))

## [0.2.6](https://github.com/forst-lang/forst/compare/v0.2.5...v0.2.6) (2026-04-19)


### Bug Fixes

* **release:** Restore permissions for NPM release ([3c2650b](https://github.com/forst-lang/forst/commit/3c2650b20d10052cfddca9383c697511b1f87da4))

## [0.2.5](https://github.com/forst-lang/forst/compare/v0.2.4...v0.2.5) (2026-04-19)


### Bug Fixes

* **release:** align npm auth with GitHub Node.js publish guide ([ac613b3](https://github.com/forst-lang/forst/commit/ac613b3683dc95cd8bd7bacb2ff493397e3e52fd))

## [0.2.4](https://github.com/forst-lang/forst/compare/v0.2.3...v0.2.4) (2026-04-19)


### Bug Fixes

* **cli:** document Node 18+ in README ([67e7abc](https://github.com/forst-lang/forst/commit/67e7abc1d9db416a11f9d0a5b1429a0c2ad88b6f))
* **release:** npm whoami and --workspaces=false ([67e7abc](https://github.com/forst-lang/forst/commit/67e7abc1d9db416a11f9d0a5b1429a0c2ad88b6f))

## [0.2.3](https://github.com/forst-lang/forst/compare/v0.2.2...v0.2.3) (2026-04-18)


### Bug Fixes

* **release:** run sidecar publish when CLI job is skipped ([594962c](https://github.com/forst-lang/forst/commit/594962c5a15724ba28ffbf55ede92bfe8b259f20))

## [0.2.2](https://github.com/forst-lang/forst/compare/v0.2.1...v0.2.2) (2026-04-18)


### Features

* support Go dot-imports across parser, checker, and LSP ([496c40f](https://github.com/forst-lang/forst/commit/496c40f5d2118295f85c155f2446d4d1b672069b))


### Bug Fixes

* **sidecar:** keep registry @forst/cli off workspace:* ([31763bf](https://github.com/forst-lang/forst/commit/31763bf55e1a26049aa9c3b5c051ba0e63cdc822))

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
