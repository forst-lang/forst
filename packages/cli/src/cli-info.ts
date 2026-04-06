import { execFileSync, spawnSync } from "node:child_process";
import {
  resolveForstBinary,
  type ResolveForstBinaryOptions,
} from "./resolve.js";
import { getCliPackageVersion } from "./version.js";

/**
 * Prints `@forst/cli` npm version, resolved native binary path, and `forst version` stdout.
 * Intended for diagnostics (`npx forst --forst-cli-info`).
 */
export async function printForstCliInfo(
  resolveOptions?: ResolveForstBinaryOptions
): Promise<void> {
  console.log(`@forst/cli (npm): ${getCliPackageVersion()}`);
  const bin = await resolveForstBinary(resolveOptions);
  console.log(`Compiler binary: ${bin}`);
  const out = execFileSync(bin, ["version"], { encoding: "utf8" }).trim();
  console.log(`Compiler version output:\n${out}`);
}

/**
 * Short version banner for `--version` / `-V`: npm semver plus the first line of `forst version`
 * (no binary path — use `--forst-cli-info` for full diagnostics).
 */
export async function printForstCliVersion(
  resolveOptions?: ResolveForstBinaryOptions
): Promise<void> {
  const npm = getCliPackageVersion();
  const bin = await resolveForstBinary(resolveOptions);
  const full = execFileSync(bin, ["version"], { encoding: "utf8" }).trim();
  const first = full.split(/\r?\n/, 1)[0] ?? full;
  console.log(`@forst/cli ${npm}`);
  console.log(first);
}

/**
 * Runs `go version -m <binary>` on the resolved compiler: prints paths, versions, and hashes embedded
 * at link time (the same metadata `go version` uses). Requires `go` on PATH.
 * (See `go help version` — the `-m` flag lists embedded module information.)
 */
export async function printForstGoBuildInfo(
  resolveOptions?: ResolveForstBinaryOptions
): Promise<void> {
  const bin = await resolveForstBinary(resolveOptions);
  const env = resolveOptions?.env ?? process.env;
  const go = process.platform === "win32" ? "go.exe" : "go";

  const r = spawnSync(go, ["version", "-m", bin], {
    encoding: "utf8",
    env: env as NodeJS.ProcessEnv,
  });

  if (r.error) {
    const code = (r.error as NodeJS.ErrnoException).code;
    if (code === "ENOENT") {
      throw new Error(
        "go: not found on PATH. Install Go from https://go.dev/dl/ to use `go version -m` on the Forst binary."
      );
    }
    throw r.error;
  }

  if (r.status !== 0) {
    const stderr = r.stderr?.trim() ?? "";
    throw new Error(
      stderr || `go version -m exited with code ${r.status ?? "unknown"}`
    );
  }

  console.log(`Embedded build metadata (go version -m):`);
  console.log(r.stdout.trimEnd());
}
