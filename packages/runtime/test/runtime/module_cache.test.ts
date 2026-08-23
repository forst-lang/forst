import { describe, expect, test, afterEach } from "bun:test";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { fileURLToPath } from "node:url";
import { importModule, clearModuleCache } from "../../src/runtime/module_cache.js";
import { runTestEffect } from "../helpers/run-effect.js";

const testDir = path.dirname(fileURLToPath(import.meta.url));
const fixtureUrl = pathToFileURL(
  path.resolve(testDir, "../fixtures/sync-add.ts")
);

describe("importModule cache", () => {
  afterEach(() => {
    clearModuleCache();
  });

  test("returns the same module instance on repeated import", async () => {
    const first = await runTestEffect(importModule(fixtureUrl));
    const second = await runTestEffect(importModule(fixtureUrl));
    expect(first).toBe(second);
    expect(typeof first.add).toBe("function");
  });

  test("clearModuleCache allows importing again after reset", async () => {
    await runTestEffect(importModule(fixtureUrl));
    clearModuleCache();
    const again = await runTestEffect(importModule(fixtureUrl));
    expect(typeof again.add).toBe("function");
  });
});
