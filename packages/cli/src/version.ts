import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

function readPackageJson(): {
  version?: string;
  forst?: { compilerRelease?: string };
} {
  const here = dirname(fileURLToPath(import.meta.url));
  const pkgPath = join(here, "..", "package.json");
  const raw = readFileSync(pkgPath, "utf8");
  return JSON.parse(raw) as {
    version?: string;
    forst?: { compilerRelease?: string };
  };
}

/**
 * Semver of this npm package.
 */
export function getCliPackageVersion(): string {
  const v = readPackageJson().version;
  if (!v || typeof v !== "string") {
    throw new Error("@forst/cli: missing package.json version");
  }
  return v;
}

/**
 * Pinned GitHub compiler release tag (without `v`) for the native binary this
 * package expects — independent of the npm package semver when they diverge.
 * Falls back to npm `version` when `forst.compilerRelease` is unset.
 */
export function getBundledCompilerReleaseVersion(): string {
  const pkg = readPackageJson();
  const pinned = pkg.forst?.compilerRelease?.trim();
  if (pinned) {
    return pinned;
  }
  return getCliPackageVersion();
}
