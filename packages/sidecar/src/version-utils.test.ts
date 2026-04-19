import { describe, expect, it } from "bun:test";
import { ForstUtils } from "./utils";

describe("parseForstVersionStdout", () => {
  it("parses forst version line", () => {
    expect(
      ForstUtils.parseForstVersionStdout("forst 1.2.3 abcdef 2024-01-01\n")
    ).toBe("1.2.3");
  });

  it("returns empty on bad input", () => {
    expect(ForstUtils.parseForstVersionStdout("nope")).toBe("");
  });

  it("is case-insensitive on forst prefix", () => {
    expect(ForstUtils.parseForstVersionStdout("FORST 9.8.7\n")).toBe("9.8.7");
  });

  it("uses only the first line", () => {
    expect(
      ForstUtils.parseForstVersionStdout("forst 1.0.0 x\nforst 2.0.0 y\n")
    ).toBe("1.0.0");
  });

  it("does not look past the first line", () => {
    expect(ForstUtils.parseForstVersionStdout("hello\nforst 1.0.0\n")).toBe("");
  });
});
