# Changelog

## [0.8.0](https://github.com/forst-lang/forst/compare/cli-v0.7.0...cli-v0.8.0) (2026-08-17)


### ⚠ BREAKING CHANGES

* `forst build -o` no longer emits Go sources or a fixed `bin/forst-invoke` name. Use `forst generate` for Go output and exec the path in `manifest.binary` with `FORST_BOUNDARY_ROOT` set.
* **examples:** FORST_INVOKE_ONLY was removed. Set FORST_SKIP_NODE_HOST=1 (or true) when running a hostMode built binary with Node external. manifest.json invokeOnlyDefault is now skipNodeHostDefault.

### Features

* add native program build with hostMode runtime ([#185](https://github.com/forst-lang/forst/issues/185)) ([5e5717d](https://github.com/forst-lang/forst/commit/5e5717d1b53ce281348e50afa7c3d9d6bdb5eba8))


### Bug Fixes

* **compiler:** remove legacy companions and tighten generate paths ([5e5717d](https://github.com/forst-lang/forst/commit/5e5717d1b53ce281348e50afa7c3d9d6bdb5eba8))
* **examples:** refresh goldens with forst generate after build output change ([5e5717d](https://github.com/forst-lang/forst/commit/5e5717d1b53ce281348e50afa7c3d9d6bdb5eba8))
* **invokeserver:** synchronize embedded server lifecycle ([5e5717d](https://github.com/forst-lang/forst/commit/5e5717d1b53ce281348e50afa7c3d9d6bdb5eba8))
* **nodert:** respect invoke-only for hostMode and auth handoff ([5e5717d](https://github.com/forst-lang/forst/commit/5e5717d1b53ce281348e50afa7c3d9d6bdb5eba8))

## [0.7.0](https://github.com/forst-lang/forst/compare/cli-v0.6.1...cli-v0.7.0) (2026-08-16)


### ⚠ BREAKING CHANGES

* **release:** @forst/cli will follow the Go compiler semver on the next linked release. The first publish after this change may jump from the old CLI line (for example 0.6.x) to the current compiler line.

### Continuous Integration

* **release:** couple @forst/cli with compiler via linked-versions ([e6d0349](https://github.com/forst-lang/forst/commit/e6d0349ba2423fc42d9d3e3bc887e3ca23281dcc))

## [0.6.1](https://github.com/forst-lang/forst/compare/cli-v0.6.0...cli-v0.6.1) (2026-08-16)


### Bug Fixes

* **cli:** improve pin diagnostics and download errors ([335b3c8](https://github.com/forst-lang/forst/commit/335b3c8454575852061911355a992cd8d4a0edf1))

## [0.6.0](https://github.com/forst-lang/forst/compare/cli-v0.5.0...cli-v0.6.0) (2026-08-14)


### ⚠ BREAKING CHANGES

* **generate:** scope domain error tags to Forst packages ([#176](https://github.com/forst-lang/forst/issues/176))

### Features

* **generate:** scope domain error tags to Forst packages ([#176](https://github.com/forst-lang/forst/issues/176)) ([f053420](https://github.com/forst-lang/forst/commit/f053420bcfe8810a71ac55f037908230ed0c009e))

## [0.5.0](https://github.com/forst-lang/forst/compare/cli-v0.4.1...cli-v0.5.0) (2026-08-11)


### ⚠ BREAKING CHANGES

* **cli:** @forst/cli/invoke no longer exports prepareConnectInvokeEnv, resolveHostInvokeAuthHandoff, startHostInvokeAuthRecvListener, or envInvokeAuthRecvFd. Use prepareInvokeConnect and getInvokeAuthHandoff instead.

### Features

* **cli:** shrink host invoke connect API ([#171](https://github.com/forst-lang/forst/issues/171)) ([7e6fdc7](https://github.com/forst-lang/forst/commit/7e6fdc7ccd31329ae91cecd8405e75b1d9ae68b3))


### Bug Fixes

* **cli:** add explicit type for DEFAULT_EMBEDDED_INVOKE_BASE_URL ([7e6fdc7](https://github.com/forst-lang/forst/commit/7e6fdc7ccd31329ae91cecd8405e75b1d9ae68b3))
* **generate:** skip throw in emitted invoke auth handoff parsing ([7e6fdc7](https://github.com/forst-lang/forst/commit/7e6fdc7ccd31329ae91cecd8405e75b1d9ae68b3))

## [0.4.1](https://github.com/forst-lang/forst/compare/cli-v0.4.0...cli-v0.4.1) (2026-08-11)


### Features

* **generate:** prefer host auth handoff in emitted transport ([85c08aa](https://github.com/forst-lang/forst/commit/85c08aa7f2bf82946916085d026057cd1424e855))
* **invoke:** relay host-mode auth over inherited fds ([#168](https://github.com/forst-lang/forst/issues/168)) ([85c08aa](https://github.com/forst-lang/forst/commit/85c08aa7f2bf82946916085d026057cd1424e855))


### Bug Fixes

* **cli:** safe host auth handoff parse and early recv listen ([85c08aa](https://github.com/forst-lang/forst/commit/85c08aa7f2bf82946916085d026057cd1424e855))

## [0.4.0](https://github.com/forst-lang/forst/compare/cli-v0.3.0...cli-v0.4.0) (2026-08-11)


### ⚠ BREAKING CHANGES

* **invokeserver:** harden local invoke with UDS and HMAC auth ([#166](https://github.com/forst-lang/forst/issues/166))

### Features

* **invokeserver:** harden local invoke with UDS and HMAC auth ([#166](https://github.com/forst-lang/forst/issues/166)) ([cf593cb](https://github.com/forst-lang/forst/commit/cf593cb82f6aac4ee6fccd0ed0562cc5176c6fc0))

## [0.3.0](https://github.com/forst-lang/forst/compare/cli-v0.2.0...cli-v0.3.0) (2026-08-09)


### ⚠ BREAKING CHANGES

* **generate:** emit typed domain errors for nominal failures ([#160](https://github.com/forst-lang/forst/issues/160))

### Features

* **generate:** emit typed domain errors for nominal failures ([#160](https://github.com/forst-lang/forst/issues/160)) ([ba0f837](https://github.com/forst-lang/forst/commit/ba0f8377d67b4d91a35e796925af8b84d5de20db))

## [0.2.0](https://github.com/forst-lang/forst/compare/cli-v0.1.2...cli-v0.2.0) (2026-08-09)


### ⚠ BREAKING CHANGES

* **generate:** emit self-contained @forst/gen TS client package ([#158](https://github.com/forst-lang/forst/issues/158))

### Features

* **generate:** emit self-contained @forst/gen TS client package ([#158](https://github.com/forst-lang/forst/issues/158)) ([f89bf85](https://github.com/forst-lang/forst/commit/f89bf8574b0aeaa80ba8c8f7c6c1c04700c04b2c))

## [Unreleased]

### Features

* **cli:** Add `@forst/cli/invoke` with `startForstInvokeServer` for Node→Forst HTTP invoke lifecycle (attach, spawn `dev`/`embedded`, `/health` readiness, SIGTERM then SIGKILL). Orthogonal to `@forst/node-runtime`.
* **cli:** Declare `@forst/errors` as a runtime dependency so apps with `@forst/cli` can import shared invoke failure classes without adding Effect.
* **cli:** Host-mode invoke auth handoff via `FORST_INVOKE_AUTH_RECV_FD` (`prepareInvokeConnect` / `getInvokeAuthHandoff`), plus shared `DEFAULT_EMBEDDED_INVOKE_PORT` and `DEFAULT_EMBEDDED_INVOKE_BASE_URL`.

### Bug Fixes

* **cli:** Parse host auth handoff JSON as `unknown` and reject non-object roots (including `null`) instead of throwing a TypeError.

## [0.1.2](https://github.com/forst-lang/forst/compare/cli-v0.1.1...cli-v0.1.2) (2026-07-20)


### Bug Fixes

* **nodert:** honor `HostReadyTimeout` for bootstrap ready wait ([#149](https://github.com/forst-lang/forst/issues/149)) ([120c4c8](https://github.com/forst-lang/forst/commit/120c4c8e2e470c8d6ec16e881b117ac26f15fda0))

## [0.1.1](https://github.com/forst-lang/forst/compare/cli-v0.1.0...cli-v0.1.1) (2026-07-13)


### Features

* **dev:** add runtime profile for embedded invoke and host mode ([#142](https://github.com/forst-lang/forst/issues/142)) ([fcfc8c0](https://github.com/forst-lang/forst/commit/fcfc8c08fc72eaba997a7bebb6303542a2684781))

## [0.1.0](https://github.com/forst-lang/forst/compare/cli-v0.0.37...cli-v0.1.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* **node-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130))

### Features

* **node-interop:** import node, nodert runtime, embedded invoke ([#130](https://github.com/forst-lang/forst/issues/130)) ([85bcdcc](https://github.com/forst-lang/forst/commit/85bcdcc81ed9551eecdd1f38e1bc15378e42ef8c))

## [0.0.37](https://github.com/forst-lang/forst/compare/cli-v0.0.36...cli-v0.0.37) (2026-06-28)


### Bug Fixes

* **cli:** fallback to prior release when latest binary 404s ([d072230](https://github.com/forst-lang/forst/commit/d0722305db1e8117ddb6c598e309ab5761270f04))

## [0.0.36](https://github.com/forst-lang/forst/compare/cli-v0.0.35...cli-v0.0.36) (2026-06-28)


### Bug Fixes

* **cli:** pin compiler release and verify digests when available ([f43e22a](https://github.com/forst-lang/forst/commit/f43e22a38737dbaa27e93e823ced36367b64b874))
* **vscode-forst:** fail closed when compiler download is enabled ([f43e22a](https://github.com/forst-lang/forst/commit/f43e22a38737dbaa27e93e823ced36367b64b874))

## [0.0.35](https://github.com/forst-lang/forst/compare/cli-v0.0.34...cli-v0.0.35) (2026-04-19)


### Features

* **cli:** add preferLatestRelease for GitHub latest vs bundled semver ([4e2eff9](https://github.com/forst-lang/forst/commit/4e2eff97cb0fa3626325145bd0feb4b360237fe3))

## [0.0.34](https://github.com/forst-lang/forst/compare/cli-v0.0.33...cli-v0.0.34) (2026-04-19)


### Bug Fixes

* **release:** Clear token based auth as much as possible ([6240d2f](https://github.com/forst-lang/forst/commit/6240d2f175f86a7f1a647cfdd4b01005f973f26c))

## [0.0.33](https://github.com/forst-lang/forst/compare/cli-v0.0.32...cli-v0.0.33) (2026-04-19)


### Bug Fixes

* **release:** Update repository URLs to not use git+ prefix ([2bfca0f](https://github.com/forst-lang/forst/commit/2bfca0f7b25c37d629bd892f49533b82706f6913))

## [0.0.32](https://github.com/forst-lang/forst/compare/cli-v0.0.31...cli-v0.0.32) (2026-04-19)


### Bug Fixes

* **release:** unset NODE_AUTH_TOKEN for release ([006984d](https://github.com/forst-lang/forst/commit/006984dc747f353966eaa9f7e0523a596cd4c4e6))

## [0.0.31](https://github.com/forst-lang/forst/compare/cli-v0.0.30...cli-v0.0.31) (2026-04-19)


### Bug Fixes

* **release:** update GitHub action and update npm to latest ([2139b3c](https://github.com/forst-lang/forst/commit/2139b3c63075de44fc0a2815eb3821a7eea74467))

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
