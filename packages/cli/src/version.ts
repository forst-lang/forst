import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Semver of this npm package — must match the compiler release tag (without `v`)
 * that ships the corresponding binaries.
 */
export function getCliPackageVersion(): string {
  const here = dirname(fileURLToPath(import.meta.url));
  const pkgPath = join(here, "..", "package.json");
  const raw = readFileSync(pkgPath, "utf8");
  const v = JSON.parse(raw) as { version?: string };
  if (!v.version || typeof v.version !== "string") {
    throw new Error("@forst/cli: missing package.json version");
  }
  return v.version;
}
