import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test } from "bun:test";
import { getCliPackageVersion } from "./version.js";

test("getCliPackageVersion matches package.json version field", () => {
  const here = dirname(fileURLToPath(import.meta.url));
  const pkgPath = join(here, "..", "package.json");
  const raw = readFileSync(pkgPath, "utf8");
  const expected = (JSON.parse(raw) as { version: string }).version;

  expect(getCliPackageVersion()).toBe(expected);
});
