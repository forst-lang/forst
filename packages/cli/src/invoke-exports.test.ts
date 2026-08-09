import { describe, expect, test } from "bun:test";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const pkgRoot = join(import.meta.dir, "..");

describe("@forst/cli/invoke package exports", () => {
  test("package.json declares import and require for ./invoke", () => {
    const pkg = JSON.parse(
      readFileSync(join(pkgRoot, "package.json"), "utf8")
    ) as {
      exports: Record<string, Record<string, string>>;
      dependencies: Record<string, string>;
    };
    const invoke = pkg.exports["./invoke"];
    expect(invoke.import).toBe("./dist/invoke.js");
    expect(invoke.require).toBe("./dist/invoke.js");
    expect(invoke.types).toBe("./dist/invoke.d.ts");
    expect(pkg.exports["."].require).toBe("./dist/index.js");
    expect(pkg.dependencies["@forst/errors"]).toBe(">=0.1.0");
  });

  test("dist/invoke.js and .d.ts exist after build", () => {
    expect(existsSync(join(pkgRoot, "dist/invoke.js"))).toBe(true);
    expect(existsSync(join(pkgRoot, "dist/invoke.d.ts"))).toBe(true);
  });

  test("import barrel exports startForstInvokeServer", async () => {
    const mod = await import("./invoke.js");
    expect(typeof mod.startForstInvokeServer).toBe("function");
    expect(typeof mod.readInvokeReadyUrl).toBe("function");
  });
});
