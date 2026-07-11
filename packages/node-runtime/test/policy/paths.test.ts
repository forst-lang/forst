import { describe, expect, test } from "bun:test";
import {
  matchesExcludePatterns,
  matchesGlobPattern,
  setFilesExcludePatterns,
  validateModuleIdSyntax,
} from "../../src/policy/paths.js";

describe("paths exclude patterns", () => {
  test("matchesGlobPattern supports doublestar segments", () => {
    expect(matchesGlobPattern("**/node_modules/**", "pkg/node_modules/x.ts")).toBe(
      true
    );
    expect(matchesGlobPattern("**/*.skip.ts", "legacy/payment.skip.ts")).toBe(
      true
    );
    expect(matchesGlobPattern("**/*.skip.ts", "legacy/payment.ts")).toBe(false);
  });

  test("validateModuleIdSyntax rejects files.exclude matches from initialize params", () => {
    setFilesExcludePatterns(["**/*.skip.ts", "**/secret/**"]);
    expect(() => validateModuleIdSyntax("legacy/payment.skip.ts")).toThrow(
      /files\.exclude/
    );
    expect(() => validateModuleIdSyntax("secret/payment.ts")).toThrow(
      /files\.exclude/
    );
    expect(() => validateModuleIdSyntax("legacy/payment.ts")).not.toThrow();
  });

  test("matchesExcludePatterns returns false when patterns empty", () => {
    expect(matchesExcludePatterns("legacy/payment.ts", [])).toBe(false);
  });
});
