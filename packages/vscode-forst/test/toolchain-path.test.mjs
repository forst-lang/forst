import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const {
  envWithToolchainPath,
  pathEnvKey,
  resolveGoPathDir,
  toolchainPathPrefixes,
} = require("../out/toolchainPath.js");

test("toolchainPathPrefixes (darwin) includes Homebrew and user Go bin", () => {
  const prefixes = toolchainPathPrefixes({
    home: "/tmp/home",
    platform: "darwin",
    env: {},
  });
  assert.ok(prefixes.includes("/opt/homebrew/bin"));
  assert.ok(prefixes.includes(path.join("/tmp/home", "go", "bin")));
  assert.ok(prefixes.includes("/usr/local/go/bin"));
});

test("toolchainPathPrefixes (linux) includes distro and snap bins", () => {
  const prefixes = toolchainPathPrefixes({
    home: "/home/dev",
    platform: "linux",
    env: {},
  });
  assert.ok(prefixes.includes("/usr/lib/go/bin"));
  assert.ok(prefixes.includes("/snap/bin"));
  assert.ok(prefixes.includes(path.join("/home/dev", ".asdf", "shims")));
  assert.ok(
    prefixes.includes(path.join("/home/dev", ".local", "share", "mise", "shims"))
  );
});

test("toolchainPathPrefixes (win32) includes Program Files Go and Scoop", () => {
  const prefixes = toolchainPathPrefixes({
    home: "C:\\Users\\dev",
    platform: "win32",
    env: {
      ProgramFiles: "C:\\Program Files",
      "ProgramFiles(x86)": "C:\\Program Files (x86)",
      LOCALAPPDATA: "C:\\Users\\dev\\AppData\\Local",
      ChocolateyInstall: "C:\\ProgramData\\chocolatey",
    },
  });
  assert.ok(prefixes.includes("C:\\Program Files\\Go\\bin"));
  assert.ok(prefixes.includes("C:\\Users\\dev\\scoop\\shims"));
  assert.ok(prefixes.includes("C:\\ProgramData\\chocolatey\\bin"));
  assert.ok(prefixes.includes("C:\\Users\\dev\\go\\bin"));
});

test("toolchainPathPrefixes honors GOROOT and GOPATH from env", () => {
  const prefixes = toolchainPathPrefixes({
    home: "/tmp/home",
    platform: "linux",
    env: {
      GOROOT: "/opt/go",
      GOPATH: "/data/gopath:/other/gopath",
    },
  });
  assert.ok(prefixes.includes("/opt/go/bin"));
  assert.ok(prefixes.includes("/data/gopath/bin"));
  assert.ok(prefixes.includes("/other/gopath/bin"));
});

test("toolchainPathPrefixes includes ~/sdk/go* bins when present", () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "forst-sdk-"));
  fs.mkdirSync(path.join(home, "sdk", "go1.22.0", "bin"), { recursive: true });
  const prefixes = toolchainPathPrefixes({
    home,
    platform: "linux",
    env: {},
  });
  assert.ok(prefixes.includes(path.join(home, "sdk", "go1.22.0", "bin")));
});

test("pathEnvKey prefers existing Path casing on Windows-style env", () => {
  assert.equal(pathEnvKey({ Path: "C:\\Windows" }), "Path");
  assert.equal(pathEnvKey({ PATH: "/usr/bin" }), "PATH");
});

test("envWithToolchainPath prepends existing toolchain dirs missing from PATH", () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "forst-path-"));
  const goBin = path.join(home, "go", "bin");
  fs.mkdirSync(goBin, { recursive: true });
  const env = envWithToolchainPath(
    { PATH: "/usr/bin:/bin" },
    { home, platform: "linux", env: {} }
  );
  const parts = env.PATH.split(path.delimiter);
  assert.equal(parts[0], goBin);
  assert.ok(parts.includes("/usr/bin"));
});

test("envWithToolchainPath does not duplicate dirs already on PATH", () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "forst-path-"));
  const goBin = path.join(home, "go", "bin");
  fs.mkdirSync(goBin, { recursive: true });
  const before = `${goBin}${path.delimiter}/usr/bin`;
  const env = envWithToolchainPath(
    { PATH: before },
    { home, platform: "linux", env: {} }
  );
  const parts = env.PATH.split(path.delimiter);
  assert.equal(parts.filter((p) => p === goBin).length, 1);
  assert.ok(parts.includes(goBin));
  assert.ok(parts.includes("/usr/bin"));
});

test("resolveGoPathDir accepts directory or go binary path", () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "forst-gopah-"));
  const bin = path.join(home, "custom", "bin");
  fs.mkdirSync(bin, { recursive: true });
  const goBin = path.join(bin, "go");
  fs.writeFileSync(goBin, "#!/bin/sh\n");
  assert.equal(resolveGoPathDir(bin, "linux"), bin);
  assert.equal(resolveGoPathDir(goBin, "linux"), bin);
  assert.equal(resolveGoPathDir("", "linux"), undefined);
  assert.equal(
    resolveGoPathDir("/missing/go", "linux"),
    path.posix.dirname("/missing/go")
  );
});

test("envWithToolchainPath prepends forst.go.path override first", () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "forst-goov-"));
  const override = path.join(home, "override", "bin");
  const defaultGo = path.join(home, "go", "bin");
  fs.mkdirSync(override, { recursive: true });
  fs.mkdirSync(defaultGo, { recursive: true });
  const env = envWithToolchainPath(
    { PATH: `/usr/bin${path.delimiter}${override}` },
    { home, platform: "linux", env: {}, goPath: override }
  );
  const parts = env.PATH.split(path.delimiter);
  assert.equal(parts[0], override);
  assert.equal(parts.filter((p) => p === override).length, 1);
  assert.ok(parts.includes(defaultGo));
});

test("envWithToolchainPath writes Path key on Windows-style env objects", () => {
  // win32 path joins are not real dirs on non-Windows hosts; only run the
  // existence + Path rewrite check on Windows.
  if (process.platform !== "win32") {
    assert.equal(pathEnvKey({ Path: "C:\\Windows\\System32" }), "Path");
    return;
  }
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "forst-winpath-"));
  const goBin = path.win32.join(home, "go", "bin");
  fs.mkdirSync(goBin, { recursive: true });
  const env = envWithToolchainPath(
    { Path: "C:\\Windows\\System32" },
    { home, platform: "win32", env: {} }
  );
  assert.ok(env.Path?.toLowerCase().includes(goBin.toLowerCase()));
  assert.equal(env.PATH, undefined);
});
