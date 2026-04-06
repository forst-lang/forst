import { execFileSync } from "node:child_process";
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
